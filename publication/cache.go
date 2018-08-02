package kubeless

import (
	"time"

	"github.com/gomodule/redigo/redis"
)

type Cacher interface {
	Get(string) ([]byte, error)
	Set(string, []byte, time.Duration) error
	Delete(string) error
	IsExist(string) bool
	ClearAll(string) error
}

type RedisCache struct {
	client *redis.Pool
}

func NewRedisCache(addr string) Cacher {
	c := &redis.Pool{
		MaxIdle:     4,
		IdleTimeout: 180 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr) },
	}
	return &RedisCache{c}
}

func (r *RedisCache) Get(key string) ([]byte, error) {
	c := r.client.Get()
	defer c.Close()
	return redis.Bytes(c.Do("GET", key))
}

func (r *RedisCache) Set(key string, val []byte, t time.Duration) error {
	c := r.client.Get()
	defer c.Close()
	_, err := c.Do("SET", key, val, "EX", int64(t/time.Second))
	return err
}

func (r *RedisCache) Delete(key string) error {
	c := r.client.Get()
	defer c.Close()
	_, err := c.Do("DEL", key)
	return err
}

func (r *RedisCache) IsExist(key string) bool {
	c := r.client.Get()
	defer c.Close()
	v, err := redis.Bool(c.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return v
}

func (r *RedisCache) ClearAll(prefix string) error {
	c := r.client.Get()
	defer c.Close()
	keys, err := redis.String(c.Do("KEYS", prefix+":*"))
	if err != nil {
		return err
	}
	for _, k := range keys {
		if _, err := c.Do("DEL", k); err != nil {
			return err
		}
	}
	return nil
}
