# Gotify 安全功能完整部署指南

## 📈 功能概览

本文档描述 Gotify 的所有安全功能及其部署配置，包括：
- ✅ 分层速率限制（4层架构）
- ✅ 认证失败黑名单机制
- ✅ XSS 防护（Markdown清理）
- ✅ 强制修改默认密码
- ✅ 安全响应头

---

## 🎯 安全特性详解

### 1. 分层速率限制（Core Protection）

Gotify 实现了 **4层独立速率限制**，从宽松到严格：

| 级别 | 目标端点 | 默认限制 | 生产建议（低/中/高流量） |
|------|----------|----------|-------------------------------------------|
| **Level 1** | 所有请求 | 20 req/s (burst 50) | 10/20/50, 15/40/100 |
| **Level 2** | 认证API | 10 req/s (burst 20) | 5/10/20, 10/20/40 |
| **Level 3** | 消息发送 | 15 req/s (burst 30) | 10/15/30, 20/30/60 |
| **Level 4** | 管理API | 5 req/s (burst 10) | 2/3/5, 3/5/10 |

#### 端点映射

```
Level 1: 所有请求
  → GET /version
  → GET /health
  → GET /swagger
  → 静态资源请求

Level 2: /client/*, /application/*, /plugin/*, /user/*
  → GET    /application
  → POST   /application
  → DELETE /application/:id
  → GET    /client
  → POST/   /client
  → DELETE /client/:id
  → PUT    /client/:id
  → GET/   /message
  → DELETE /message
  → DELETE /message/:id

Level 3: POST /message (Application Token)

Level 4: /user/* (Admin Token)
  → GET    /user
  → DELETE /user/:id
  → GET    /user/:id
  → POST    /user/:id
```

### 2. 认证失败黑名单（Brute Force Protection）

**工作原理：**

```
请求失败 → 记录IP → 计数
         ↓
5分钟窗口内 >= 5次失败
         ↓
拉黑该IP 1小时
         ↓
返回 HTTP 403 + Retry-After头
```

**配置参数：**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `maxfailures` | 5 | 失败次数阈值 |
| `windowseconds` | 300 | 时间窗口（秒，默认5分钟）|
| `blockduration` | 3600 | 拉黑时长（秒，默认1小时）|
| `cleanupinterval` | 300 | 清理过期条目（秒）|
| `whitelist` | 127.0.1, 192.168.1.0/24 | 白名单IP/CIDR |

**白名单支持：**
- 单个IP: `127.0.0.1`
- CIDR网段: `192.168.1.0/24`
- IPv6 CIDR: `2001:db8::/32`

⚠️ **白名单IP不受任何速率限制或黑名单限制！**

### 3. 其他安全特性

#### XSS 防护
- 位置：`ui/src/common/Markdown.tsx`
- 技术：`rehype-sanitize@^6.0.0`
- 作用：清理所有 Markdown 内容中的 HTML

#### 安全响应头
```go
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Strict-Transport-Security: max-age=31536000; includeSubDomains (启用SSL时)
```

#### 强制修改密码
- 新安装用户默认 `MustChangePassword: true`
- API 返回 `mustChangePassword` 字段
- 首次登录后强制修改密码

---

---

## 🔧 Docker环境变量配置示例

```yaml
# 修改docker-compose.yml中的environment部分

environment:
  # Level 1: 全局限制（宽松）
  GOTIFY_SERVER_RATELIMIT_GLOBAL_ENABLED: "true"
  GOTIFY_SERVER_RATELIMIT_GLOBAL_REQUESTSPERSECOND: "20"
  GOTIFY_SERVER_RATELIMIT_GLOBAL_BURST: "50"

  # Level 2: 认证API（中等）
  GOTIFY_SERVER_RATELIMIT_AUTH_ENABLED: "true"
  GOTIFY_SERVER_RATELIMIT_AUTH_REQUESTSPERSECOND: "10"
  GOTIFY_SERVER_RATELIMIT_AUTH_BURST: "20"

  # Level 3: 消息发送（较高）
  GOTIFY_SERVER_RATELIMIT_MESSAGE_ENABLED: "true"
  GOTIFY_SERVER_RATELIMIT_MESSAGE_REQUESTSPERSECOND: "15"
  GOTIFY_SERVER_RATELIMIT_MESSAGE_BURST: "30"

  # Level 4: 管理API（严格）
  GOTIFY_SERVER_RATELIMIT_ADMIN_ENABLED: "true"
  GOTIFY_SERVER_RATELIMIT_ADMIN_REQUESTSPERSECOND: "5"
  GOTIFY_SERVER_RATELIMIT_ADMIN_BURST: "10"
```

---

## 📋 验证速率限制

### 测试方法1: curl命令
```bash
# 快速发送多个请求测试速率限制
for i in {1..55}; do
  curl -I http://localhost:80/version
done

# 当超过限制时，会返回 HTTP 429
```

### 测试方法2: 重启服务查看配置
```bash
# 启动服务
docker-compose up -d

# 查看日志确认配置加载
docker-compose logs gotify | grep -i "rate"
```

---

## 🔍 配置建议

### 低流量环境
- Global: 10 req/s, burst 20
- Auth: 5 req/s, burst 10
- Message: 8 req/s, burst 15
- Admin: 3 req/s, burst 5

### 高流量环境（负载均衡后端）
- Global: 50 req/s, burst 100
- Auth: 20 req/s, burst 40
- Message: 30 req/s, burst 60
- Admin: 10 req/s, burst 20

### 严格安全环境
- Global: 10 req/s, burst 20
- Auth: 5 req/s, burst 10
- Message: 5 req/s, burst 10
- Admin: 2 req/s, burst 5

---

## ⚠️ 注意事项

1. **向后兼容性**：
   - 旧的单一配置模式已废弃
   - 请更新配置文件或环境变量

2. **性能影响**：
   - 速率限制会轻微增加CPU开销
   - 内存使用量增加（每个IP需存储状态）

3. **WebSocket**：
   - `/stream` 端点仍使用Level 2（Auth）限制
   - 连接建立后不受限制

4. **负载均衡**：
   - 每个Gotify实例独立限制
   - 如果有多个实例，总限制 = 实例数 × 单实例限制

---

## 📚 下一步（第二阶段：黑名单机制）

### 计划实施的功能

1. **验证失败计数**
   - 记录认证失败的IP
   - 5分钟内5次失败→拉黑1小时

2. **白名单支持**
   - 支持单个IP和CIDR格式
   - 白名单IP免受所有限制

3. **管理API**
   - 查看当前黑名单
   - 查看IP状态
   - 手动解除拉黑
   - 清空黑名单
   - 管理白名单

4. **配置**
   - MaxFailures: 失败次数阈值
   - WindowSeconds: 时间窗口（秒）
   - BlockDuration: 拉黑时长（秒）
   - Whitelist: 白名单数组
   - CleanupInterval: 清理间隔（秒）

---

## 🐛 故障排除

### 问题1: 配置未生效
```bash
# 检查配置文件是否正确加载
docker-compose logs gotify | grep "rate"
```

### 问题2: 速率限制过于严格
```bash
# 调整docker-compose.yml中的环境变量
# 然后重启
docker-compose restart gotify
```

### 问题3: 编译错误
```bash
# 清理并重新构建
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

---

**第一阶段（速率限制）实施完成！准备进入第二阶段（黑名单机制）。**
