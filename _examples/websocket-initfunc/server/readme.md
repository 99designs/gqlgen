# WebSocket Init App

Example server app using websocket `InitFunc`.

## Build and Run the server app

First get an update from gqlgen:  
```bash
go mod tidy
go get -u github.com/99designs/gqlgen
```

Next just make the build:  
```bash
make build
```

Run the server:  
```bash
./server 
2022/07/07 16:49:46 connect to http://localhost:8080/ for GraphQL playground
```

You may now implement a websocket client to subscribe for websocket messages.  