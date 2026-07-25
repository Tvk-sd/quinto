BINARY := quinto
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: build test vet fmt release demo demo-gif clean

build:
	go build -o $(BINARY) ./cmd/quinto

test:
	go test ./...

vet:
	gofmt -l . && go vet ./...

# Static binaries, no cgo — the reason this project uses SQLite rather than
# DuckDB. Every target below builds from one machine with no toolchain setup.
release:
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		echo "  $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -trimpath -ldflags="-s -w" \
			-o dist/$(BINARY)-$$os-$$arch ./cmd/quinto || exit 1; \
	done
	@ls -lh dist/

demo: build
	./$(BINARY) demo

# Records the README's demo GIF against the sample dataset.
# Requires vhs: brew install vhs
demo-gif: build demo
	vhs docs/demo.tape

clean:
	rm -rf dist $(BINARY)
