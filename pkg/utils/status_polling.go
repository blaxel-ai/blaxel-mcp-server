package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/logger"
)

// ResourceType represents the type of resource being polled
type ResourceType string

// StatusChecker defines the interface for checking resource status
type StatusChecker interface {
	GetResource(ctx context.Context, name string) (interface{}, error)
	ExtractStatus(resource interface{}) string
	GetResourceType() ResourceType
}

// isFinalStatus checks if the given status is a final status that indicates
// the resource is no longer in a building/deploying state
func isFinalStatus(status string) bool {
	finalStatuses := []string{
		"DEPLOYED",    // Successfully deployed
		"FAILED",      // Failed to deploy
		"TERMINATED",  // Terminated
		"DEACTIVATED", // Deactivated
		"DELETING",    // Being deleted
	}

	for _, finalStatus := range finalStatuses {
		if status == finalStatus {
			return true
		}
	}
	return false
}

// isBuildingStatus checks if the given status indicates the resource is still building
func isBuildingStatus(status string) bool {
	buildingStatuses := []string{
		"CREATED",      // Just created
		"UPDATED",      // Just updated
		"UPLOADING",    // Uploading
		"BUILDING",     // Building
		"DEPLOYING",    // Deploying
		"DEACTIVATING", // Deactivating
	}

	for _, buildingStatus := range buildingStatuses {
		if status == buildingStatus {
			return true
		}
	}
	return false
}

// WaitForResourceStatus waits for a resource to reach a final status
func WaitForResourceStatus(ctx context.Context, resourceName string, checker StatusChecker) error {
	maxAttempts := 60 // 60 attempts with 2 second intervals = 120 seconds max
	resourceType := checker.GetResourceType()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Get the resource to check its status
		resource, err := checker.GetResource(ctx, resourceName)
		if err != nil {
			logger.Printf("Failed to get %s status (attempt %d/%d): %v", resourceType, attempt, maxAttempts, err)
			if attempt < maxAttempts {
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("failed to get %s status after %d attempts: %w", resourceType, maxAttempts, err)
		}

		if resource == nil {
			logger.Printf("%s not found (attempt %d/%d)", resourceType, attempt, maxAttempts)
			if attempt < maxAttempts {
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("%s not found after %d attempts", resourceType, maxAttempts)
		}

		// Extract status from the resource response
		status := checker.ExtractStatus(resource)

		logger.Printf("%s '%s' status check attempt %d/%d: %s", resourceType, resourceName, attempt, maxAttempts, status)

		if isFinalStatus(status) {
			if status == "DEPLOYED" {
				logger.Printf("%s '%s' successfully deployed", resourceType, resourceName)
				return nil
			} else {
				return fmt.Errorf("%s '%s' reached final status '%s' (not deployed)", resourceType, resourceName, status)
			}
		} else if isBuildingStatus(status) {
			logger.Printf("%s '%s' still building, status: %s", resourceType, resourceName, status)
			if attempt < maxAttempts {
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("%s '%s' did not reach final status within timeout, last status: %s", resourceType, resourceName, status)
		} else {
			logger.Printf("%s '%s' unknown status: %s", resourceType, resourceName, status)
			if attempt < maxAttempts {
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("%s '%s' unknown status after %d attempts: %s", resourceType, resourceName, maxAttempts, status)
		}
	}

	return fmt.Errorf("%s '%s' status check timed out after %d attempts", resourceType, resourceName, maxAttempts)
}

// WaitForResourceDeletion waits for a resource to be fully deleted (404 response)
func WaitForResourceDeletion(ctx context.Context, resourceName string, checker StatusChecker) error {
	maxAttempts := 60 // 60 attempts with 2 second intervals = 120 seconds max
	resourceType := checker.GetResourceType()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Get the resource to check its status
		resource, err := checker.GetResource(ctx, resourceName)
		if err != nil {
			// Check if it's a 404 error, which means the resource is deleted
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
				logger.Printf("%s '%s' successfully deleted (404 response)", resourceType, resourceName)
				return nil
			}
			logger.Printf("Failed to get %s status during deletion (attempt %d/%d): %v", resourceType, attempt, maxAttempts, err)
			if attempt < maxAttempts {
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("failed to get %s status during deletion after %d attempts: %w", resourceType, maxAttempts, err)
		}

		if resource == nil {
			logger.Printf("%s %s not found during deletion (attempt %d/%d)", resourceType, resourceName, attempt, maxAttempts)
			return nil
		}

		// Extract status from the resource response
		status := checker.ExtractStatus(resource)

		logger.Printf("%s '%s' deletion status check attempt %d/%d: %s", resourceType, resourceName, attempt, maxAttempts, status)

		if status == "DELETING" {
			logger.Printf("%s '%s' still being deleted, status: %s", resourceType, resourceName, status)
			if attempt < maxAttempts {
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("%s '%s' still in deleting state after %d attempts", resourceType, resourceName, maxAttempts)
		} else if status == "DELETED" {
			logger.Printf("%s '%s' successfully deleted", resourceType, resourceName)
			return nil
		} else {
			// If the resource is in any other state, it's an error
			return fmt.Errorf("%s '%s' is in unexpected state '%s' during deletion", resourceType, resourceName, status)
		}
	}

	return fmt.Errorf("%s '%s' deletion check timed out after %d attempts", resourceType, resourceName, maxAttempts)
}
