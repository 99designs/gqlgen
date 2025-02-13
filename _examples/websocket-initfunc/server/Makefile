bin_name=server

build:
	@echo "building binary..."
# go generate gives missing go sum entry for module errors
# https://github.com/john-markham/gqlgen/issues/1483
# you will need to first do a go get -u github.com/john-markham/gqlgen
	go run -mod=mod github.com/john-markham/gqlgen generate .
	go build -o ${bin_name} server.go
