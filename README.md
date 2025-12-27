# NanoLog 🚀

**轻量级、高性能的 Go 原生日志数据库 (The SQLite for Logs)**

[![Docker Pulls](https://img.shields.io/docker/pulls/cofferstech/nanolog.svg)](https://hub.docker.com/r/cofferstech/nanolog)
[![Go Report Card](https://goreportcard.com/badge/github.com/coffersTech/nanolog)](https://goreportcard.com/report/github.com/coffersTech/nanolog)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

NanoLog 是一个转为云原生设计的轻量级日志存储引擎。它不像 Elasticsearch 那样沉重，也不像 Plain Text 那样难以检索。它定位为**日志界的 SQLite**：单二进制文件、极致性能、开箱即用。

## ✨ 核心特性

- 🚀 **极速启动**：单二进制文件，0 运行时依赖，Docker 镜像仅约 20MB。
- 💾 **列式存储**：自研 `.nano` 格式，搭配 ZSTD 压缩，存储成本仅为原始 JSON 的 10%。
- 🔍 **混合查询**：内存 (MemTable) + 磁盘 (Columnar Storage) 混合检索，支持秒级关键词高亮回溯。
- 🎨 **可视化控制台**：内置 Vue 3 仪表盘，支持 Live Tail (实时日志流) 和服务端过滤。
- 🔌 **生态友好**：提供官方 Spring Boot Starter，零配置无感接入。
- 🛡️ **安全停机**：支持优雅停机 (Graceful Shutdown)，确保退出时内存数据 100% 落盘。

## 🚀 快速开始

### 使用 Docker (推荐)

这是最快的体验方式。我们将容器内的 `8080` 端口映射到宿主机，并挂载数据卷以实现持久化。

```bash
# 拉取镜像
docker pull cofferstech/nanolog:latest

# 启动容器
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/root/data \
  --name nanolog \
  cofferstech/nanolog:latest
```

### 使用 Docker Compose

在项目根目录下创建一个 `docker-compose.yml`：

```yaml
version: '3.8'
services:
  nanolog:
    image: cofferstech/nanolog:latest
    container_name: nanolog
    ports:
      - "8080:8080"
    volumes:
      - ./data_docker:/root/data
    restart: unless-stopped
```

然后运行：`docker-compose up -d`

## ⚙️ 配置指南

NanoLog 支持通过命令行参数进行灵活配置：

| 参数 | 说明 | 默认值 |
| :--- | :--- | :--- |
| `--port` | 服务的监听端口 | `8080` |
| `--data` | `.nano` 数据文件存储目录 | `./data` |
| `--retention` | 日志保留时间 (Go duration 格式) | `168h` (7天) |
| `--web` | 静态网页资源目录 | `./web` |

## 🔌 接入指南

### Java / Spring Boot 接入

使用官方提供的 Spring Boot Starter 即可实现零代码接入：

1. **添加依赖**:
```xml
<dependency>
    <groupId>tech.coffers</groupId>
    <artifactId>nanolog-spring-boot-starter</artifactId>
    <version>0.1.0</version>
</dependency>
```

2. **配置配置项** (可选):
```yaml
nanolog:
  server-url: http://localhost:8080
  service: my-service-name
```

### HTTP API 接入

你可以直接通过 HTTP POST 向 NanoLog 推送日志：

**Endpoint**: `POST /api/ingest`

**Payload**:
```json
{
  "timestamp": 1735282400000000000,
  "level": "ERROR",
  "service": "order-service",
  "message": "Payment gateway timeout"
}
```

## 🎨 可视化界面

访问 `http://localhost:8080` 即可进入内置的控制台。

- **Live Tail**: 开启 "Auto Refresh" 实时观测系统脉搏。
- **智能搜索**: 支持 `level=ERROR` 等逻辑语法解析。

## 🛠️ 开发与贡献

这是一个 Monorepo 项目，包含以下模块：

- `server/`: Go 编写的高性能日志引擎。
- `web/`: Vue 3 + Tailwind CSS 编写的前端控制台。
- `sdks/`: 官方支持的各语言 SDK。

**本地开发启动**:
```bash
cd server
go run cmd/nanolog/main.go --data=./test_data
```

---

**NanoLog** - 让日志存储回归简单。 
如果你喜欢这个项目，请给一个 ⭐️ **Star**！
