package openai

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestVectorStore(t *testing.T) {
	// Load environment variables from .env file
	if err := godotenv.Load("../.env"); err != nil {
		t.Logf("Warning: No .env file found: %v", err)
	}

	// Create a new OpenAI client
	client := NewClient()

	// Check if API key is set via exported method
	if client.key == "" {
		t.Skip("OpenAI API key is not set. Skipping test.")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test thread ID
	threadID := "test_thread_" + fmt.Sprintf("%d", time.Now().Unix())

	t.Logf("Creating vector store for thread: %s", threadID)

	// Test creating a vector store
	vsID, err := client.EnsureVectorStore(ctx, threadID)
	if err != nil {
		t.Fatalf("Failed to create vector store: %v", err)
	}
	t.Logf("Successfully created vector store: %s", vsID)

	// Create a temporary test file
	testFilePath := "../tmp/test_file.txt"
	os.MkdirAll("../tmp", 0755)
	err = os.WriteFile(testFilePath, []byte("This is a test file for OpenAI API"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	// Test uploading a file
	fileID, err := client.UploadAssistantFile(ctx, threadID, testFilePath)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}
	t.Logf("Successfully uploaded file: %s", fileID)

	// Test adding file to vector store
	err = client.AddFileToVectorStore(ctx, vsID, fileID)
	if err != nil {
		t.Fatalf("Failed to add file to vector store: %v", err)
	}
	t.Logf("Successfully added file to vector store")
}
