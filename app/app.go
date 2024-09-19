package app

import (
	"fmt"
	"net/http"

	"github.com/candinya/rsshub-smart-layer/modules"
	"github.com/candinya/rsshub-smart-layer/modules/translate"
	"github.com/candinya/rsshub-smart-layer/modules/translate/providers"
	"github.com/candinya/rsshub-smart-layer/types"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type app struct {
	cfg *types.Config

	l     *zap.Logger
	redis *redis.Client

	lb *modules.LoadBalancer
	tp translate.Provider
	ip *modules.ImageProxy

	e *echo.Echo
}

func Start(cfg *types.Config) error {
	a := app{
		cfg: cfg,
	}

	var err error

	// Initialize logger
	if cfg.System.Debug {
		a.l, err = zap.NewDevelopment()
	} else {
		a.l, err = zap.NewProduction()
	}
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize redis
	redisOpts, err := redis.ParseURL(cfg.System.Redis.URL)
	if err != nil {
		return fmt.Errorf("failed to parse redis url: %w", err)
	}
	a.redis = redis.NewClient(redisOpts)

	// Initialize load balancer
	a.lb, err = modules.NewLoadBalancer(cfg.RSSHub, cfg.System.RequestTimeout, a.l)
	if err != nil {
		return fmt.Errorf("failed to initialize load balancer: %w", err)
	}

	// Initialize translator
	if cfg.Translate != nil {
		a.tp, err = providers.NewTranslator(cfg.Translate, a.l)
		if err != nil {
			return fmt.Errorf("failed to initialize translator: %w", err)
		}
	}

	// Initialize image proxy
	if cfg.ImageProxy != nil {
		a.ip = modules.NewImageProxy(cfg.ImageProxy, a.l)
	}

	// Initialize echo
	a.e = echo.New()

	// Set logger
	a.e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			a.l.Info("request",
				zap.String("URI", v.URI),
				zap.Int("status", v.Status),
			)

			return nil
		},
	}))

	// Add panic recover
	a.e.Use(middleware.Recover())

	// Apply health check route (root)
	a.e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "RSSHub Smart Layer is running")
	})

	// Apply main route
	a.e.GET("/:platform/*", a.process)

	// Apply image proxy route
	if a.ip != nil {
		a.e.GET(a.ip.Path(), a.ip.Proxy)
	}

	return a.e.Start(cfg.System.Listen)
}
