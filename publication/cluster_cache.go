package kubeless

import (
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
)

type RedisClusterCache struct {
	client *redisc.Cluster
}

func createPool(addr string, opts ...redis.DialOption) (*redis.Pool, error) {
	return &redis.Pool{
		MaxIdle:     4,
		MaxActive:   6,
		IdleTimeout: 180 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr, opts...) },
	}, nil

}

func NewRedisClusterCache(addrs []string) Cacher {
	c := &redisc.Cluster{
		StartupNodes: addrs,
		CreatePool:   createPool,
	}
	return &RedisClusterCache{client: c}
}

func (r *RedisClusterCache) Get(key string) ([]byte, error) {
	c := r.client.Get()
	defer c.Close()
	return redis.Bytes(c.Do("GET", key))
}

func (r *RedisClusterCache) Set(key string, val []byte, t time.Duration) error {
	c := r.client.Get()
	defer c.Close()
	_, err := c.Do("SET", key, val, "EX", int64(t/time.Second))
	return err
}

func (r *RedisClusterCache) Delete(key string) error {
	c := r.client.Get()
	defer c.Close()
	_, err := c.Do("DEL", key)
	return err
}

func (r *RedisClusterCache) IsExist(key string) bool {
	c := r.client.Get()
	defer c.Close()
	v, err := redis.Bool(c.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return v
}

func (r *RedisClusterCache) ClearAll(prefix string) error {
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
