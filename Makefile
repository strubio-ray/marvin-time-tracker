.PHONY: build test run clean

build:
	go build -o server/marvin-relay ./server

test:
	go test ./server/...

run: build
	./server/marvin-relay

clean:
	rm -f server/marvin-relay
