FROM golang:1.10

RUN curl -L -o /bin/dep https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 && chmod +x /bin/dep
RUN go get -u github.com/alecthomas/gometalinter github.com/vektah/gorunpkg
RUN gometalinter --install

WORKDIR /go/src/github.com/99designs/gqlgen

COPY Gopkg.* /go/src/github.com/99designs/gqlgen/
RUN dep ensure -v --vendor-only

COPY . /go/src/github.com/99designs/gqlgen/
