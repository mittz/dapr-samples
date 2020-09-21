# Hello World

This tutorial will demonstrate how to get Dapr running locally on your machine. We'll be deploying a Go app (goapp) that subscribes to order messages and persists them. The following architecture diagram illustrates the components that make up the first part sample:

![Architecture Diagram](./img/Architecture_Diagram.png)

Later on, we'll deploy a different Go app (publisher) to act as the publisher. The architecture diagram below shows the addition of the new component:

![Architecture Diagram Final](./img/Architecture_Diagram_B.png)

## Prerequisites
This sample requires you to have the following installed on your machine:
- [Docker](https://docs.docker.com/)
- [Go 1.14.x or later](https://golang.org/dl/)

## Step 1 - Setup Dapr

Follow [instructions](https://github.com/dapr/docs/blob/master/getting-started/environment-setup.md#environment-setup) to download and install the Dapr CLI and initialize Dapr.

## Step 2 - Understand the Code

Now that we've locally set up Dapr, clone the repo, then navigate to the Hello World sample:

```bash
git clone https://github.com/mittz/dapr-samples.git
cd dapr-samples/statestore/redis-dapr
```


In the `app.go` you'll find a simple `net/http` application, which exposes a few routes and handlers. First, let's take a look at the top of the main method:

```go
daprHost = "127.0.0.1"
daprPort = getEnv("DAPR_HTTP_PORT", "3500")
daprAddr = fmt.Sprintf("%s:%s", daprHost, daprPort)
appPort = "8080"
stateStoreName = "statestore"
daprStateURI = fmt.Sprintf("http://%s/v1.0/state/%s", daprAddr, stateStoreName)
```
When we use the Dapr CLI, it creates an environment variable for the Dapr port, which defaults to 3500. We'll be using this in step 3 when we POST messages to our system. The `stateStoreName` is the name given to the state store. We'll come back to that later on to see how that name is configured.

Next, let's take a look at the ```postOrder``` method for the `neworder` handler:

```go
func postOrder(w http.ResponseWriter, r *http.Request) {
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != io.EOF && err != nil {
		log.Println("Error: Failed to read JSON data ", err)
	}
	state := State{Key: "order", Value: order}
	input, _ := json.Marshal([]State{state})
	log.Println(string(input))

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
		io.WriteString(w, "Succeeded to store your request.\n")
	}
}
```

Here we're exposing an endpoint that will receive and handle `neworder` messages. We first log the incoming message, and then persist the order ID to our Redis store by posting a state array to the `/state/<state-store-name>` endpoint.

Alternatively, we could have persisted our state by simply returning it with our response object:

```go
state := State{Key: "order", Value: order}
input, _ := json.Marshal([]State{state})
```

We chose to avoid this approach, as it doesn't allow us to verify if our message successfully persisted.

We also expose a GET endpoint, `/order`:

```go
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
```

This calls out to our Redis cache to grab the latest value of the "order" key, which effectively allows our Go app (goapp) to be _stateless_.

> **Note**: If we only expected to have a single instance of the Go app (goapp), and didn't expect anything else to update "order", we instead could have kept a local version of our order state and returned that (reducing a call to our Redis store). We would then create a `/state` POST endpoint, which would allow Dapr to initialize our app's state when it starts up. In that case, our Go app (goapp) would be _stateful_.

## Step 3 - Run the Go App (goapp) with Dapr

1. Run Go app (goapp) with Dapr:

    ```sh
    dapr run --app-id goapp --app-port 8080 --port 3500 go run app.go
    ```

The command should output text that looks like the following, along with logs:

```
Starting Dapr with id goapp. HTTP Port: 3500. gRPC Port: 9165
You're up and running! Both Dapr and your app logs will appear here.
...
```
> **Note**: the `--app-port` (the port the app runs on) is configurable. Our Go app (goapp) happens to run on port 8080, but we could configure it to run on any other port. Also note that the Dapr `--port` parameter is optional, and if not supplied, a random available port is used.

The `dapr run` command looks for a `components` directory which holds yaml definition files for components Dapr will be using at runtime. When running locally, if the directory is not found it is created with yaml files which provide default definitions for a local development environment (learn more about this flow [here](https://github.com/dapr/docs/blob/master/walkthroughs/daprrun.md)). Review the `statestore.yaml` file in the `components` directory:

```yml
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.redis
...
```

We can see the yaml file defined the state store to be Redis and is naming it `statestore`. This is the name which was used in `app.go` to make the call to the state store in our application:

```go
stateStoreName = "statestore"
daprStateURI = fmt.Sprintf("http://%s/v1.0/state/%s", daprAddr, stateStoreName)
```

While in this sample we used the default yaml files, usually a developer would modify them or create custom yaml definitions depending on the application and scenario.

## Step 4 - Post Messages to your Service

Now that Dapr and our Go app (goapp) are running, let's POST messages against it, using different tools. **Note**: here we're POSTing against port 3500 - if you used a different port, be sure to update your URL accordingly.

First, let's POST the message by using Dapr cli in a new command line terminal:

Windows Command Prompt
```sh
dapr invoke --app-id goapp --method neworder --payload "{\"data\": { \"orderId\": \"41\" } }"
```

Windows PowerShell
```sh
dapr invoke --app-id goapp --method neworder --payload '{\"data\": { \"orderId\": \"41\" } }'
```

Linux or MacOS
```sh
dapr invoke --app-id goapp --method neworder --payload '{"data": { "orderId": "41" } }'
```

Now, we can also do this using `curl` with:

```sh
curl -XPOST -d @sample.json -H "Content-Type:application/json" http://localhost:3500/v1.0/invoke/goapp/method/neworder
```

Or, we can also do this using the Visual Studio Code [Rest Client Plugin](https://marketplace.visualstudio.com/items?itemName=humao.rest-client)

[sample.http](sample.http)
```http
POST http://localhost:3500/v1.0/invoke/goapp/method/neworder

{
  "data": {
    "orderId": "42"
  }
}
```

Last but not least, we can use the Postman GUI. For more details on how to test this with Postman, please refer to [dapr/samples](https://github.com/dapr/samples/blob/master/1.hello-world/README.md)

## Step 5 - Confirm Successful Persistence

Now, let's just make sure that our order was successfully persisted to our state store. Create a GET request against: `http://localhost:3500/v1.0/invoke/goapp/method/order`. **Note**: Again, be sure to reflect the right port if you chose a port other than 3500.

```sh
curl http://localhost:3500/v1.0/invoke/goapp/method/order // Request through Dapr
curl http://localhost:8080/order // Request through the app interface
```

or using the Visual Studio Code [Rest Client Plugin](https://marketplace.visualstudio.com/items?itemName=humao.rest-client)

[sample.http](sample.http)
```http
GET http://localhost:3500/v1.0/invoke/goapp/method/order
```

This invokes the `/order` route, which calls out to our Redis store for the latest data. Observe the expected result!