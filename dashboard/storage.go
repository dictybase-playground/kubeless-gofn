package kubeless

import (
	"github.com/go-redis/redis"
)

type Storage interface {
	Get(string, string) (string, error)
	Set(string, string, string) error
	Delete(string, ...string) error
	IsExist(string, string) bool
}

type RedisStorage struct {
	master *redis.Client
	slave  *redis.Client
}

func NewRedisStorage(master, slave string) Storage {
	return &RedisStorage{
		master: redis.NewClient(&redis.Options{Addr: master}),
		slave:  redis.NewClient(&redis.Options{Addr: slave}),
	}
}

func (r *RedisStorage) Get(key, field string) (string, error) {
	return r.slave.HGet(key, field).Result()
}

func (r *RedisStorage) Set(key, field, val string) error {
	return r.master.HSet(key, field, val).Err()
}

func (r *RedisStorage) Delete(key string, fields ...string) error {
	return r.master.HDel(key, fields...).Err()
}

func (r *RedisStorage) IsExist(key, field string) bool {
	b, err := r.slave.HExists(key, field).Result()
	if err != nil {
		return false
	}
	return b
}

func (r *RedisStorage) ClearAll(prefix string) error {
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
