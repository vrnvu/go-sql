.PHONY: run test lint

run:
	go run ./cmd/cli/main.go -input ./resources/query_params.csv -workers 4

test:
	go test ./... -count=1 -race

lint:
	golangci-lint run --config .golangci.yml
