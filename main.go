package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sashabaranov/go-openai"
)

const (
	cacheFile             = "cache/response-cache.json"
	defaultCacheSizeLimit = 10 * 1024 * 1024 // 10MB
)

type CacheEntry struct {
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

type Cache struct {
	Responses map[string]CacheEntry `json:"responses"`
}

type CachingClient struct {
	*openai.Client
	cacheEnabled   bool
	cacheSizeLimit int64
}

func NewCachingClient(apiKey string, cacheEnabled bool, cacheSizeLimit int64) *CachingClient {
	client := openai.NewClient(apiKey)
	return &CachingClient{
		Client:         client,
		cacheEnabled:   cacheEnabled,
		cacheSizeLimit: cacheSizeLimit,
	}
}

func generateHash(req openai.ChatCompletionRequest) (string, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func loadCache() (*Cache, error) {
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return &Cache{Responses: make(map[string]CacheEntry)}, nil
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

func saveCache(cache *Cache) error {
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

func clearCache() error {
	return os.RemoveAll(filepath.Dir(cacheFile))
}

func (c *CachingClient) fetchResponse(ctx context.Context, req openai.ChatCompletionRequest) (string, bool, error) {
	resp, err := c.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", false, err
	}
	return resp.Choices[0].Message.Content, false, nil
}

func (c *CachingClient) getResponse(ctx context.Context, req openai.ChatCompletionRequest) (string, bool, error) {
	if !c.cacheEnabled {
		return c.fetchResponse(ctx, req)
	}

	cache, err := loadCache()
	if err != nil {
		return "", false, err
	}

	hash, err := generateHash(req)
	if err != nil {
		return "", false, err
	}

	if entry, found := cache.Responses[hash]; found {
		entry.Timestamp = time.Now()
		cache.Responses[hash] = entry
		if err := saveCache(cache); err != nil {
			return "", false, err
		}
		return entry.Response, true, nil
	}

	response, _, err := c.fetchResponse(ctx, req)
	if err != nil {
		return "", false, err
	}

	cache.Responses[hash] = CacheEntry{
		Response:  response,
		Timestamp: time.Now(),
	}

	if err := c.evictIfNeeded(cache); err != nil {
		return "", false, err
	}

	if err := saveCache(cache); err != nil {
		return "", false, err
	}

	return response, false, nil
}

func (c *CachingClient) evictIfNeeded(cache *Cache) error {
	cacheSize := int64(0)
	for _, entry := range cache.Responses {
		cacheSize += int64(len(entry.Response))
	}

	if cacheSize <= c.cacheSizeLimit {
		return nil
	}

	// Sort entries by timestamp
	entries := make([]struct {
		Hash      string
		Timestamp time.Time
	}, 0, len(cache.Responses))
	for hash, entry := range cache.Responses {
		entries = append(entries, struct {
			Hash      string
			Timestamp time.Time
		}{hash, entry.Timestamp})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	// Evict least recently used entries
	for cacheSize > c.cacheSizeLimit && len(entries) > 0 {
		oldest := entries[0]
		cacheSize -= int64(len(cache.Responses[oldest.Hash].Response))
		delete(cache.Responses, oldest.Hash)
		entries = entries[1:]
	}

	return nil
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable not set.")
		os.Exit(1)
	}

	cacheEnabled := flag.Bool("cache-requests", false, "Enable caching of requests")
	cacheSizeLimit := flag.Int64("cache-size-limit", defaultCacheSizeLimit, "Cache size limit in bytes")
	flag.Parse()

	client := NewCachingClient(apiKey, *cacheEnabled, *cacheSizeLimit)
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
	maxTokens := 100 // Example max tokens value

	for _, model := range models {
		fmt.Printf("Testing model: %s\n", model)
		for _, prompt := range prompts {
			req := openai.ChatCompletionRequest{
				Model: model,
				Messages: []openai.ChatCompletionMessage{
					{Role: "user", Content: prompt},
				},
				Seed:      &seed,
				MaxTokens: maxTokens,
			}
			response, cached, err := client.getResponse(ctx, req)
			if err != nil {
				fmt.Printf("Error fetching response for prompt '%s': %v\n", prompt, err)
				continue
			}
			if cached {
				fmt.Printf("Cached response for prompt '%s': %s\n", prompt, response)
			} else {
				fmt.Printf("API response for prompt '%s': %s\n", prompt, response)
			}
		}
	}
}
