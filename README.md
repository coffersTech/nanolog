# NanoLog 🚀

**轻量级、高性能的 Go 原生日志数据库 (The SQLite for Logs)**

[![Docker Pulls](https://img.shields.io/docker/pulls/cofferstech/nanolog.svg)](https://hub.docker.com/r/cofferstech/nanolog)
[![Go Report Card](https://goreportcard.com/badge/github.com/coffersTech/nanolog)](https://goreportcard.com/report/github.com/coffersTech/nanolog)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

NanoLog 是一个专为云原生设计的轻量级日志存储引擎。它不像 Elasticsearch 那样沉重，也不像 Plain Text 那样难以检索。它定位为**日志界的 SQLite**：单二进制文件、极致性能、内置管理面板。

## ✨ v0.3.x 核心特性

- 🚀 **极速启动**：单二进制文件，0 运行时依赖，Docker 镜像仅约 20MB。
- 💾 **列式存储**：自研 `.nano` 格式，搭配 ZSTD 压缩，存储成本仅为原始 JSON 的 10%。
- 🔍 **混合查询**：内存 (MemTable) + 磁盘 (Columnar Storage) 混合检索，支持秒级逻辑查询。
- 🎨 **管理面板**：内嵌式 Vue 3 控制台，支持用户管理、API 密钥管理及系统配置。
- 🛡️ **安全加固 (Security at Rest)**：
    - **静态加密**：核心元数据 `.nanolog.sys` 采用 AES-GCM 算法强制加密。
    - **密钥隔离**：支持环境变量、外部文件或自动生成 Master Key (`.nanolog.key`)。
    - **RBAC 权限**：内置角色访问控制，SuperAdmin 专属管理权限。
    - **Bcrypt 散列**：用户密码采用 Bcrypt 强散列，杜绝明文存储。

## 🚀 快速开始

### 方案一：使用 Docker (推荐)

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/root/data \
  --name nanolog \
  cofferstech/nanolog:latest
```
> [!IMPORTANT]
> 默认情况下，`.nanolog.key` (密钥) 与 `.nanolog.sys` (加密数据) 都会保存在绑定的 `/root/data` 目录下。请务必妥善备份该目录。

### 方案二：源码运行 (使用快捷脚本)

```bash
git clone https://github.com/coffersTech/nanolog.git
cd nanolog/server

# 直接启动（默认端口 8080，数据存放在 ../data）
./run.sh start

# 带参数启动
./run.sh start --port 9000 --data ./my-data
```

## 🛠️ 初始化与登录

1. **系统初始化**: 启动后访问 `http://localhost:8080`，系统会提示进入初始化模式。
2. **创建管理员**: 设置第一个 `SuperAdmin` 账号。系统会加密保存并锁定。
3. **安全提示**: 如果密钥是自动生成的，控制台会打印醒目的 **WARNING**。请及时备份 `.nanolog.key`。

## ⚙️ 核心配置参数

| 参数 | 说明 | 默认值 |
| :--- | :--- | :--- |
| `--port` | 服务的监听端口 | `8080` |
| `--data` | 数据文件、密钥及元数据存储目录 | `./data` |
| `--web` | 静态网页资源目录 | `../web` |
| `--key` | 手动指定 Master Key 文件路径 | `<data>/.nanolog.key` |
| `--retention` | 数据保留时长 (例如 `168h`, `7d`) | `168h` |

## 🔌 接入指南 (API Auth)

从 v0.3.x 开始，任何向 `/api/ingest` 推送数据的请求都必须在 Header 中携带 API Key。

### HTTP 接入
**Header**: `Authorization: Bearer <YOUR_API_KEY>`

```bash
curl -X POST http://localhost:8080/api/ingest \
  -H "Authorization: Bearer sk-xxxxxx" \
  -d '{"level":"INFO", "msg":"Hello NanoLog"}'
```

### Java / Spring Boot 接入
1. **添加依赖**:
```xml
<dependency>
    <groupId>tech.coffers</groupId>
    <artifactId>nanolog-spring-boot-starter</artifactId>
    <version>0.1.1</version>
</dependency>
```
2. **配置配置项**:
```yaml
nanolog:
  server-url: http://localhost:8080
  api-key: sk-xxxxxxx
  service: order-api
```

---

**NanoLog** - 让日志存储回归简单。  
如果你喜欢这个项目，请给一个 ⭐️ **Star**！
