.PHONY: run test lint

run:
	go run ./cmd/cli/main.go -input ./resources/query_params.csv -workers 4 -timeout 10

test:
	go test ./... -count=1 -race -short -v

test-slow:
	go test ./... -count=1 -race -v

test-snap:
	UPDATE_SNAPS=true go test ./...

lint:
	golangci-lint run --config .golangci.yml
