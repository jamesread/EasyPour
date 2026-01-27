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
docker run -p 9654:9654 -v easypour-config:/config ghcr.io/jamesread/easypour:latest
```

Config and menu live in `/config` (config.yaml and menu.yaml). Use a named volume `-v easypour-config:/config` to persist and edit them, or bind-mount a host dir: `-v /path/to/your/config:/config`.

The API is available at `http://localhost:9654`. For the full web UI, run the frontend locally (see [Development](#development) below).

