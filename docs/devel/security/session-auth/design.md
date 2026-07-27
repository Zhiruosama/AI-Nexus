# 可撤销会话认证：需求设计

## 1. 设计原则

JWT 负责证明 Token 是服务端签发且未被篡改；Redis 负责保存可撤销的当前会话状态。两者职责不同，不能用 JWT 单独承担注销和踢下线能力。

## 2. JWT Claims

JWT 采用以下字段：

```json
{
  "sub": "user-uuid",
  "jti": "session-uuid",
  "exp": 1780000000,
  "iat": 1779000000,
  "iss": "ai-nexus-auth"
}
```

| 字段 | 含义 | 来源 |
| --- | --- | --- |
| `sub` | 用户 ID | 登录用户 |
| `jti` | 本次登录会话 ID | 服务端安全随机生成 |
| `exp` | Token 过期时间 | 服务端计算 |
| `iat` | Token 签发时间 | 服务端记录 |
| `iss` | Token 签发者 | 固定配置 |

不再额外保存含义重复的 `user_id` 字段；业务代码统一从 `sub` 读取用户 ID。

## 3. Redis 会话模型

### 3.1 Key

```text
auth:session:{jti}
```

示例：

```text
auth:session:session-uuid-456
```

### 3.2 Value

第一版可保存用户 ID 字符串：

```text
user-uuid-123
```

后续需要设备管理时可扩展为 JSON：

```json
{
  "user_id": "user-uuid-123",
  "created_at": "2026-07-27T12:00:00Z",
  "device": "web"
}
```

### 3.3 TTL

Redis TTL 应设置为 JWT `exp - now`，并且不能长于 JWT 的剩余有效期。Redis 只保存当前会话状态，不保存永久用户数据。

### 3.4 用户会话索引

为了支持密码重置和账号注销时撤销用户全部会话，额外维护：

```text
auth:user-sessions:{user_id} -> Set<jti>
```

创建会话时同时写入会话 Key 和用户会话集合；撤销单个会话时同时删除会话 Key 和集合成员；撤销全部会话时根据集合批量删除对应会话 Key。

## 4. 核心流程

### 4.1 登录

```text
校验账号密码
  ↓
生成随机 jti
  ↓
生成包含 sub、jti、exp 的 JWT
  ↓
SET auth:session:{jti} = sub EX ttl
  ↓
返回 JWT
```

Redis 写入失败时，登录不能返回成功 Token，避免出现“客户端拿到 Token 但服务端没有会话”的不一致状态。

### 4.2 请求认证

```text
Authorization: Bearer <token>
  ↓
解析 JWT
  ↓
校验签名、算法、exp、iss
  ↓
读取 sub、jti
  ↓
GET auth:session:{jti}
  ↓
不存在：401
  ↓
值 != sub：401
  ↓
写入 Principal{UserID: sub, SessionID: jti}
  ↓
进入业务 Handler
```

认证中间件的所有失败分支都必须调用 `Abort`，保证后续 Handler 不会继续执行。

### 4.3 退出登录

```text
认证成功
  ↓
从 Principal 读取 jti
  ↓
DEL auth:session:{jti}
  ↓
返回成功
```

删除操作应具备幂等性：会话已经不存在时，重复退出仍可返回成功或明确的已退出结果。

### 4.4 密码重置与账号注销

```text
完成身份校验
  ↓
读取 auth:user-sessions:{user_id}
  ↓
删除该用户全部 auth:session:{jti}
  ↓
删除用户会话集合
  ↓
执行密码更新或账号注销
```

采用“先撤销会话，再更新数据库”的安全顺序。Redis 不可用时敏感操作直接失败，
避免数据库已经发生变更、旧会话却仍然有效。若后续数据库操作失败，用户需要重新
登录，但不会遗留可继续使用的旧会话。

## 5. 代码落地顺序

1. 扩展 Claims，增加 `SessionID/jti`，统一使用 `sub` 表示用户 ID。
2. 登录服务生成随机会话 ID，并写入会话 Key 与用户会话索引。
3. 定义 `Principal`，由认证中间件写入 Gin Context。
4. 改造 AuthMiddleware，加入 Redis 会话读取和一致性校验。
5. 改造 Logout，按 `jti` 删除会话。
6. 保持现有业务 Handler 从 Context 读取用户 ID 的兼容层，再逐步迁移到 Principal。
7. 补充单元测试和登录/退出集成测试。

## 6. 测试设计

### 单元测试

- Claims 生成包含非空 `sub`、`jti`、`exp`。
- 只接受约定的 `HS256` 签名算法。
- 过期 Token 解析失败。
- Redis 会话不存在时认证失败。
- Redis 用户与 `sub` 不一致时认证失败。
- 认证失败不会调用后续 Handler。
- 单会话撤销不影响同一用户的其他会话。
- 全部撤销只删除目标用户的会话。

### 集成测试

```text
登录 → 获取受保护资源 → 退出 → 使用原 Token 再次请求
```

预期最后一步返回 `401 Unauthorized`。

## 7. 兼容与迁移

旧 Token 没有 `jti`，不能通过新的会话校验。实现切换后，旧 Token 应统一失效，用户重新登录即可；不应为了兼容旧 Token 而保留绕过 Redis 的安全分支。

旧实现以用户 ID 作为 Redis Key 且没有 TTL。切换后这些旧 Key 不再参与认证，可在确认回滚窗口结束后单独清理。

## 8. 后续扩展

- 按用户查询全部会话；
- 设备、IP、最后活跃时间；
- 管理员踢下线；
- Refresh Token；
- 高风险操作再次验证。
