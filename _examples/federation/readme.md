### Federation

[Read the docs](https://gqlgen.com/recipes/federation/)

## Testing

If you want to set breakpoints and debug the federation example, you can first run the subgraphs using:

  $ go run ./all/main.go

Then start the gateway using
  $ npm run start-gateway

You can then connect your preferred Golang debugger to the Go process as you make requests to the router.
