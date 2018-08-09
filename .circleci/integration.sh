#!/bin/sh

set -eu

function cleanup {
  echo "Cleaning up containers..."
  docker kill $SERVER_CONTAINER 1>/dev/null 2>/dev/null || true
  docker rm --force -v $SERVER_CONTAINER 1>/dev/null 2>/dev/null || true
}
trap cleanup EXIT

SERVER_CONTAINER=$(docker run -d \
    -e PORT=1234 \
    --name integration_server \
    gqlgen/golang go run ./integration/server/server.go \
)

sleep 2

docker run \
    -e SERVER_URL=http://integration_server:1234/query \
    --link=integration_server \
    gqlgen/node ./node_modules/.bin/jest

echo "### server logs"
docker logs $SERVER_CONTAINER

exit $(docker inspect $SERVER_CONTAINER --format='{{.State.ExitCode}}')


