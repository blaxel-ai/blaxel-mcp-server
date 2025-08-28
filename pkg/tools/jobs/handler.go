package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/formatter"
	"github.com/blaxel-ai/toolkit/sdk"
)

// SDKHandler implements JobHandler using the SDK client
type SDKHandler struct {
	sdkClient *sdk.ClientWithResponses
	readOnly  bool
}

// NewSDKHandler creates a new SDK-based job handler
func NewSDKHandler(cfg *config.Config) (JobHandler, error) {
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDK client: %w", err)
	}

	return &SDKHandler{
		sdkClient: sdkClient,
		readOnly:  cfg.ReadOnly,
	}, nil
}

// ListJobs implements JobHandler.ListJobs
func (h *SDKHandler) ListJobs(ctx context.Context, status string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.ListJobsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("list jobs failed with status %d", resp.StatusCode())
	}

	jobs := []sdk.Job{}
	if resp.JSON200 != nil {
		jobs = *resp.JSON200
	}

	// Apply filter (on status in this case)
	if status != "" {
		var filtered []sdk.Job
		for _, job := range jobs {
			if job.Status != nil && strings.EqualFold(*job.Status, status) {
				filtered = append(filtered, job)
			}
		}
		jobs = filtered
	}

	// Format the jobs using the formatter
	formatted := formatter.FormatJobs(jobs)
	return []byte(formatted), nil
}

// GetJob implements JobHandler.GetJob
func (h *SDKHandler) GetJob(ctx context.Context, id string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.GetJobWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get job failed with status %d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("job not found")
	}

	// Convert to JSON for better formatting
	jsonData, err := json.MarshalIndent(*resp.JSON200, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format job data: %w", err)
	}

	return jsonData, nil
}

// DeleteJob implements JobHandler.DeleteJob
func (h *SDKHandler) DeleteJob(ctx context.Context, id string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.DeleteJobWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job: %w", err)
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return nil, fmt.Errorf("delete job failed with status %d", resp.StatusCode())
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Job '%s' deleted successfully", id),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// IsReadOnly implements JobHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}
