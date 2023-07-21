GO_BUILD_TARGET ?= ./bin/go-simple-upload-server
DOCKER_TAG ?= go-simple-upload-server
DOCKER_PORT ?= 3000
APP_PORT ?= 80

.PHONY: build
build:
	go build -o $(GO_BUILD_TARGET)

.PHONY: docker/build
docker/build:
	docker build --tag=$(DOCKER_TAG) --file=Dockerfile .

.PHONY: docker/run
docker/run: docker/build
	docker run --rm --interactive --tty --publish=127.0.0.1:$(DOCKER_PORT):$(APP_PORT)/tcp $(DOCKER_TAG) /app -server-root=/tmp -port=$(APP_PORT)

.PHONY: docker/start
docker/start: docker/build
	docker run --rm --interactive --tty --publish=127.0.0.1:$(DOCKER_PORT):$(APP_PORT)/tcp --detach $(DOCKER_TAG) /app -server-root=/tmp -port=$(APP_PORT)