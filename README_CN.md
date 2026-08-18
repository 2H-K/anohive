# AnoHive - 实时日志监控与异常检测系统

AnoHive 是一个高性能的实时日志聚合与异常检测系统。它可以从多个来源收集日志，解析多种日志格式，实时检测异常，并提供 Web 监控面板。

## 功能特性

- **多格式日志解析**：JSON、Docker、Kubernetes、Log4j、Apache、Nginx、Syslog、级别化日志、通用文本
- **实时异常检测**：错误率飙升、日志量激增、新错误模式、日志速率变化
- **REST API**：完整的日志摄入和查询 API
- **原始日志摄入**：POST 原始日志行，自动检测格式并解析
- **WebSocket 流**：实时推送日志到连接客户端
- **Web 监控面板**：暗色主题 UI，支持虚拟滚动、实时过滤、WebSocket
- **CLI 工具**：命令行交互工具
- **SQLite 存储**：WAL 模式 + 连接池，支持高并发
- **Prometheus 指标**：/api/metrics 端点用于监控
- **日志保留策略**：基于可配置保留期的自动清理
- **高性能**：在普通硬件上可达 15,000+ 条/秒
- **容器化支持**：Dockerfile + Docker Compose + Kubernetes 部署
- **配置热重载**：无需重启即可更新配置
- **资源监控**：内存/协程压力检测，自动降级

## 架构图

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  日志来源    │────▶│   Collector  │────▶│   Parser    │
└─────────────┘     └──────────────┘     └──────┬──────┘
                                                 │
                    ┌──────────────┐     ┌──────▼──────┐
                    │   Detector   │◀────│  Log Entry  │
                    └──────┬───────┘     └─────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                  ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   SQLite     │  │   WebSocket   │  │  REST API    │
│   Storage    │  │   Broadcast   │  │              │
└──────────────┘  └──────────────┘  └──────────────┘
```

## 快速开始

### 构建

```bash
make build
```

### 运行服务器

```bash
./build/pulse -port 8080
```

### 摄入日志

```bash
# 单条日志
curl -X POST http://localhost:8080/api/logs/ingest \
  -H "Content-Type: application/json" \
  -d '{"source": "myapp", "entries": [{"level": "ERROR", "message": "Connection failed"}]}'

# 批量摄入
curl -X POST http://localhost:8080/api/logs/ingest \
  -H "Content-Type: application/json" \
  -d '{"source": "myapp", "entries": [
    {"level": "INFO", "message": "Server started"},
    {"level": "ERROR", "message": "Database error"},
    {"level": "WARN", "message": "High memory usage"}
  ]}'
```

### 摄入原始日志（自动解析）

```bash
curl -X POST http://localhost:8080/api/logs/raw \
  -H "Content-Type: application/json" \
  -d '{"source": "myapp", "lines": [
    "stdout | 2024-01-15T10:30:00.123456789Z Server started",
    "stderr | 2024-01-15T10:30:01.123456789Z ERROR: Connection failed",
    "2024-01-15 10:30:00.123 ERROR [main] com.example.Service - Failed"
  ]}'
```

### 查询日志

```bash
# 获取最新日志
curl "http://localhost:8080/api/logs?limit=10"

# 按级别过滤
curl "http://localhost:8080/api/logs?level=ERROR"

# 搜索
curl "http://localhost:8080/api/logs?search=database"

# 组合过滤
curl "http://localhost:8080/api/logs?source=myapp&level=ERROR&limit=20"
```

### CLI 使用

```bash
# 检查服务器健康状态
./build/anohive-cli health

# 查看最新日志
./build/anohive-cli logs --level ERROR --limit 10

# 摄入日志
./build/anohive-cli ingest --level ERROR "Something went wrong"

# 查看异常
./build/anohive-cli anomalies --severity CRITICAL

# 实时流式日志
./build/anohive-cli stream

# 备份数据库
./build/anohive-cli backup -db /var/lib/anohive/data/anohive.db -output /tmp/pulse-backup.db

# 恢复数据库
./build/anohive-cli restore -db /var/lib/anohive/data/anohive.db -input /tmp/pulse-backup.db
```

## API 端点

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | /api/health | 健康检查 |
| GET | /api/health/live | K8s 存活探针 |
| GET | /api/health/ready | K8s 就绪探针 |
| GET | /api/v1/logs | 查询日志（支持过滤） |
| POST | /api/v1/logs/ingest | 摄入结构化日志（需认证） |
| POST | /api/v1/logs/raw | 摄入原始日志行（需认证） |
| GET | /api/v1/anomalies | 列出检测到的异常 |
| GET | /api/v1/stats | 系统统计信息 |
| GET | /api/v1/sources | 列出活跃来源 |
| PUT | /api/v1/config/thresholds | 更新检测阈值（需认证） |
| POST | /api/v1/config/reload | 重新加载配置（需认证） |
| GET | /api/v1/admin/resources | 资源使用统计（需认证） |
| GET | /api/metrics | Prometheus 格式指标 |
| GET | /api/metrics/json | JSON 格式指标 |
| WS | /ws | WebSocket 实时流 |

## 配置

### 服务器参数

| 参数 | 默认值 | 描述 |
|------|--------|------|
| -host | 0.0.0.0 | 服务器绑定地址 |
| -port | 8080 | 服务器端口 |
| -db | anohive.db | SQLite 数据库路径 |
| -static | | 静态文件目录 |
| -config | | 配置文件路径 |
| -buffer | 10000 | 采集器缓冲区大小 |

### 环境变量

| 变量 | 默认值 | 描述 |
|------|--------|------|
| PULSE_HOST | 0.0.0.0 | 服务器绑定地址 |
| PULSE_PORT | 8080 | 服务器端口 |
| ANOIVE_DB_PATH | anohive.db | 数据库路径 |
| PULSE_RETENTION_HOURS | 168 | 日志保留小时数（7天） |
| PULSE_MAX_LOGS | 1000000 | 最大日志保留数量 |
| ANOIVE_API_KEY | anohive-dev-key-2024 | 认证 API Key |
| PULSE_LOG_LEVEL | info | 日志级别 (debug/info/warn/error) |
| PULSE_LOG_FORMAT | text | 日志格式 (text/json) |
| PULSE_ALLOWED_ORIGINS | * | 允许的 CORS 来源 |
| PULSE_RATE_LIMIT | 100 | 每分钟每 IP 请求限制 |
| PULSE_WS_MAX_CONNECTIONS | 100 | 最大 WebSocket 连接数 |
| PULSE_ALERT_ENABLED | false | 启用 Webhook 告警 |
| PULSE_ALERT_WEBHOOK_URL | | Webhook 告警 URL |

### 检测阈值

```bash
# 通过 API 更新
curl -X PUT http://localhost:8080/api/config/thresholds \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"error_rate": 0.5, "rate_multiplier": 4.0, "burst": 200}'

# 通过 CLI 更新
./build/anohive-cli config error_rate 0.5
./build/anohive-cli config burst 200
```

## 支持的日志格式

1. **JSON**：结构化日志，包含 level/message 字段
2. **Docker**：容器 stdout/stderr 带时间戳
3. **Kubernetes**：Pod/容器前缀日志
4. **RFC 5424 Syslog**：标准 syslog 带结构化数据
5. **Log4j/Logback**：Java 日志框架格式
6. **Syslog**：传统 BSD syslog 格式
7. **Leveled**：时间戳 + 级别 + 消息
8. **Apache Combined**：完整 Apache 访问日志格式
9. **Nginx**：组合访问日志格式
10. **Generic**：自动检测 ERROR/WARN/DEBUG/FATAL 关键字

## 异常检测类型

1. **错误率飙升 (ERROR_SPIKE)**：错误率超过阈值（默认 30%）
2. **日志量激增 (LOG_BURST)**：短时间内大量日志涌入
3. **新错误模式 (NEW_ERROR_TYPE)**：出现新的错误消息
4. **速率变化 (RATE_CHANGE)**：日志量按倍数增长（默认 3 倍）

## 容器化部署

### Docker Compose

```bash
# 构建并运行
docker-compose up -d

# 查看日志
docker-compose logs -f anohive

# 停止
docker-compose down
```

### Kubernetes

```bash
# 部署所有资源
kubectl apply -k deployments/kubernetes/

# 检查状态
kubectl -n anohive get pods

# 端口转发
kubectl -n anohive port-forward svc/anohive 8080:80
```

## 测试

```bash
# 运行单元测试
make test

# 带覆盖率
go test -cover ./internal/...

# 运行负载测试（需要运行中的服务器）
go test -v ./test/
```

## 项目结构

```
anohive/anohive/
├── cmd/
│   ├── server/        # 服务器入口
│   └── cli/           # CLI 入口
├── internal/
│   ├── api/           # HTTP 处理器 + WebSocket
│   ├── collector/     # 日志采集器
│   ├── config/        # 配置管理
│   ├── detector/      # 异常检测引擎
│   ├── models/        # 数据模型
│   ├── parser/        # 多格式日志解析器
│   ├── runtime/       # 资源监控与降级
│   └── storage/       # SQLite 存储层
├── web/               # React 前端
│   └── src/
│       ├── components/ # React 组件
│       ├── hooks/      # 自定义 Hooks
│       ├── services/   # API 客户端
│       └── styles/     # CSS 样式
├── deployments/       # 部署配置
│   ├── README.md      # 部署文档
│   └── kubernetes/    # K8s 清单
├── docs/              # 文档
│   ├── RUNBOOK.md     # 运维手册
│   └── API.md         # API 文档
├── test/              # 集成和负载测试
├── Dockerfile         # Docker 构建文件
├── docker-compose.yml # Docker Compose 配置
├── .github/workflows/ # CI/CD 工作流
├── Makefile           # 构建自动化
└── README.md
```

## 性能指标

本地测试环境：
- **日志摄入**：15,000+ 条/秒（批量 + SQLite WAL 模式）
- **原始日志摄入**：1,800+ 行/秒（带格式解析）
- **并发操作**：50 并发写入无错误
- **解析器**：500K-1.5M 解析/秒（取决于格式）
- **检测器**：1.7M-3M 条/秒
- **内存占用**：~3.5MB / 23K 条日志

## 许可证

MIT

## 相关链接

- GitHub: https://github.com/2H-K/anohive
- 问题反馈: https://github.com/2H-K/anohive/issues
