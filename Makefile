.PHONY: build run clean test

BINARY_NAME=llm-proxy

build:
	go build -o $(BINARY_NAME) cmd/proxy/main.go

run: build
	./$(BINARY_NAME) -config config.yaml

clean:
	rm -f $(BINARY_NAME)

test:
	go test ./...

deps:
	go mod tidy

install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/
	sudo mkdir -p /etc/llm-proxy
	sudo cp config.yaml.example /etc/llm-proxy/config.yaml

uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	sudo rm -rf /etc/llm-proxy
