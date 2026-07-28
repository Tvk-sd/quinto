BINARY := quinto
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: build test vet fmt release demo giffixture demo-gif clean

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

GIFFIXTURE := .scratch/quinto-v2/giffixtures

# The demo GIF walks the start screen, which needs more than one site
# configured — a throwaway fixture with fabricated names ("blog", "shop"),
# never a real account, reseeded fresh on every recording so it stays
# reproducible.
giffixture: build
	@mkdir -p $(GIFFIXTURE)/config/quinto $(GIFFIXTURE)/data/quinto
	@printf 'site  = blog.example.com\ntoken = fake-not-a-real-token\n\n[shop]\nsite  = shop.example.com\ntoken = fake-not-a-real-token\n' > $(GIFFIXTURE)/config/quinto/config
	XDG_CONFIG_HOME=$(GIFFIXTURE)/config XDG_DATA_HOME=$(GIFFIXTURE)/data ./$(BINARY) demo --db $(GIFFIXTURE)/data/quinto/quinto.db
	XDG_CONFIG_HOME=$(GIFFIXTURE)/config XDG_DATA_HOME=$(GIFFIXTURE)/data ./$(BINARY) demo

# Records the README's demo GIF against the fixture above.
# Requires vhs: brew install vhs
demo-gif: giffixture
	vhs docs/demo.tape

clean:
	rm -rf dist $(BINARY)
