package minio

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinioRepository_CachingAndSingleflight(t *testing.T) {
	client, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("mockaccess", "mocksecret", ""),
		Secure: false,
		Region: "us-east-1",
	})
	require.NoError(t, err)

	underlying := &Repository{
		client: client,
		bucket: "test-bucket",
		expiry: 5 * time.Second,
	}

	repo := NewCachedRepository(underlying, underlying.Expiry())

	ctx := context.Background()
	key := "test-key"

	// 1. Initial Call (Cache Miss)
	url1, err := repo.GenerateDownloadURL(ctx, key)
	require.NoError(t, err)
	assert.NotEmpty(t, url1)

	// Verify it got cached
	repo.cacheMu.RLock()
	item, found := repo.cache[key]
	repo.cacheMu.RUnlock()
	require.True(t, found)
	assert.Equal(t, url1, item.value)

	// 2. Second Call (Cache Hit)
	// Even though time has passed, it must return the EXACT same URL string from cache
	url2, err := repo.GenerateDownloadURL(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, url1, url2)

	// 3. Force Regeneration (Clear cache and wait 1s)
	// If we clear cache and wait for timestamp to change, it should generate a different URL
	repo.cacheMu.Lock()
	delete(repo.cache, key)
	repo.cacheMu.Unlock()

	time.Sleep(1 * time.Second)

	url3, err := repo.GenerateDownloadURL(ctx, key)
	require.NoError(t, err)
	assert.NotEqual(t, url1, url3) // X-Amz-Date changes, so URL and Signature change

	// 4. Concurrent Calls (Singleflight)
	newKey := "concurrent-key"
	var wg sync.WaitGroup
	results := make([]string, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			url, err := repo.GenerateDownloadURL(ctx, newKey)
			if err == nil {
				results[idx] = url
			}
		}(i)
	}
	wg.Wait()

	// All concurrent requests must receive the exact same URL due to singleflight deduplication
	firstRes := results[0]
	assert.NotEmpty(t, firstRes)
	for _, res := range results {
		assert.Equal(t, firstRes, res)
	}
}

func TestMinioRepository_Cleanup(t *testing.T) {
	repo := &CachedRepository{
		cache: make(map[string]cacheItem),
	}

	now := time.Now()
	repo.cache["valid"] = cacheItem{value: "url1", expiresAt: now.Add(1 * time.Hour)}
	repo.cache["expired"] = cacheItem{value: "url2", expiresAt: now.Add(-1 * time.Minute)}

	repo.cleanupCache()

	repo.cacheMu.RLock()
	defer repo.cacheMu.RUnlock()

	assert.Contains(t, repo.cache, "valid")
	assert.NotContains(t, repo.cache, "expired")
}
