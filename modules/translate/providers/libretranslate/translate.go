package libretranslate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

type libreTranslateRequestBody struct {
	Q      string  `json:"q"`
	Source string  `json:"source"`
	Target string  `json:"target"`
	Format *string `json:"format,omitempty"`
	APIKey string  `json:"api_key"`
}

type libreTranslateResponseBody struct {
	TranslatedText string `json:"translatedText"`
}

var htmlFormat = "html" // Use as constant

func (t *lt) Translate(src string, lang string, isHTML bool) (*string, error) {
	// Prepare request body
	reqBody := &libreTranslateRequestBody{
		Q:      src,
		Source: "auto", // Auto detect
		Target: lang,   // Specified by request
	}

	if isHTML {
		reqBody.Format = &htmlFormat
	}

	if t.key != nil {
		reqBody.APIKey = *t.key
	}

	t.l.Debug("translate request", zap.Any("body", reqBody))

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", t.url, bytes.NewReader(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	defer res.Body.Close()

	var resBody libreTranslateResponseBody
	err = json.NewDecoder(res.Body).Decode(&resBody)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	t.l.Debug("translate response", zap.Any("body", resBody))

	// Return translated result
	return &resBody.TranslatedText, nil
}
