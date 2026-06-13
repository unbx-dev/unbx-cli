package client

import (
	"context"
	"synk/models"
	"synk/utils"
	"os"
)

func BulkScan(ctx context.Context, request models.BulkScanRequest) (*models.BulkScanResponse, error) {
	baseURL := os.Getenv("SYNK_API_URL")
	apiKey := os.Getenv("ACCESS_TOKEN")
	secretKey := os.Getenv("SECRET_TOKEN")
	c := utils.NewSynkClient(baseURL, apiKey, secretKey)
	return c.BulkScan(ctx, &request)
}
