.PHONY: setup run test lint

setup:
	go mod download
	go mod verify
	@echo "Installing golangci-lint..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.5.0

run:
	go run ./cmd/cli/main.go -input ./resources/query_params.csv -workers 4 -timeout 10

test:
	go test ./... -count=1 -race -short 

test-slow:
	go test ./... -count=1 -race -v

test-snap:
	UPDATE_SNAPS=true go test ./...

test-cover:
	go test ./... -count=1 -race -covermode=atomic -coverprofile=coverage.out

lint:
	golangci-lint run --config .golangci.yml
