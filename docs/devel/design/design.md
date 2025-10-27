# 图像生成模块

目标：只做最基本功能——接受用户传来的文本或图片，将其丢给模型生成结果并返回。

## 接口一览
- POST `/image/generate`
  - 描述：输入文本或图片（二选一或同时），将请求转发给默认模型；返回生成图片。
  - 请求方式：`application/json` 或 `multipart/form-data`。
  - 参数（最小化）：
    - `prompt` string（可选）文本提示词；提供则走文本生图。
    - `image_file` file（可选）源图；提供则走图生图。
    - `num` int（可选，默认`1`），生成张数。
  - 约束：至少提供 `prompt` 或 `image_file` 之一。
  - 响应：
    - `images`: `[ { id, url | base64 } ]`（最简返回，后续可扩展为存储元数据）。

- GET `/image/:id/file`
  - 描述：按图片 `id` 返回图片二进制内容。
  - 查询：`format?`（`png`/`jpg`/`webp`，默认`png`），`size?`（`origin`/`thumb`，默认`origin`）。
  - 响应：图片二进制（`Content-Type`按`format`）。

## 返回方式与选择
- 模式A：返回 URL（默认）
  - `POST /image/generate` 返回 JSON：`images: [{ id, url }]`。
  - 客户端用 `GET /image/:id/file?format=png&size=origin` 拉取二进制。
  - 适合多图与大图，响应体小，易做缓存/CDN。

- 模式B：直接返回图片
  - JSON Base64：`images: [{ id, base64 }]`，便于快速预览；但响应体较大。
  - 二进制直传（仅当 `num=1`）：
    - 方式一：设置请求头 `Accept: image/png`（或 `image/jpeg`、`image/webp`）。
    - 方式二：参数选择 `return_mode=inline&inline_type=binary&format=png`。
    - 响应头：`Content-Type: image/<format>`；可选 `Content-Disposition: inline; filename="image.<format>"`。

- 选择参数（建议）：
  - `return_mode`: `url` | `inline`（默认 `url`）。
  - `inline_type`: `base64` | `binary`（当 `return_mode=inline` 时有效）。
  - `format`: `png` | `jpg` | `webp`（适用于二进制直传与 `GET /image/:id/file`）。

- 响应示例：
  - URL 模式：
    ```json
    { "images": [ { "id": "123", "url": "/image/123/file?format=png" } ] }
    ```
  - Inline Base64：
    ```json
    { "images": [ { "id": "123", "base64": "data:image/png;base64,iVBORw0K..." } ] }
    ```
  - 二进制直传：响应为图片字节流，`Content-Type: image/png`。

- 约束与建议：
  - 当 `num > 1` 时不支持 `binary` 直传，建议使用 `url` 或 `base64`。
  - 大图/高并发场景优先使用 URL 模式以降低响应体体积。
  - 如需安全控制可引入签名 URL 或 `expires_at`（后续扩展，不在极简版实现范围）。

## 最小实现建议
- 模块目录：`internal/{routes,controller,service,dao}/image`，照抄 `demo` 的分层结构。
- 存储：
  - 数据表（极简）：`images(id, path, created_at)` 或直接文件系统存路径。
  - 返回时给出 `id` 与 `url`（或直接 `base64`）。
- 模型调用：
  - `service` 层根据是否携带 `prompt` 或 `image_file` 调用对应模型接口。
  - 先只支持一个默认模型（如本地或占位 mock），后续再扩展选择模型。

## 错误码（极简）
- `400` 参数错误（未提供 `prompt` 与 `image_file`）。
- `404` 图片不存在。
- `500` 服务内部错误。

## 示例
- 文本生图（JSON）：
  - `POST /image/generate`
  - Body: `{ "prompt": "a cat playing piano", "size": "512x512" }`

- 图生图（FormData）：
  - `POST /image/generate`
  - Form: `image_file=@/path/to/image.png; size=512x512`
