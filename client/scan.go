package client

import (
	"context"
	"os"
	"unbx/config"
	"unbx/models"
	"unbx/utils"
)

func BulkScan(ctx context.Context, request models.BulkScanRequest) (*models.BulkScanResponse, error) {
	baseURL := config.UnbxAPIURL
	if override := os.Getenv("UNBX_API_URL"); override != "" {
		baseURL = override
	}
	apiKey := os.Getenv("ACCESS_TOKEN")
	secretKey := os.Getenv("SECRET_TOKEN")
	c := utils.NewUnbxClient(baseURL, apiKey, secretKey)
	return c.BulkScan(ctx, &request)
}
