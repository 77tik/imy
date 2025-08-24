package utils

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// DefaultCache returns the global default Cache.
func DefaultCache() *cache.Cache {
	return defaultCache
}

var defaultCache = NewCache()

func NewCache() *cache.Cache {
	return cache.New(5*time.Minute, 10*time.Minute)
}
