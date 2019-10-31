help: ## display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

init: ## setup development environment
	grep -E "_ \".+\"" ./tool/tool.go | sed -r 's/_ "(.+)"/ \1/' | xargs go get
	go mod tidy

fmt: ## run formatter
	goimports -w $(shell find . -type f -name '*.go')

lint: ## check go lint
	go vet ./...
	$(eval GOLINT := $(shell go list -f {{.Target}} golang.org/x/lint/golint))
	$(GOLINT) ./...

test: ## run go test
	go test ./...

clean: ## clean up go module
	go mod tidy
