# Type System Extension example

https://facebook.github.io/graphql/draft/#sec-Type-System-Extensions

```
$ go run ./server/server.go
2018/10/25 12:46:45 connect to http://localhost:8080/ for GraphQL playground

$ curl -X POST 'http://localhost:8080/query' --data-binary '{"query":"{ todos { id text state verified } }"}'
{"data":{"todos":[{"id":"Todo:1","text":"Buy a cat food","state":"NOT_YET","verified":false},{"id":"Todo:2","text":"Check cat water","state":"DONE","verified":true},{"id":"Todo:3","text":"Check cat meal","state":"DONE","verified":true}]}}
```
