APP_NAME=scheduler

.PHONY: build run test vet tidy

build:
	go build -o bin/$(APP_NAME).exe ./cmd/scheduler

run:
	go run ./cmd/scheduler

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy
