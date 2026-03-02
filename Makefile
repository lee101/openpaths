.PHONY: build test run clean lint

build:
	go build -o bin/openpaths ./cmd/openpaths

test:
	go test -v -race ./...

run: build
	./bin/openpaths

clean:
	rm -rf bin/

lint:
	go vet ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
