package chat

import (
	"context"
	"time"
)

// AIClient abstracts the OpenAI client for easier mocking in unit tests.
// Only the methods actually used by chat handler are listed.
// We purposely keep parameter types generic (any) for the context to allow *gin.Context
// to be passed directly without pulling in gin dependency here.
type AIClient interface {
	GetAssistantID() string
	CreateThread(ctx context.Context) (string, error)
	StreamMessage(ctx context.Context, prompt string) (<-chan string, error)
	StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error)
	EnsureVectorStore(ctx context.Context, threadID string) (string, error)
	UploadAssistantFile(ctx context.Context, threadID, filePath string) (string, error)
	PollFileProcessed(ctx context.Context, fileID string, timeout time.Duration) error
	AddFileToVectorStore(ctx context.Context, vsID, fileID string) error
	AddSessionBytes(threadID string, delta int64)
	CountThreadFiles(threadID string) int
	GetSessionBytes(threadID string) int64
	TranscribeFile(ctx context.Context, filePath string) (string, error)
	StreamAssistantMessageWithFile(ctx context.Context, threadID, prompt, filePath string) (<-chan string, error)
	// Target a specific vector store (e.g., global books vector or thread's PDF vector)
	StreamAssistantWithSpecificVectorStore(ctx context.Context, threadID, prompt, vectorStoreID string) (<-chan string, error)
	DeleteThreadArtifacts(ctx context.Context, threadID string) error
	// nuevas utilidades de inspección/reset
	ForceNewVectorStore(ctx context.Context, threadID string) (string, error)
	ListVectorStoreFiles(ctx context.Context, threadID string) ([]string, error)
	GetVectorStoreID(threadID string) string
}
