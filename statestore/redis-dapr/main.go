package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	daprHost       string
	daprPort       string
	daprAddr       string
	appPort        string
	stateStoreName string
	daprStateURI   string
)

type Order struct {
	Data struct {
		OrderID string `json:"orderId"`
	} `json:"data"`
}

type State struct {
	Key   string `json:"key"`
	Value Order  `json:"value"`
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get(fmt.Sprintf("%s/order", daprStateURI))
	if err != nil {
		log.Printf("Unable to access to Dapr state: %v\n", err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Unable to read response body: %v\n", err.Error())
	}
	io.WriteString(w, string(body))
}

func postOrder(w http.ResponseWriter, r *http.Request) {
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != io.EOF && err != nil {
		log.Println("Error: Failed to read JSON data ", err)
	}
	log.Printf("Got a new order! Order ID: %s\n", order.Data.OrderID)
	state := State{Key: "order", Value: order}
	input, _ := json.Marshal([]State{state})

	res, err := http.Post(daprStateURI, "application/json", bytes.NewBuffer(input))
	if err != nil {
		log.Printf("Failed to request to Dapr state: %v\n", err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
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
	daprHost = "127.0.0.1"
	daprPort = getEnv("DAPR_HTTP_PORT", "3500")
	daprAddr = fmt.Sprintf("%s:%s", daprHost, daprPort)
	appPort = "8080"
	stateStoreName = "statestore"
	daprStateURI = fmt.Sprintf("http://%s/v1.0/state/%s", daprAddr, stateStoreName)

	http.HandleFunc("/order", getOrder)
	http.HandleFunc("/neworder", postOrder)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", appPort), nil))
}
