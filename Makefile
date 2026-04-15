BINARY   := smartworkstation
MODULE   := github.com/digiogithub/smartworkstation
LDFLAGS  := -ldflags="-s -w"
BINDIR   := bin

# ── Host build ────────────────────────────────────────────────────────────────
.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINDIR)/$(BINARY) .

# ── Linux AMD64 (workstation) ─────────────────────────────────────────────────
.PHONY: build-amd64
build-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build $(LDFLAGS) -o $(BINDIR)/$(BINARY)-linux-amd64 .

# ── Raspberry Pi Zero / Zero W  (ARMv6, single-core 700 MHz) ─────────────────
# Pi Zero W is ARMv6 — use GOARM=6.
# Pi 2/3/4 use GOARCH=arm GOARM=7  or  GOARCH=arm64.
.PHONY: build-arm6
build-arm6:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 \
	go build $(LDFLAGS) -o $(BINDIR)/$(BINARY)-linux-arm6 .

# ── Raspberry Pi 3/4/5  (ARMv7 32-bit) ────────────────────────────────────────
.PHONY: build-arm7
build-arm7:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 \
	go build $(LDFLAGS) -o $(BINDIR)/$(BINARY)-linux-arm7 .

# ── Raspberry Pi 3/4/5  (ARM64) ───────────────────────────────────────────────
.PHONY: build-arm64
build-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
	go build $(LDFLAGS) -o $(BINDIR)/$(BINARY)-linux-arm64 .

# ── Build all targets ─────────────────────────────────────────────────────────
.PHONY: build-all
build-all: build-amd64 build-arm6 build-arm7 build-arm64
	@echo ""
	@echo "Built binaries:"
	@ls -lh $(BINDIR)/

# ── Deploy to Pi Zero W via SSH ───────────────────────────────────────────────
# Usage: make deploy-pi PI_HOST=pi@192.168.1.xxx
PI_HOST ?= pi@raspberrypi.local
.PHONY: deploy-pi
deploy-pi: build-arm6
	ssh $(PI_HOST) "mkdir -p ~/smartworkstation"
	scp $(BINDIR)/$(BINARY)-linux-arm6 $(PI_HOST):~/smartworkstation/$(BINARY)
	scp config.toml $(PI_HOST):~/smartworkstation/config.toml
	@echo "Deployed to $(PI_HOST):~/smartworkstation/"
	@echo "Run: ssh $(PI_HOST) 'cd ~/smartworkstation && ./$(BINARY)'"

# ── Standard Go targets ───────────────────────────────────────────────────────
.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: clean
clean:
	rm -rf $(BINDIR)

$(BINDIR):
	mkdir -p $(BINDIR)

build build-amd64 build-arm6 build-arm7 build-arm64: | $(BINDIR)
