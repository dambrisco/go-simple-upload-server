GO_BUILD_TARGET ?= ./bin/go-simple-upload-server
DOCKER_TAG ?= go-simple-upload-server
DOCKER_PORT ?= 3000
DOCKER_CONTAINER_NAME ?= $(DOCKER_TAG)
APP_PORT ?= 80
TEST_URL ?= "127.0.0.1:$(DOCKER_PORT)/files"
TEST_FILENAME ?= 'testfile'
TEST_PAYLOAD ?= '{"hello":"world"}'
TOKEN ?= "TEST-TOKEN-DO-NOT-USE"

.PHONY: build
build:
	go build -o $(GO_BUILD_TARGET)

.PHONY: test
test:
	go test

.PHONY: docker/build
docker/build:
	docker build --tag=$(DOCKER_TAG) --file=Dockerfile .

.PHONY: docker/run
docker/run: docker/build
	docker run --rm --interactive --tty --publish=127.0.0.1:$(DOCKER_PORT):$(APP_PORT)/tcp $(DOCKER_TAG) /app -server-root=/tmp -port=$(APP_PORT) -token=$(TOKEN)

.PHONY: docker/start
docker/start: docker/build
	docker run --detach --name=$(DOCKER_CONTAINER_NAME) --rm --interactive --tty --publish=127.0.0.1:$(DOCKER_PORT):$(APP_PORT)/tcp $(DOCKER_TAG) /app -server-root=/tmp -port=$(APP_PORT) -token=$(TOKEN)

.PHONY: docker/logs
docker/logs:
	docker logs $(DOCKER_CONTAINER_NAME)

.PHONY: test/put
test/put:
	curl --write-out "\n" --include --header "Authorization: Bearer $(TOKEN)" --request PUT --data $(TEST_PAYLOAD) $(TEST_URL)/$(TEST_FILENAME)

.PHONY: test/post
test/post:
	curl --write-out "\n" --include --header "Authorization: Bearer $(TOKEN)" --request POST --data $(TEST_PAYLOAD) $(TEST_URL)

.PHONY: test/get
test/get:
	curl --write-out "\n" --include --header "Authorization: Bearer $(TOKEN)" --request GET $(TEST_URL)/$(TEST_FILENAME)