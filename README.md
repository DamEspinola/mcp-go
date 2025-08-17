# MCP-GO

A production-ready MCP (Model Context Protocol) server template (oauth authorization included) built in Go that works with major AI providers.

## Motivation
The MCP specification is relatively new and lacks comprehensive documentation for building complete servers in Go. 
This template provides a fully compliant MCP server that integrates seamlessly with remote providers like Claude Web, and OpenAI, 
but with local providers like Claude Desktop too.

## Features

- 🔐 **OAuth RFC 8414 and RFC 9728 compliant**
  - Support for `.well-known/oauth-protected-resource` and `.well-known/oauth-authorization-server` endpoints
  - Both endpoints are configurable

- 🛡️ **Several JWT validation methods**
  - Delegated to external systems like Istio
  - Locally validated based on JWKS URI and CEL expressions for claims

- 📋 Access logs can exclude or redact fields
- 🚀 Production-ready: Included full examples, Dockerfile, Helm Chart and GitHub Actions for CI
- ⚡ Super easy to extend: Production vitamins added to a good juice: [mcp-go](https://github.com/mark3labs/mcp-go)


## Deployment

### Production 🚀
Deploy to Kubernetes using the Helm chart located in the `chart/` directory.

---

Our recommendations for remote servers in production:

- Use a consistent hashring HTTP proxy in front of your MCP server when using MCP Sessions

- Use an HTTP proxy that performs JWT validation in front of the MCP instead of using the included middleware:
    - Protect your MCP exactly in the same way our middleware does, but with a super tested and scalable proxy instead
    - Improve your development experience as you don't have to do anything, just develop your MCP tools

- Use an OIDP that:
    - Cover Oauth Dynamic Client Registration
    - Is able to custom your JWT claims.

👉 [Keycloak](https://github.com/keycloak/keycloak) covers everything you need in the Oauth2 side

👉 [Istio](https://github.com/istio/istio) covers all you need to validate the JWT in front of your MCP



## How to develop your MCP

### Prerequisites
- Go 1.24+

### Modify the code and run

Obviously, you can modify the entire codebase as this is a template, but:
- Most times, you will only need to code inside `internal/tools` directory **to add your MCP Tools and their logic**
- Sometimes, you will need to **add your MCP Resources**. For that, it's recommended: 
  - To clone `internal/tools` directory into `internal/resources` and rename the structures there
  - To add the missing logic in `cmd/main.go` for invoking your code

Run: `make run`

**Note:** Default YAML config executing the previous command start the server as an HTTP server. 
To start it as Stdio mode, just modify the Makefile to use other YAML provided in examples.

### Configuration Examples

#### 🔗 Remote Clients (Claude Web, OpenAI)

Remote clients like Claude Web have different requirements than local ones. 
This project is fully ready for dealing with Claude Web with zero effort in your side.

In general, if you follow our recommendations on production, all the remote clients are covered 😊

> [!NOTE]  
> Hey! look at the configuration [here](./docs/config-http.yaml)

#### 💻 Local Clients (Claude Desktop, Cursor, VSCode)

Local clients configuration is commonly based in a JSON file with a specific standard structure. For example,
Claude Desktop can be configured by modifying the settings file called `claude_desktop_config.json` with the following sections:

##### Stdio Mode

If you want to use stdio as transport layer, it's recommended to compile your Go binary and then configure the client
as follows. This is recommended in local development as it is easy to work with.

Execute the following before configuring the client:

```console
make build
```

> [!IMPORTANT]
> When using Stdio transport, there is no protection between your client and the server, as they are both running locally

```json5
// file: claude_desktop_config.json

{
  "mcpServers": {
    "stdio": {
      "command": "/home/example/mcp-go/bin/mcp-go-linux-amd64",
      "args": [
        "--config",
        "/home/example/mcp-go/docs/config-stdio.yaml"
      ]
    }
  }
}
```

##### HTTP Mode

It is possible to launch your MCP server using HTTP transport. As most of local clients doesn't support connecting to
remote servers natively, we use a package (`mcp-remote`) to act as an intermediate between the expected stdio, 
and the remote server, which is launched locally too.

This is ideal to work on all the features that will be deployed in production, as everything related to how remote clients 
will behave later is available, so everything can be truly tested

Execute the following before configuring the client:

```console
npm i mcp-remote && \
make run
```

```json5
// file: claude_desktop_config.json

{
 "mcpServers": {
   "local-proxy-remote": {
     "command": "npx",
     "args": [
       "mcp-remote",
       "http://localhost:8080/mcp",
       "--transport",
       "http-only",
       "--header",
       "Authorization: Bearer ${JWT}",
       "--header",
       "X-Validated-Jwt: ${JWT}"
     ],
     "env": {
       "JWT": "eyJhbGciOiJSUzI1NiIsImtpZCI6..."
     }
   }
 }
}
```


## 🌐 Documentation

Do you feel for reading about this topic? I gave you the advice, then don't complain:

👉 [MCP Authorization Requirements](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization#overview)

👉 [RFC 9728](https://datatracker.ietf.org/doc/rfc9728/)

👉 [MCP Go Documentation](https://mcp-go.dev/getting-started)

👉 [mcp-remote package](https://www.npmjs.com/package/mcp-remote)

