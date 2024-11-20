package redisstorage

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisStorage struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisStorage(client *redis.Client) *RedisStorage {
	return &RedisStorage{
		client: client,
		ctx:    context.Background(),
	}
}

func (r *RedisStorage) Increment(key string) (int64, error) {
	return r.client.Incr(r.ctx, key).Result()
}

func (r *RedisStorage) Expire(key string, ttl time.Duration) error {
	return r.client.Expire(r.ctx, key, ttl).Err()
}

func (r *RedisStorage) TTL(key string) (time.Duration, error) {
	return r.client.TTL(r.ctx, key).Result()
}

func (r *RedisStorage) GetTokenLimits(token string) (int, time.Duration, error) {
	limit, err := r.client.HGet(r.ctx, "token:"+token, "limit").Int()
	if err != nil {
		return 0, 0, err
	}
	ttl, err := r.client.HGet(r.ctx, "token:"+token, "ttl").Int()
	if err != nil {
		return 0, 0, err
	}
	return limit, time.Duration(ttl) * time.Second, nil
}
