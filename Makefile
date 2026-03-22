.PHONY: build test test-integration lint run install clean

BINARY := termagit
BIN_DIR := bin
CMD_DIR := cmd/termagit

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) ./$(CMD_DIR)

test:
	go test -short -race ./...

test-integration:
	go test -race ./...

lint:
	golangci-lint run ./...

run: build
	./$(BIN_DIR)/$(BINARY)

install: build
	cp $(BIN_DIR)/$(BINARY) $(GOPATH)/bin/

clean:
	rm -rf $(BIN_DIR)
