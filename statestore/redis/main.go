package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var rdb *redis.Client

type Order struct {
	Data struct {
		OrderID string `json:"orderId"`
	} `json:"data"`
}

type State struct {
	Key   string `json:"key"`
	Value Order  `json:"value"`
}

func initRedisSession() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	pong, err := client.Ping(ctx).Result()
	log.Println(pong, err)
	return client
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	val, err := rdb.Get(ctx, "order").Result()
	if err != nil && err != redis.Nil {
		panic(err)
	}
	io.WriteString(w, val)
}

func postOrder(w http.ResponseWriter, r *http.Request) {
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != io.EOF && err != nil {
		log.Println("Error: Failed to read JSON data ", err)
	}
	log.Printf("Got a new order! Order ID: %s\n", order.Data.OrderID)
	state := State{Key: "order", Value: order}
	input, _ := json.Marshal([]State{state})

	err := rdb.Set(ctx, "order", input, 0).Err()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "Failed to store your request.\n")
		log.Println(err.Error())
	} else {
		log.Println("Successfully persisted state")
		io.WriteString(w, "Succeeded to store your request.\n")
	}
}

func getEnv(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return defaultVal
}

func main() {
	appPort := "8080"
	rdb = initRedisSession()

	http.HandleFunc("/order", getOrder)
	http.HandleFunc("/neworder", postOrder)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", appPort), nil))
}
