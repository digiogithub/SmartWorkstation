# Tuya Developer Setup Guide

This guide walks you through everything needed to make SmartWorkstation appear
as a **virtual smart switch** in the Tuya / Smart Life ecosystem so you can
toggle it from the app, Alexa, Google Home, or automations.

---

## 1. Create a Tuya Developer Account

1. Go to **<https://iot.tuya.com>** and click **Sign Up**.
2. Fill in your details and verify your e-mail.
3. Select your **data center region** (EU / US / CN / IN).  
   This determines the API endpoints and **must match** `tuya.region` in
   `config.toml`.

---

## 2. Create a Cloud Project

1. In the Tuya IoT Platform, open **Cloud → Development → Create Cloud Project**.
2. Fill in:
   | Field | Recommended value |
   |---|---|
   | Project Name | SmartWorkstation |
   | Industry | Smart Home |
   | Development Method | Custom |
   | Data Center | Same region you chose in step 1 |
3. Click **Create**.
4. On the next screen (**Authorize API Services**), add:
   - **IoT Core** (required for device status + MQTT)
   - **Device Status Notification** (required for real-time events)
   - **Authorization Token Management**
5. Click **Go to Authorize** and confirm.

---

## 3. Note Your Access ID and Access Secret

1. Open **Cloud → Development → (your project) → Overview**.
2. Copy **Access ID** → `config.toml` → `tuya.client_id`
3. Click the eye icon next to **Access Secret** → copy it → `tuya.client_secret`

---

## 4. Create a Virtual Smart Switch Device

Tuya does not expose a one-click "virtual device" for Cloud projects by
default. Use the **Device Simulation** feature:

### 4a. Create a Product (once)

1. Go to **Product → Create Product**.
2. Category: **Electrical** → **Socket** (or search "Smart Switch").
3. Connection type: **Wi-Fi**.
4. Product name: e.g. `SW-WoL-Virtual`.
5. Click **Create Product**.
6. In the **Function Definition** tab, confirm there is a Boolean DP named
   `switch_1` (code). If it is absent, add it:
   - Click **Add Function** → Standard Function → `switch_1` → Boolean → Save.
7. Skip hardware-related steps; click **Exit** or **Return to project**.

### 4b. Create a Virtual Device

1. Back in your Cloud Project, open **Devices → Virtual Device**.
2. Click **Add Virtual Device**.
3. Select the product you just created (`SW-WoL-Virtual`).
4. Click **Confirm** — the platform creates the device and shows its
   **Device ID** (a 20-22 character alphanumeric string like `bf3abc…`).
5. Copy the Device ID → `config.toml` → `tuya.device_id`.

> **Tip**: If the **Virtual Device** menu item is missing, go to
> **Cloud → Project → (project) → Service Management** and enable
> **Device Simulation**.

---

## 5. Link the Virtual Device to the Smart Life App

You need to add the virtual device to your Tuya / Smart Life mobile app so you
can toggle it.

1. In the Tuya IoT Platform, go to **Link Devices → Add Device to App Account**.
2. Enter your Tuya Smart / Smart Life app account e-mail and confirm.
3. The virtual device now appears in your Smart Life app under **Me → Devices**.

Alternatively, use the **App → Add Device → Manually Add** flow with the
virtual device's QR code from the platform.

---

## 6. Populate config.toml

```toml
[tuya]
client_id     = "abcdef1234567890abcd"        # Access ID
client_secret = "1234567890abcdef1234567890ab" # Access Secret
device_id     = "bf3xxxxxxxxxxxxxxxxx"         # Virtual device ID
region        = "eu"                           # eu | us | cn | in
switch_dp     = "switch_1"
debug_events  = false

[wol]
mac_address  = "AA:BB:CC:DD:EE:FF"   # Target Mac's MAC address
broadcast_ip = "192.168.1.255"       # Your subnet's broadcast
port         = 9
repeat       = 3

[workstation]
enabled        = false   # true only on the stats-reporting machine
stats_interval = 60
```

---

## 7. Find the Target Mac's MAC Address

On the Mac you want to wake:

```bash
# Built-in Ethernet / Thunderbolt adapter
networksetup -getmacaddress "Ethernet"

# Wi-Fi (Wi-Fi WoL may need the router's ARP table trick)
networksetup -getmacaddress "Wi-Fi"
```

Make sure **Wake for network access** is enabled:
**System Settings → General → Energy Saver → Wake for network access** ✓

For wake-over-Wi-Fi, the Mac must be connected to power and the router must
support **WoL proxy** (most modern AirPort / Apple routers do).

---

## 8. Build and Run

### On the workstation (AMD64)

```bash
make build-amd64
cp config.toml.example config.toml
# edit config.toml
./bin/smartworkstation-linux-amd64
```

### On Raspberry Pi Zero W

Cross-compile from a development machine:

```bash
make build-arm6
# copy binary + config to the Pi
make deploy-pi PI_HOST=pi@192.168.1.50
```

Or build directly on the Pi (requires Go ≥ 1.21 installed):

```bash
go build -o smartworkstation .
```

### Run as a systemd service on the Pi

```ini
# /etc/systemd/system/smartworkstation.service
[Unit]
Description=SmartWorkstation WoL Bridge
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/home/pi/smartworkstation/smartworkstation /home/pi/smartworkstation/config.toml
WorkingDirectory=/home/pi/smartworkstation
Restart=on-failure
RestartSec=10
User=pi

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now smartworkstation
sudo journalctl -u smartworkstation -f
```

---

## 9. Test the Integration

1. Start `smartworkstation` with `debug_events = true` in `config.toml`.
2. Open the Smart Life app and tap the virtual switch **ON**.
3. You should see in the log:
   ```
   [tuya] raw event: {"devId":"bf3xxx…","bizCode":"statusReport","bizData":…}
   [tuya] switch switch_1 → true
   [wol]  sending magic packet to AA:BB:CC:DD:EE:FF via 192.168.1.255:9 (×3)
   [wol]  magic packet sent successfully — MAC AA:BB:CC:DD:EE:FF
   ```
4. The target Mac should power on within a few seconds.
5. Once working, set `debug_events = false` to reduce log noise.

---

## 10. Workstation Stats Mode

When `workstation.enabled = true`, the process logs system metrics every
`stats_interval` seconds:

```
[stats] CPU:  12.3%  |  RAM: 8.1 GiB / 32.0 GiB  |  Disk: 120.5 GiB / 1.0 TiB  |  VRAM(nvidia): 4.0 GiB / 8.0 GiB
```

| Metric | Source |
|--------|--------|
| CPU % | `/proc/stat` via gopsutil |
| RAM | `/proc/meminfo` via gopsutil |
| Disk | `statfs("/")` via gopsutil |
| VRAM — NVIDIA | `nvidia-smi` (must be in `$PATH`) |
| VRAM — AMD | `/sys/class/drm/card*/device/mem_info_vram_*` |
| VRAM — Pi | `vcgencmd get_mem gpu` (shows allocated, not used) |

Enable this on the workstation only; it adds ~1 s overhead per 60 s cycle
(CPU measurement requires a 1-second sample window).

---

## Troubleshooting

| Symptom | Likely cause |
|---------|-------------|
| `401 Unauthorized` on startup | Wrong `client_id` / `client_secret` or wrong `region` |
| No events received after toggle | Device not linked to Cloud Project, or wrong `device_id` |
| Events arrive but `switch_dp` not matched | DP code differs — enable `debug_events` to see raw payload |
| WoL packet sent but Mac doesn't wake | Mac "Wake for network access" is off, or wrong broadcast IP |
| Build fails for ARM6 | Ensure Go ≥ 1.21 installed; `CGO_ENABLED=0` is required |
