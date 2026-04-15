# SmartWorkstation

**SmartWorkstation** es una aplicación Go que expone un interruptor de corriente virtual en el ecosistema **Tuya / Smart Life** (Alexa, Google Home, automatizaciones) y, al encenderlo, envía un paquete **Wake-on-LAN** para arrancar un Mac (u otro equipo) en la misma red.

Opcionalmente, cuando se ejecuta en una workstation, publica periódicamente métricas del sistema por consola/journal: uso de CPU, RAM, disco y VRAM (NVIDIA, AMD o VideoCore de Raspberry Pi).

El mismo binario —compilado de forma cruzada desde cualquier equipo con Go— funciona en:

| Plataforma | Arquitectura |
|---|---|
| Workstation Linux | AMD64 |
| Raspberry Pi Zero / Zero W | ARMv6 |
| Raspberry Pi 2 / 3 / 4 | ARMv7 |
| Raspberry Pi 3 / 4 / 5 (64-bit OS) | ARM64 |

---

## Tabla de contenidos

- [¿Cómo funciona?](#cómo-funciona)
- [Requisitos previos](#requisitos-previos)
- [Instalación rápida](#instalación-rápida)
- [Compilación](#compilación)
- [Configuración](#configuración)
- [Métricas de sistema](#métricas-de-sistema)
- [Servicio systemd](#servicio-systemd)
- [Estructura del proyecto](#estructura-del-proyecto)
- [Dependencias](#dependencias)
- [Licencia](#licencia)

---

## ¿Cómo funciona?

```
┌─────────────────┐        Pulsar/TLS        ┌──────────────────────┐
│  Tuya Smart app │ ──────────────────────►  │  Tuya Cloud (IoT)    │
│  Alexa / GHome  │  switch_1 → true          │  Virtual device      │
└─────────────────┘                           └──────────┬───────────┘
                                                         │ StatusReport event
                                                         ▼
                                              ┌──────────────────────┐
                                              │  SmartWorkstation    │
                                              │  (Pi Zero W o PC)    │
                                              │                      │
                                              │  ● Escucha eventos   │
                                              │  ● Detecta flanco ↑  │
                                              │  ● Envía WoL UDP ×3  │
                                              └──────────┬───────────┘
                                                         │ Magic packet (UDP broadcast)
                                                         ▼
                                              ┌──────────────────────┐
                                              │  Mac (o cualquier    │
                                              │  equipo con WoL)     │
                                              └──────────────────────┘
```

1. El usuario pulsa el interruptor virtual en la app Tuya Smart (o mediante Alexa/Siri Shortcuts/automatización).
2. Tuya Cloud emite un evento `statusReport` vía Pulsar (protocolo propietario sobre TLS).
3. SmartWorkstation recibe el evento, detecta el flanco OFF → ON y envía N paquetes UDP mágicos al broadcast de la LAN.
4. El Mac recibe el paquete y arranca.
5. Al volver a apagar el interruptor virtual, el estado queda registrado en Tuya (SmartWorkstation no apaga el equipo, solo lo despierta).

---

## Requisitos previos

### Cuenta Tuya Developer

1. Registrarse en [iot.tuya.com](https://iot.tuya.com).
2. Crear un **Cloud Project** con las APIs:
   - *IoT Core*
   - *Device Status Notification*
3. Crear un **Virtual Device** de categoría "Socket" o "Smart Switch" con el DP `switch_1` (Boolean).
4. Vincular el virtual device a la app **Tuya Smart** o **Smart Life**.

La guía detallada paso a paso está en [`docs/tuya-setup.md`](docs/tuya-setup.md).

### Equipo destino (el Mac)

- Activar **Wake for network access** en Preferencias del Sistema → General → Energy Saver.
- Para wake-over-Wi-Fi, el equipo debe estar conectado a corriente y el router debe soportar WoL proxy.

### Herramientas de desarrollo (solo para compilar)

- [Go 1.21+](https://go.dev/dl/)
- `make` (incluido en Xcode CLI tools o en `build-essential`)

---

## Instalación rápida

### Opción A — Raspberry Pi Zero W (recomendada para uso 24/7)

```bash
# 1. Clonar el repositorio en tu máquina de desarrollo
git clone https://github.com/digiogithub/smartworkstation.git
cd smartworkstation

# 2. Crear y rellenar la configuración
cp config.toml.example config.toml
$EDITOR config.toml   # añade tus credenciales Tuya y la MAC del Mac

# 3. Compilar para ARM6, copiar al Pi e instalar el servicio (todo en un paso)
make deploy-pi-service PI_HOST=pi@192.168.1.50

# 4. Verificar que el servicio está activo
ssh pi@192.168.1.50 "journalctl -u smartworkstation -f"
```

### Opción B — Workstation Linux AMD64

```bash
git clone https://github.com/digiogithub/smartworkstation.git
cd smartworkstation
cp config.toml.example config.toml
$EDITOR config.toml

make install-service   # compila, instala y arranca el servicio systemd
make logs              # journalctl -f en tiempo real
```

### Opción C — Ejecutar directamente (sin systemd)

```bash
go run . config.toml
# o con el binario:
./bin/smartworkstation-linux-amd64 config.toml
```

---

## Compilación

El proyecto compila sin CGO (`CGO_ENABLED=0`), lo que permite generar binarios estáticos para cualquier arquitectura Linux desde la misma máquina de desarrollo.

### Targets disponibles

```bash
make build           # host actual (go build nativo)
make build-amd64     # Linux x86-64  → bin/smartworkstation-linux-amd64
make build-arm6      # Pi Zero W      → bin/smartworkstation-linux-arm6
make build-arm7      # Pi 2/3/4 32bit → bin/smartworkstation-linux-arm7
make build-arm64     # Pi 3/4/5 64bit → bin/smartworkstation-linux-arm64
make build-all       # todos los anteriores
```

### Compilación manual

```bash
# AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/smartworkstation .

# Raspberry Pi Zero W (ARMv6)
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o bin/smartworkstation-arm6 .
```

### Despliegue manual al Pi

```bash
# Solo copiar binario y config:
make deploy-pi PI_HOST=pi@192.168.1.50

# Copiar + instalar servicio systemd:
make deploy-pi-service PI_HOST=pi@192.168.1.50
```

---

## Configuración

Copia `config.toml.example` a `config.toml` y rellena los valores:

```toml
[tuya]
client_id     = "abcdef1234567890abcd"         # Access ID del Cloud Project
client_secret = "1234567890abcdef1234567890ab"  # Access Secret
device_id     = "bf3xxxxxxxxxxxxxxxxx"          # ID del virtual device
region        = "eu"                            # eu | us | cn | in
switch_dp     = "switch_1"                      # código DP del interruptor
debug_events  = false                           # true para ver eventos raw

[wol]
mac_address  = "AA:BB:CC:DD:EE:FF"   # MAC del equipo a despertar
broadcast_ip = "192.168.1.255"       # broadcast de tu segmento LAN
port         = 9                     # puerto UDP (7 o 9)
repeat       = 3                     # veces que se envía el paquete

[workstation]
enabled        = false  # true para activar el reporter de métricas
stats_interval = 60     # intervalo en segundos
```

### Obtener la MAC del Mac destino

```bash
networksetup -getmacaddress "Ethernet"  # Ethernet / Thunderbolt
networksetup -getmacaddress "Wi-Fi"     # Wi-Fi
```

### Depuración durante el setup

Activa `debug_events = true` para ver todos los eventos Tuya en el log. Una vez verificado que los eventos llegan correctamente, desactívalo para reducir el ruido.

```
[tuya] statusReport devId=bf3xxx…
[tuya]   dp: code=switch_1 value=true
[wol]  sending magic packet → AA:BB:CC:DD:EE:FF  broadcast=192.168.1.255:9  repeat=3
[wol]  magic packet sent successfully
```

---

## Métricas de sistema

Cuando `workstation.enabled = true`, el proceso registra cada `stats_interval` segundos:

```
[stats] CPU:  14.2%  |  RAM: 9.3 GiB / 32.0 GiB  |  Disk: 245.1 GiB / 1.0 TiB  |  VRAM(nvidia): 3.8 GiB / 8.0 GiB
```

| Métrica | Fuente |
|---------|--------|
| CPU % | `/proc/stat` (1 s de muestreo) |
| RAM | `/proc/meminfo` |
| Disco | `statfs("/")` |
| VRAM — NVIDIA | `nvidia-smi --query-gpu` |
| VRAM — AMD | `/sys/class/drm/card*/device/mem_info_vram_*` |
| VRAM — Raspberry Pi | `vcgencmd get_mem gpu` (total asignada) |

Activa esta opción **únicamente en la workstation**, no en el Pi Zero W (el consumo de CPU por el muestreo de 1 segundo puede ser significativo en un procesador de 700 MHz).

---

## Servicio systemd

### Instalar

```bash
# Workstation local
sudo ./scripts/install-service.sh

# Con rutas explícitas
sudo ./scripts/install-service.sh \
    -u myuser \
    -b /usr/local/bin/smartworkstation \
    -c /etc/smartworkstation/config.toml

# Via Makefile (local)
make install-service
```

### Gestión del servicio

```bash
sudo systemctl start   smartworkstation
sudo systemctl stop    smartworkstation
sudo systemctl restart smartworkstation
sudo systemctl status  smartworkstation

# Logs en tiempo real
journalctl -u smartworkstation -f

# Últimas 50 líneas
journalctl -u smartworkstation -n 50
```

### Desinstalar

```bash
sudo ./scripts/uninstall-service.sh
# o
make uninstall-service
```

El script de desinstalación detiene, deshabilita y elimina el unit file. El binario y `config.toml` no se tocan.

### Política de reinicio

El servicio se reinicia automáticamente si el proceso muere, con 10 segundos de espera entre intentos. Si falla 5 veces en 2 minutos, systemd lo marca como failed y deja de reintentar (evita bucles infinitos ante errores de autenticación). Para reiniciarlo manualmente tras corregir la causa:

```bash
sudo systemctl reset-failed smartworkstation
sudo systemctl start smartworkstation
```

---

## Estructura del proyecto

```
smartworkstation/
├── main.go                          # Punto de entrada, señales OS
├── go.mod / go.sum                  # Módulo Go y dependencias
├── config.toml.example              # Configuración comentada de ejemplo
├── Makefile                         # Targets de compilación, despliegue y servicio
│
├── internal/
│   ├── config/config.go             # Carga y validación del TOML
│   ├── wol/wol.go                   # Magic packet Wake-on-LAN (UDP puro)
│   ├── tuya/client.go               # Conector Tuya Pulsar + dispatch de eventos
│   └── stats/
│       ├── collector.go             # Recolector CPU/RAM/Disco periódico
│       ├── vram_linux.go            # Detección VRAM: NVIDIA, AMD, VideoCore
│       └── vram_other.go            # Stub no-Linux
│
├── systemd/
│   └── smartworkstation.service     # Plantilla del unit file (con hardening)
│
├── scripts/
│   ├── install-service.sh           # Instala y habilita el servicio systemd
│   └── uninstall-service.sh         # Deshabilita y elimina el servicio
│
└── docs/
    └── tuya-setup.md                # Guía completa Tuya Developer + virtual device
```

---

## Dependencias

| Paquete | Versión | Uso |
|---------|---------|-----|
| `github.com/tuya/tuya-connector-go` | v1.0.5 | SDK oficial Tuya: autenticación, Pulsar, eventos tipados |
| `github.com/BurntSushi/toml` | v1.3.2 | Parseo del fichero de configuración |
| `github.com/shirou/gopsutil/v3` | v3.23.12 | Métricas de CPU, RAM y disco multiplataforma |

El resto de dependencias en `go.mod` son transitivas (Pulsar client, logrus, protobuf, etc.).

---

## Licencia

```
MIT License

Copyright (c) 2024 digiogithub

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
