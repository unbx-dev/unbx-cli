package utils

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"synk/models"
	"net/http"
	"strconv"
	"time"
)

type SynkClient struct {
	baseURL    string
	apiKey     string
	secretKey  string
	httpClient *http.Client
}

func NewSynkClient(baseURL, apiKey, secretKey string) *SynkClient {
	return &SynkClient{
		baseURL:   baseURL,
		apiKey:    apiKey,
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// signRequest adds HMAC authentication headers to the request.
// Signature = HMAC-SHA256(api_key + "." + timestamp, secret_key)
func (c *SynkClient) signRequest(req *http.Request) {
	if c.apiKey == "" || c.secretKey == "" {
		return
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	message := c.apiKey + "." + ts
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(message))
	sig := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-Synk-Api-Key", c.apiKey)
	req.Header.Set("X-Synk-Timestamp", ts)
	req.Header.Set("X-Synk-Signature", sig)
}

func (c *SynkClient) BulkScan(ctx context.Context, scanRequest *models.BulkScanRequest) (*models.BulkScanResponse, error) {
	reqBytes, err := json.Marshal(scanRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize: %w", err)
	}
	endpoint := fmt.Sprintf("%s/api/v1/scan/bulk", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.signRequest(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send to server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error. Status code: %d", resp.StatusCode)
	}
	var scanResponse models.BulkScanResponse
	if err := json.NewDecoder(resp.Body).Decode(&scanResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &scanResponse, nil
}
