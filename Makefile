.PHONY: docker-build
docker-build:
	docker build --tag=go-simple-upload-server --file=Dockerfile .

.PHONY: build
build:
	go build -o ./bin/go-simple-upload-server