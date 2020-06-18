package redis

import (
	"fmt"

	"github.com/go-redis/redis/v7"
)

const (
	hostName = "127.0.0.1"
	port     = "6379"
	passWord = ""
	dB       = 0
)

// Connection ...
type Connection struct {
	database int
	rClient  *redis.Client
}

var connection *Connection

// CreateConnection ..
func CreateConnection(database int) *Connection {
	if connection == nil {
		redisClient := redis.NewClient(&redis.Options{
			Addr:     hostName + ":" + port,
			Password: passWord,
			DB:       database,
		})
		connection = &Connection{
			database: database,
			rClient:  redisClient,
		}
	}
	return connection
}

//GetKey ...
func (conn *Connection) GetKey(key string) (string, error) {
	value, err := conn.rClient.Get(key).Result()
	if err == redis.Nil {
		return "", nil
	} else if err != nil {
		return "", err
	} else {
		return value, nil
	}
}

//SetKey ...
func (conn *Connection) SetKey(key string, value interface{}) error {
	fmt.Println(value)
	err := conn.rClient.Set(key, value, 0).Err()
	if err != nil {
		return err
	}
	return nil
}
