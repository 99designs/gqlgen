### Federation

[Read the docs](https://gqlgen.com/recipes/federation/)

## Testing

If you want to set breakpoints and debug the federation example, you can first run the subgraphs using:

  $ go run ./all/main.go

Then start the gateway using
  $ npm run start-gateway

You can then connect your preferred Golang debugger to the Go process as you make requests to the router.

## Federation v1

`v1/` holds three Federation v1 subgraphs (`accounts`, `content`, `progress`)
served by `go run ./v1`. `start.sh` starts them with the other subgraphs, and the
gateway composes them into a second supergraph on port 4010.
`src/__test__/federation-v1.spec.ts` sends operations through that gateway.
`v1/fetches_test.go` replays the fetches the Apollo query planner sends the
subgraphs, so `go test` covers them without Node. The subgraph schemas match
gqlparser's `testdata/federation/subgraphs`.
