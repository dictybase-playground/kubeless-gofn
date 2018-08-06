package kubeless

import (
	"time"

	"github.com/go-redis/redis"
)

type RedisReplicationCache struct {
	master *redis.Client
	slave  *redis.Client
}

func NewRedisReplicationCache(master, slave string) Cacher {
	return &RedisReplicationCache{
		master: redis.NewClient(&redis.Options{Addr: master}),
		slave:  redis.NewClient(&redis.Options{Addr: slave}),
	}
}

func (r *RedisReplicationCache) Get(key string) ([]byte, error) {
	return r.slave.Get(key).Bytes()
}

func (r *RedisReplicationCache) Set(key string, val []byte, t time.Duration) error {
	return r.master.Set(key, val, t).Err()
}

func (r *RedisReplicationCache) Delete(key string) error {
	return r.master.Del(key).Err()
}

func (r *RedisReplicationCache) IsExist(key string) bool {
	rs, err := r.slave.Exists(key).Result()
	if err != nil {
		return false
	}
	if rs == 0 {
		return false
	}
	return true
}

func (r *RedisReplicationCache) ClearAll(prefix string) error {
	iter := r.master.Scan(0, prefix+"*", 0).Iterator()
	for iter.Next() {
		if err := r.master.Del(iter.Val()).Err(); err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return nil
}
