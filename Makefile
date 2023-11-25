BINARY_NAME=loyality-system
 
all: dep test lint build run

build: 
## build: Build project
	go build -o ${BINARY_NAME} .\cmd\gophermart\main.go

test: 
## test: Run project tests
	go test ./... -db="postgres://postgres:6406655@localhost:5432/testdata" -r="localhost:8080"

run: 
## run: Run project
	./${BINARY_NAME}

clean: 
## clean: Cache clean
	go clean
	rm ${BINARY_NAME}

dep: 
## dep: download lib
	go mod download

lint: 
## lint: Run linters
	golangci-lint run -D errcheck

help: 
## help: Show help for each of the Makefile recipes.
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/-/'