# AI-Nexus Postman testing

Import both JSON files into Postman, select the **AI-Nexus Local** environment,
then run a login request. Its test script saves the returned `token` as
`jwt_token`, which supplies the Collection-level Bearer authentication.

The environment defaults to `http://localhost:8000`, which is the correct
address when Postman runs on the same Windows machine as WSL. A Postman client
on another LAN device needs a separate Windows firewall/networking rule; do
not assume the transient WSL address is directly reachable.

There are 35 application endpoints in the route registrations: 34 HTTP
endpoints and one WebSocket endpoint. The Collection has 36 HTTP request
examples because it includes three login variants for the single login route.

```text
{{ws_base_url}}/user/ws?token={{jwt_token}}
```

Create a WebSocket request in Postman with that URL after logging in. The
WebSocket token is a query parameter (not an Authorization header).

`/chat/conversations/:conv_id/messages` responds as SSE. Postman can send the
request and display the stream, but an SSE-capable client is easier for long
conversations.

`send-code` now depends on Email-Service at `127.0.0.1:8080`; Nexus receives
delivery callbacks on `:8081`. A successful request returns `202` and a
`request_id`. Reuse the same `Idempotency-Key` header only when retrying the
same logical send. Email-Service's Fake Provider validates the pipeline but
does not deliver a readable email; use its explicitly configured SMTP mode for
manual registration tests. Image workers still need `MODELSCOPE_API_KEY`.
