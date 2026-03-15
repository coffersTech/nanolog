<p align="center">
  <img src="web/public/logo.png" width="128" alt="NanoLog Logo">
</p>

# NanoLog 🚀

**Lightweight, high-performance Go-native log database (The SQLite for Logs)**

[![Docker Pulls](https://img.shields.io/docker/pulls/cofferstech/nanolog.svg)](https://hub.docker.com/r/cofferstech/nanolog)
[![Go Report Card](https://goreportcard.com/badge/github.com/coffersTech/nanolog)](https://goreportcard.com/report/github.com/coffersTech/nanolog)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[中文文档](./README.md)

---

NanoLog is a lightweight log storage engine designed specifically for cloud-native environments. It avoids the heaviness of Elasticsearch while providing better search capabilities than plain text files. Positioned as the **SQLite for Logs**, it features a single binary, extreme performance, and a built-in management console.

## ✨ 1.0.0 Key Features

- 🚀 **Lightning Fast**: Single binary, zero runtime dependencies, Docker image size ~20MB.
- 💾 **Columnar Storage**: Custom `.nano` format with ZSTD compression, reducing storage costs to ~10% of raw JSON.
- 🔍 **Hybrid Query**: Combines MemTable and Columnar Storage for second-level logical queries.
- 🎨 **Admin Console**: Embedded Vue 3 dashboard for user management, API key management, and system configuration.
- 🌐 **Multi-language Support**: Native English/Chinese switching for internationalization.
- 🛡️ **Security at Rest**:
    - **Encryption**: Core metadata `.nanolog.sys` is encrypted using AES-GCM.
    - **Emergency Reset**: Built-in CLI tools for password recovery.
    - **RBAC**: Role-based access control with SuperAdmin privileges.
    - **Password Hashing**: Bcrypt-hashed passwords for maximum security.

## 🚀 Quick Start

### Option 1: Using Docker (Recommended)

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/root/data \
  --name nanolog \
  cofferstech/nanolog:latest
```
> [!IMPORTANT]
> By default, `.nanolog.key` and `.nanolog.sys` are stored in the `/root/data` directory. Please ensure this directory is backed up.

### Option 2: Running from Source

```bash
git clone https://github.com/coffersTech/nanolog.git
cd nanolog/server
```

#### run.sh Commands

| Command | Description |
|------|------|
| `./run.sh standalone` | Start full-featured standalone mode ⭐ |
| `./run.sh console` | Start as Console node (requires `--data-nodes`) |
| `./run.sh ingester` | Start as Ingester storage node |
| `./run.sh reset-password` | Emergency password reset tool |
| `./run.sh start` | Compile and start with custom parameters |
| `./run.sh build` | Compile to `bin/nanolog` |
| `./run.sh test` | Run unit tests |
| `./run.sh tidy` | Run go mod tidy |

#### Usage Examples

```bash
# Standalone mode (Recommended for dev/test)
./run.sh standalone --port 8080

# Console mode (Aggregating multiple Ingesters)
./run.sh console --port 8000 --data-nodes=http://localhost:8081,http://localhost:8082

# Ingester mode (Storage node)
./run.sh ingester --port 8081 --data ./data_1
```

## 🛠️ Initialization

1. **System Init**: Visit `http://localhost:8080` after startup. The system will prompt for initialization.
2. **Create Admin**: Set up the first `SuperAdmin` account.
3. **Security Warning**: If the Master Key is auto-generated, a **WARNING** will appear in the logs. Back up `.nanolog.key` immediately.

## ⚙️ Core Configuration

| Parameter | Description | Default |
| :--- | :--- | :--- |
| `--port` | HTTP listening port | `8080` |
| `--data` | Directory for data, keys, and metadata | `./data` |
| `--web` | Directory for static web assets | `../web` |
| `--key` | Manual path to Master Key file | `<data>/.nanolog.key` |
| `--role` | Server role (`standalone`\|`console`\|`ingester`) | `standalone` |
| `--data-nodes` | List of data nodes (console role only) | (empty) |
| `--admin-addr` | Management address for ingester reporting | `localhost:8080` |
| `--retention` | Data retention (e.g., `168h`, `7d`) | `168h` |

## 🔌 Integration Guide (API Auth)

From v0.3.x, all requests to `/api/ingest` must include an API Key in the Header.

### HTTP Integration
**Header**: `Authorization: Bearer <YOUR_API_KEY>`

```bash
curl -X POST http://localhost:8081/api/ingest \
  -H "Authorization: Bearer sk-xxxxxx" \
  -d '{"level":"INFO", "msg":"Hello NanoLog"}'
```

### Java / Spring Boot
1. **Dependency**:
```xml
<dependency>
    <groupId>tech.coffers</groupId>
    <artifactId>nanolog-spring-boot-starter</artifactId>
    <version>0.1.1</version>
</dependency>
```
2. **Configuration**:
```yaml
nanolog:
  server-url: http://localhost:8080
  api-key: sk-xxxxxxx
  service: order-api
```

### Python
1. **Install SDK**:
```bash
pip install nanolog-sdk
```

2. **Usage**:
```python
from nanolog import NanoLogHandler
import logging

logger = logging.getLogger("my_app")
logger.addHandler(NanoLogHandler(
    server_url="http://localhost:8080",
    api_key="sk-xxxx",
    service="my-service"
))

logger.info("Hello from Python")
```

### Go
1. **Get Module**:
```bash
go get github.com/coffersTech/nanolog/sdks/go/nanolog
```

2. **Usage with slog**:
```go
import (
    "log/slog"
    "github.com/coffersTech/nanolog/sdks/go/nanolog"
)

handler := nanolog.NewHandler(nanolog.Options{
    ServerURL: "http://localhost:8080",
    APIKey:    "sk-xxxx",
    Service:   "go-service",
})
logger := slog.New(handler)

logger.Info("Hello from Go")
```

## 🌐 Distributed Deployment

NanoLog supports horizontal scaling with true read-write separation. A single `console` node can manage dozens of `ingester` nodes. Support for high-performance Nginx templates and Keepalive connections is included in 1.0.0.

### Quick Start

```bash
docker-compose -f docker-compose-distributed.yml up -d
```

---

**NanoLog** - Log storage made simple.  
If you like this project, please give us a ⭐️ **Star**!
