GOPROXY ?= https://goproxy.cn,direct
export GOPROXY

.PHONY: build run test-mcp deps clean

build:
	go build -o my_lobster .

run: build
	./my_lobster

test-mcp:
	go run ./cmd/mcptest/

deps:
	pip3 install mcp_weather_server
	go mod tidy

clean:
	rm -f my_lobster
