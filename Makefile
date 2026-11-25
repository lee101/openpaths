.PHONY: build test test-go test-frontend lint secrets install-hooks run clean sync-openrouter sync-openrouter-check

# github.com/lee101/gobed's cgo int8 kernels hard-code -mavx512* CFLAGS, which
# only compile on amd64. Elsewhere (Apple Silicon, linux/arm64) build without
# cgo so gobed's pure-Go path is used; override with CGO_ENABLED=1 if needed.
ifneq ($(shell go env GOARCH),amd64)
export CGO_ENABLED ?= 0
endif

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

sync-openrouter:
	python3 scripts/sync-openrouter-models.py

sync-openrouter-check:
	python3 scripts/sync-openrouter-models.py --check
