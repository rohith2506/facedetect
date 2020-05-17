package redis

import (
	"errors"

	"github.com/go-redis/redis/v7"
)

type redisParams struct {
	hostName string
	port     string
	passWord string
	DB       int
}

var redisClient *redis.Client

// CreateConnection ..
func CreateConnection() *redis.Client {
	if redisClient == nil {
		rParams := redisParams{
			hostName: "localhost",
			port:     "6379",
			passWord: "",
			DB:       0,
		}
		redisClient = redis.NewClient(&redis.Options{
			Addr:     rParams.hostName + ":" + rParams.port,
			Password: rParams.passWord,
			DB:       rParams.DB,
		})
	}
	return redisClient
}

func (rClient *redis.Client) GetKey(key string) (string, error) {
	value, err := rClient.Get(key).Result()
	if err == redis.Nil {
		return "", errors.New("Key is not present")
	} else if err != nil {
		return "", err
	} else {
		return value, nil
	}
}

func (rClient *redis.Client) SetKey(key string, value string) error {
	err := rClient.Set(key, value, 0).Err()
	if err != nil {
		return err
	}
	return nil
}
