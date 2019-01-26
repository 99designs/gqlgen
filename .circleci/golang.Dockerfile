FROM golang:1.11

RUN curl -sL --fail https://github.com/golangci/golangci-lint/releases/download/v1.13/golangci-lint-1.13-linux-amd64.tar.gz | tar zxv --strip-components=1 --dir=/go/bin

WORKDIR /projects/gqlgen

COPY go.* /projects/gqlgen/
RUN go mod download

COPY . /projects/gqlgen/
