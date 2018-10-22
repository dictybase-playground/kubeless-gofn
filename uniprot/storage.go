package kubeless

import (
	"github.com/go-redis/redis"
)

// Storage interface is for manging key value data
type Storage interface {
	Get(string, string) (string, error)
	Set(string, string, string) error
	Delete(string, ...string) error
	IsExist(string, string) bool
	Close() error
}

type redisStorage struct {
	master *redis.Client
	slave  *redis.Client
}

// NewRedisStorage is the constructor for redis for
// storing hash based key value
func NewRedisStorage(master, slave string) Storage {
	return &redisStorage{
		master: redis.NewClient(&redis.Options{Addr: master}),
		slave:  redis.NewClient(&redis.Options{Addr: slave}),
	}
}

// Close closes the redis connection
func (r *redisStorage) Close() error {
	if err := r.master.Close(); err != nil {
		return err
	}
	if err := r.slave.Close(); err != nil {
		return err
	}
	return nil
}

// Get fetches the value of a hash field
func (r *redisStorage) Get(key, field string) (string, error) {
	return r.slave.HGet(key, field).Result()
}

// Sets set the value of a hash field
func (r *redisStorage) Set(key, field, val string) error {
	return r.master.HSet(key, field, val).Err()
}

// Delete deletes one or more hash fields
func (r *redisStorage) Delete(key string, fields ...string) error {
	return r.master.HDel(key, fields...).Err()
}

// IsExist determine if a hash field exist
func (r *redisStorage) IsExist(key, field string) bool {
	b, err := r.slave.HExists(key, field).Result()
	if err != nil {
		return false
	}
	return b
}

// CleaAll remove all keys based based on a prefix
func (r *redisStorage) ClearAll(prefix string) error {
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
