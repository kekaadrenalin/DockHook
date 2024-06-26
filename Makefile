.PHONY: clean
clean:
	@go clean -i

.PHONY: test
test:
	go test -cover -race ./...

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags "-s -w"

.PHONY: docker
docker:
	@docker build -t kekaadrenalin/dockhook .
