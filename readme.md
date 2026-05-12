# Resilient HTTP Proxy

A lightweight, secure, and resilient local HTTP proxy written in Go. It forwards requests to a backend service with strict timeouts, automatic retries, and robust error handling.

Designed to act as a protective layer between sensitive clients (e.g. authentication modules) and backend HTTP services.

## Features

- **Automatic TLS** — Generates self-signed certificate on first run (valid for 5 years)
- **Configurable Timeouts** — Prevents hanging requests
- **Automatic Retries** — Configurable retry logic on transient failures
- **Environment Variable Configuration** — No config files required
- **Secure by Default** — Runs with HTTPS locally
- **Clean Failure Handling** — Returns proper HTTP responses even when backend is unreachable
- **Lightweight & Reliable** — Single static binary, suitable for production use

## Environment Variables

| Variable           | Description                              | Default                  | Required |
|--------------------|------------------------------------------|--------------------------|----------|
| `PI_TARGET`        | Backend URL to forward requests to       | -                        | Yes      |
| `PI_LISTEN`        | Local address to listen on               | `127.0.0.1:8443`        | No       |
| `PI_TIMEOUT`       | Request timeout duration                 | `15s`                    | No       |
| `PI_RETRIES`       | Number of retries on failure             | `2`                      | No       |
| `PI_INSECURE`      | Skip TLS verification to backend         | `false`                  | No       |
| `PI_CERT`          | Path to certificate file                 | `/etc/resilient-proxy/proxy.crt` | No   |
| `PI_KEY`           | Path to private key file                 | `/etc/resilient-proxy/proxy.key` | No   |

## Installation

### 1. Build from source

```bash
go build -o resilient-proxy resilient-proxy.go
cp resilient-proxy /usr/local/bin/
chmod +x  /usr/local/bin/resilient-proxy
```

### 2. Create directory for certificates

```bash
sudo mkdir -p /etc/resilient-proxy
sudo chown root:root /etc/resilient-proxy
```

### 3. Systemd Service Example

Create `/etc/systemd/system/resilient-proxy.service`:

```ini
[Unit]
Description=Resilient HTTPS Proxy
After=network.target

[Service]
Type=simple
Environment=PI_TARGET=https://backend.example.com
Environment=PI_LISTEN=127.0.0.1:8443
Environment=PI_TIMEOUT=5s
Environment=PI_RETRIES=2
ExecStart=/usr/local/bin/resilient-proxy
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now resilient-proxy
```

## Usage

Once running, the proxy listens on `https://127.0.0.1:8443` (or your configured address) and forwards all requests to the target backend.

Example usage from another application:

```http
https://127.0.0.1:8443/auth/validate   →   forwards to   https://backend.example.com/auth/validate
```

## Security Notes

- Uses **self-signed TLS** for local communication to prevent plaintext sniffing.
- Certificate is automatically generated if not present.
- Only listens on localhost by default.
- Private key permissions are set to `0600`.

---
