# AI-Nexus Postman testing

Import both JSON files into Postman, select the **AI-Nexus Local** environment,
then run a login request. Its test script saves the returned `token` as
`jwt_token`, which supplies the Collection-level Bearer authentication.

The environment defaults to the current WSL LAN address. Use
`http://localhost:8000` instead when Postman runs on the same Windows machine.

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

The external gRPC verification service is not part of this repository; until
it is running, `send-code` cannot issue a real verification code. Model
creation and image generation also need a valid provider configuration, and
image workers need `MODELSCOPE_API_KEY`.
