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

# ── Deploy + instalar servicio en Pi (todo en un paso) ────────────────────────
# Compila para ARM6, copia binario, config y scripts al Pi, e instala el servicio.
.PHONY: deploy-pi-service
deploy-pi-service: build-arm6
	ssh $(PI_HOST) "mkdir -p ~/smartworkstation/scripts ~/smartworkstation/systemd"
	scp $(BINDIR)/$(BINARY)-linux-arm6   $(PI_HOST):~/smartworkstation/$(BINARY)
	scp config.toml                       $(PI_HOST):~/smartworkstation/config.toml
	scp scripts/install-service.sh        $(PI_HOST):~/smartworkstation/scripts/
	scp scripts/uninstall-service.sh      $(PI_HOST):~/smartworkstation/scripts/
	scp systemd/$(BINARY).service         $(PI_HOST):~/smartworkstation/systemd/
	ssh $(PI_HOST) "chmod +x ~/smartworkstation/scripts/*.sh"
	ssh $(PI_HOST) "sudo ~/smartworkstation/scripts/install-service.sh \
	    -u \$$(id -un) \
	    -b ~/smartworkstation/$(BINARY) \
	    -c ~/smartworkstation/config.toml"
	@echo "Servicio instalado y arrancado en $(PI_HOST)."
	@echo "Ver logs: ssh $(PI_HOST) 'journalctl -u $(BINARY) -f'"

# ── Instalar servicio systemd en la máquina LOCAL ────────────────────────────
# Útil para instalar en la workstation directamente (sin SSH).
# Usage: make install-service          (usa el binario AMD64 por defecto)
#        make install-service BIN=./bin/smartworkstation-linux-amd64
BIN ?= $(BINDIR)/$(BINARY)-linux-amd64
.PHONY: install-service
install-service: build-amd64
	sudo ./scripts/install-service.sh \
	    -u $$(id -un) \
	    -b $(abspath $(BIN)) \
	    -c $(abspath config.toml)

# ── Desinstalar servicio systemd de la máquina LOCAL ─────────────────────────
.PHONY: uninstall-service
uninstall-service:
	sudo ./scripts/uninstall-service.sh

# ── Ver logs del servicio (máquina local) ────────────────────────────────────
.PHONY: logs
logs:
	journalctl -u $(BINARY) -f

.PHONY: status
status:
	systemctl status $(BINARY)

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
