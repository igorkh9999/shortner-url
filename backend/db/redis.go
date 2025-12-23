package db

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisDB struct {
	client *redis.Client
}

func NewRedisDB(redisURL string) (*RedisDB, error) {
	var opt *redis.Options
	
	// Try parsing as URL first
	if parsed, err := redis.ParseURL(fmt.Sprintf("redis://%s", redisURL)); err == nil {
		opt = parsed
	} else {
		// Try as simple host:port
		opt = &redis.Options{
			Addr: redisURL,
		}
	}

	// Increase pool size for high concurrency (1000+ concurrent requests)
	// Each redirect request may need: Get (cache) + Set (cache miss) + Incr (counter) = 3 ops
	// Async operations also need connections, so we need more capacity
	opt.PoolSize = 200
	opt.MinIdleConns = 50
	// Set timeouts to prevent hanging connections
	// Balance between fast failure and allowing Redis to respond under load
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 200 * time.Millisecond  // Allow time for Redis to respond
	opt.WriteTimeout = 200 * time.Millisecond // Same for writes
	opt.PoolTimeout = 50 * time.Millisecond   // Fast fail if pool exhausted

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisDB{client: client}, nil
}

func (r *RedisDB) Close() error {
	return r.client.Close()
}

func (r *RedisDB) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get key: %w", err)
	}
	return val, nil
}

func (r *RedisDB) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}
	return nil
}

func (r *RedisDB) Incr(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key: %w", err)
	}
	// Set TTL if this is the first increment (fire-and-forget to avoid blocking)
	if val == 1 {
		go func() {
			// Use background context to avoid cancellation
			bgCtx := context.Background()
			if err := r.client.Expire(bgCtx, key, 60*time.Second).Err(); err != nil {
				// Log error but don't block the request
				// This is non-critical - the key will expire eventually or be cleaned up
			}
		}()
	}
	return val, nil
}

func (r *RedisDB) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}
	return nil
}

func (r *RedisDB) GetInt(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get int: %w", err)
	}
	return val, nil
}

// Ping checks Redis connectivity
func (r *RedisDB) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

