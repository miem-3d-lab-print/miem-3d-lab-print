.PHONY: docs build run test vet

docs:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g src/cmd/app/main.go -o docs --parseInternal --parseDependency

build:
	go build -trimpath -o bin/server ./src/cmd/app

run:
	go run ./src/cmd/app

test:
	go test -race ./...

vet:
	go vet ./...
