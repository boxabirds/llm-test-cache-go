package main

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

var (
	keepCache        = flag.Bool("keep-cache", false, "Keep the cache after tests for manual inspection")
	maxTokens        = flag.Int("max-tokens", 0, "Maximum tokens for the ChatCompletion request")
	testCacheability = flag.Bool("test-cacheability", false, "Test if the API configuration is deterministic")
	cacheSizeLimit   = flag.Int64("cache-size-limit", defaultCacheSizeLimit, "Cache size limit in bytes")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestCacheAPIResponses(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatal("Error: OPENAI_API_KEY environment variable not set. Please set it to run the tests.")
	} else {
		t.Log("Found OPENAI_API_KEY environment variable.")
	}

	client := NewCachingClient(apiKey, true, *cacheSizeLimit)
	ctx := context.Background()

	models := []string{"gpt-3.5-turbo-1106", "gpt-3.5-turbo-0125"}
	prompts := []string{
		"Tell me a joke.",
		"Explain the theory of relativity.",
		"What's the capital of France?",
		"How does a computer work?",
		"What's the meaning of life?",
	}

	seed := 12345

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			for i := 0; i < 3; i++ {
				for _, prompt := range prompts {
					req := openai.ChatCompletionRequest{
						Model: model,
						Messages: []openai.ChatCompletionMessage{
							{Role: "user", Content: prompt},
						},
						Seed:      &seed,
						MaxTokens: *maxTokens,
					}
					response, cached, err := client.getResponse(ctx, req)
					assert.NoError(t, err)
					if i == 0 {
						assert.False(t, cached, "First run should be a cache miss")
					} else {
						assert.True(t, cached, "Subsequent runs should use cache")
					}
					t.Logf("Response for prompt '%s': %s\n", prompt, response)
				}
			}

			if *testCacheability {
				// Test for deterministic responses
				req := openai.ChatCompletionRequest{
					Model: model,
					Messages: []openai.ChatCompletionMessage{
						{Role: "user", Content: prompts[0]},
					},
					Seed:      &seed,
					MaxTokens: *maxTokens,
				}
				response1, _, err := client.getResponse(ctx, req)
				assert.NoError(t, err)
				response2, _, err := client.getResponse(ctx, req)
				assert.NoError(t, err)
				assert.Equal(t, response1, response2, "API / model is not behaving in a cacheable way")
			}
		})
	}

	// Clear cache after tests unless keepCache flag is set
	if !*keepCache {
		err := clearCache()
		assert.NoError(t, err, "Failed to clear cache")
	}
}
