package types

import "time"

type Config struct {
	System     ConfigSystem      `yaml:"system"`
	RSSHub     ConfigRSSHubList  `yaml:"rsshub"`
	Translate  *ConfigTranslate  `yaml:"translate,omitempty"`
	ImageProxy *ConfigImageProxy `yaml:"image_proxy,omitempty"`
}

type ConfigSystem struct {
	Debug bool `yaml:"debug"`
	Redis struct {
		URL         string        `yaml:"url"`
		Prefix      string        `yaml:"prefix"`
		CacheExpire time.Duration `yaml:"cache_expire"`
	} `yaml:"redis"`
	Listen         string        `yaml:"listen"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

type ConfigRSSHubList []ConfigRSSHub

type ConfigRSSHub struct {
	URL       string   `yaml:"url"`
	Platforms []string `yaml:"platforms,omitempty"`
	Fallback  bool     `yaml:"fallback"`
}

type ConfigTranslate struct {
	Provider    string `yaml:"provider"`
	DefaultLang string `yaml:"default_lang"`
	Settings    string `yaml:"settings"`
	HostBase    string `yaml:"host_base"`
}

type ConfigImageProxy struct {
	Path  string                          `yaml:"path"`
	Rules map[string]ConfigImageProxyRule `yaml:"rules"`
}

type ConfigImageProxyRule struct {
	Origin  *string `yaml:"origin"`
	Referer *string `yaml:"referer"`
}
