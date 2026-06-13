package client

import (
	"context"
	"os"
	"synk/config"
	"synk/models"
	"synk/utils"
)

func BulkScan(ctx context.Context, request models.BulkScanRequest) (*models.BulkScanResponse, error) {
	baseURL := config.SynkAPIURL
	if override := os.Getenv("SYNK_API_URL"); override != "" {
		baseURL = override
	}
	apiKey := os.Getenv("ACCESS_TOKEN")
	secretKey := os.Getenv("SECRET_TOKEN")
	c := utils.NewSynkClient(baseURL, apiKey, secretKey)
	return c.BulkScan(ctx, &request)
}
