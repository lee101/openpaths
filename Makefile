.PHONY: build test test-go test-frontend lint secrets install-hooks run clean

build:
	go build -o bin/openpaths ./cmd/openpaths

test:
	go test -v -race ./...

test-go:
	go test ./... -count=1

test-frontend:
	npm run lint
	npm run build
	npm run verify:assets

secrets:
	gitleaks git --config .gitleaks.toml --redact --no-banner --exit-code 1

install-hooks:
	./scripts/install-gitleaks-hook.sh

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
