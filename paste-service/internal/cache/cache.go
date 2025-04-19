package cache

import "time"

type Cache interface {
	Get(key string) (interface{}, bool)
	GetTyped(key string, result interface{}) bool
	Set(key string, value interface{}, ttl time.Duration)
	Invalidate(key string)
	Clear()
}
