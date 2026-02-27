.PHONY: build test run clean lint

build:
	go build -o bin/openpath ./cmd/openpath

test:
	go test -v -race ./...

run: build
	./bin/openpath

clean:
	rm -rf bin/

lint:
	go vet ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
