#!/bin/sh

function cleanup {
  echo "Cleaning up containers..."
  docker kill $SERVER_CONTAINER 1>/dev/null 2>/dev/null || true
  docker rm --force -v $SERVER_CONTAINER 1>/dev/null 2>/dev/null || true
}
trap cleanup EXIT

SERVER_CONTAINER=$(docker run -d gqlgen/golang go run ./integration/server/server.go)

sleep 2


echo "### server logs"
docker logs $SERVER_CONTAINER

exit $(docker inspect $SERVER_CONTAINER --format='{{.State.ExitCode}}')


