BINARY_NAME=loyality-system
 
all: dep test lint build run
 
build:
	go build -o ${BINARY_NAME} .\cmd\gophermart\main.go
 
test:
	go test ./... -db="postgres://postgres:6406655@localhost:5432/testdata" -r="localhost:8080"
 
run:
	./${BINARY_NAME}
 
clean:
	go clean
	rm ${BINARY_NAME}

dep:
	go mod download

lint:
	golangci-lint run -D errcheck