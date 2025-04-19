package cache

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisInstances     []*RedisCache
	redisInstancesLock sync.Mutex
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

func (c *RedisCache) GetTyped(key string, result interface{}) bool {
	val, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Ошибка получениия из Redis для ключа %s: %v", key, err)
		}
		return false
	}

	if err := json.Unmarshal([]byte(val), result); err != nil {
		log.Printf("Ошибка десериализации JSON для ключа %s: %v", key, err)
		return false
	}

	return true
}

func NewRedisCache(redisURL string) (*RedisCache, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(options)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Printf("Успешное подключениие к Redis: %s", redisURL)

	cache := &RedisCache{
		client: client,
		ctx:    ctx,
	}

	redisInstancesLock.Lock()
	redisInstances = append(redisInstances, cache)
	redisInstancesLock.Unlock()

	return cache, nil
}

func (c *RedisCache) Get(key string) (interface{}, bool) {
	val, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Ошибка получения из Redis для ключа %s: %v", key, err)
		}
		return nil, false
	}

	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return val, true
	}

	return result, true
}

func (c *RedisCache) Set(key string, value interface{}, ttl time.Duration) {
	var data string
	switch v := value.(type) {
	case string:
		data = v
	default:
		jsonData, err := json.Marshal(value)
		if err != nil {
			log.Printf("Ошибка сериализации в JSON для ключа %s: %v", key, err)
			return
		}
		data = string(jsonData)
	}

	err := c.client.Set(c.ctx, key, data, ttl).Err()
	if err != nil {
		log.Printf("Ошибка установки в Redis для ключа %s: %v", key, err)
	}
}

func (c *RedisCache) Invalidate(key string) {
	err := c.client.Del(c.ctx, key).Err()
	if err != nil {
		log.Printf("Ошибка удаления из Redis для ключа %s: %v", key, err)
	}
}

func (c *RedisCache) Clear() {
	err := c.client.FlushAll(c.ctx).Err()
	if err != nil {
		log.Printf("Ошибка очистки Redis: %v", err)
	} else {
		log.Println("Redis кэш полностью очищен")
	}
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func CloseRedisConnections() {
	redisInstancesLock.Lock()
	defer redisInstancesLock.Unlock()

	for _, instance := range redisInstances {
		if err := instance.Close(); err != nil {
			log.Printf("Ошибка при закрытии соединения с Redis: %v", err)
		}
	}

	redisInstances = nil
}
