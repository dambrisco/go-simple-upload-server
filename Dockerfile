FROM golang:1.20 AS build

RUN mkdir -p /go/src/app
WORKDIR /go/src/app

# resolve dependency before copying whole source code
COPY go.mod go.sum ./
RUN go mod download

# copy other sources & build
COPY . /go/src/app
RUN make build GO_BUILD_TARGET=/go/bin/app

FROM ubuntu:latest AS run
COPY --from=build /go/bin/app /app

CMD ["/app"]