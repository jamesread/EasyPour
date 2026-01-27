<div align = "center">
  <img alt = "project logo" src = "https://github.com/jamesread/EasyPour/blob/main/logo.svg" width = "128" />
  <h1>EasyPour</h1>

  Send sips and snacks straight to servers — or your spouse.

[![Maturity Badge](https://img.shields.io/badge/maturity-Production-brightgreen)](#none)
[![Discord](https://img.shields.io/discord/846737624960860180?label=Discord%20Server)](https://discord.gg/jhYWWpNJ3v)
</div>

## Quick Start

```bash
docker pull ghcr.io/jamesread/easypour:latest
docker run -p 9654:9654 ghcr.io/jamesread/easypour:latest
```

The API is available at `http://localhost:9654`. For the full web UI, run the frontend locally (see [Development](#development) below).

## Development

To run the full stack from source:

```bash
make generate
make install
make dev-service   # terminal 1: backend
make dev-frontend  # terminal 2: frontend
```

Visit `http://localhost:3000` for the web UI.

## Project Structure

- `protocol/` - ConnectRPC protocol definitions (buf + protobuf)
- `service/` - Go backend service
- `frontend/` - Vue 3 + Vite frontend

## Prerequisites

- Go 1.21+
- Node.js 18+
- buf CLI: `go install github.com/bufbuild/buf/cmd/buf@latest`
