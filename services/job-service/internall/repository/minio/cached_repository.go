package minio

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type cacheItem struct {
	value     string
	expiresAt time.Time
}

// repository defines the interface for the operations we want to cache.
type repository interface {
	GenerateUploadURL(ctx context.Context, objectKey string) (string, error)
	GenerateDownloadURL(ctx context.Context, objectKey string) (string, error)
	CopyObject(ctx context.Context, srcKey, dstKey string) error
}

// CachedRepository is a decorator for repository that adds in-memory caching and singleflight.
type CachedRepository struct {
	repo   repository
	expiry time.Duration

	cacheMu sync.RWMutex
	cache   map[string]cacheItem
	sfGroup singleflight.Group
}

func NewCachedRepository(repo repository, expiry time.Duration) *CachedRepository {
	cachedRepo := &CachedRepository{
		repo:   repo,
		expiry: expiry,
		cache:  make(map[string]cacheItem),
	}

	// Start a background ticker to evict expired items periodically
	go cachedRepo.startCleanupWorker(1 * time.Minute)

	return cachedRepo
}

func (r *CachedRepository) startCleanupWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		r.cleanupCache()
	}
}

func (r *CachedRepository) cleanupCache() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	now := time.Now()
	for k, item := range r.cache {
		if now.After(item.expiresAt) {
			delete(r.cache, k)
		}
	}
}

func (r *CachedRepository) GenerateUploadURL(ctx context.Context, objectKey string) (string, error) {
	return r.repo.GenerateUploadURL(ctx, objectKey)
}

func (r *CachedRepository) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	return r.repo.CopyObject(ctx, srcKey, dstKey)
}

func (r *CachedRepository) GenerateDownloadURL(ctx context.Context, objectKey string) (string, error) {
	// 1. Check cache under read lock
	r.cacheMu.RLock()
	item, found := r.cache[objectKey]
	if found && time.Now().Before(item.expiresAt) {
		r.cacheMu.RUnlock()
		return item.value, nil
	}
	r.cacheMu.RUnlock()

	// 2. Cache miss: use singleflight to deduplicate concurrent signatures for the same objectKey
	val, err, _ := r.sfGroup.Do(objectKey, func() (interface{}, error) {
		// Double check under RLock inside singleflight, as another goroutine might have already written it
		r.cacheMu.RLock()
		item, found := r.cache[objectKey]
		if found && time.Now().Before(item.expiresAt) {
			r.cacheMu.RUnlock()
			return item.value, nil
		}
		r.cacheMu.RUnlock()

		// Generate the URL via the underlying repository
		urlStr, err := r.repo.GenerateDownloadURL(ctx, objectKey)
		if err != nil {
			return "", err
		}

		// Calculate safety TTL. E.g. cache it for 2/3 of the pre-signed link lifetime (safety buffer)
		safetyDuration := (r.expiry * 2) / 3
		if safetyDuration <= 0 {
			safetyDuration = r.expiry / 2
		}
		if safetyDuration <= 0 {
			safetyDuration = time.Minute
		}

		expiresAt := time.Now().Add(safetyDuration)

		// Save to cache
		r.cacheMu.Lock()
		r.cache[objectKey] = cacheItem{
			value:     urlStr,
			expiresAt: expiresAt,
		}
		r.cacheMu.Unlock()

		return urlStr, nil
	})

	if err != nil {
		return "", err
	}

	return val.(string), nil
}
