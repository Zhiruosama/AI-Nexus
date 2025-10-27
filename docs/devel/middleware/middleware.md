使用 gin.New()，完全自定义中间件

## Recovery: 最外层，捕获所有的 panic
并将panic传出的错误信息 以及堆栈内信息进行打印 做进一步错误判断


## Logging: 一次请求生成一个 Request ID，可以通过此 ID 追踪所有日志

## CORS: 处理跨域

## Security Headers: 添加安全头，防止跨域攻击

## Rate Limiting / Idempotency / Deduplication: 在执行核心逻辑前进行流量控制、幂等设计、重复请求判断

## Authentication (OAuth2/JWT): 验证用户身份

## Request Validation: 校验请求数据，正式进入 Controller 层
