# Chat App

Example app using subscriptions to build a chat room.

### Server

```bash
go run ./server/server.go
```

### Client

The react app uses two different implementation for the websocket link

- [apollo-link-ws](https://www.apollographql.com/docs/react/api/link/apollo-link-ws) which uses the deprecated [subscriptions-transport-ws](https://github.com/apollographql/subscriptions-transport-ws) library
- [graphql-ws](https://github.com/enisdenjo/graphql-ws)

First you need to install the dependencies

```bash
npm install
```

Then to run the app with the `apollo-link-ws` implementation do

```bash
npm run start
```

or to run the app with the `graphql-ws` implementation (and the newer `graphql-transport-ws` protocol) do

```bash
npm run start:graphql-transport-ws
```
