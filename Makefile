.PHONY: setup run test lint docker-up docker-down docker-logs docker-shell

setup:
	go mod download
	go mod verify
	@echo "Installing golangci-lint..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.5.0

run:
	@echo "Running: go run ./cmd/cli/main.go -input ./resources/query_params.csv -workers 4 -timeout 10"
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

docker-up:
	@echo "Starting TigerData database..."
	docker-compose up -d
	@echo "TigerData is running on localhost:5432"

docker-down:
	@echo "Stopping TigerData..."
	docker-compose down

docker-logs:
	docker-compose logs -f tigerdata

docker-shell:
	docker-compose exec tigerdata psql -U tigerdata -d homework
