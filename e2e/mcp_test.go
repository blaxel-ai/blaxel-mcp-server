package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

// TestMain runs before all tests and loads environment variables
func TestMain(m *testing.M) {
	// Load .env file from the project root (parent directory of e2e)
	envPath := filepath.Join("..", ".env")
	if err := godotenv.Load(envPath); err != nil {
		// It's okay if .env doesn't exist, we might use actual env vars
		fmt.Printf("Note: Could not load .env file from %s: %v\n", envPath, err)
	}

	os.Exit(m.Run())
}
