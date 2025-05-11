package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
	"log"
)

var Client *redis.Client
var Ctx = context.Background()

func InitRedis() {
	Client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err := Client.Ping(Ctx).Result()
	if err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}
}
