# codex-app-server-go

[![xc compatible](https://xcfile.dev/badge.svg)](https://xcfile.dev)

Codex App Server for Golang

## Supported Protocols

* Stdio

## Roadmaps

* See [ROADMAP.md](./ROADMAP.md) for more details

## Documentation

* Official Codex App Server: https://developers.openai.com/codex/app-server
  * Cached: [docs/codex-app-server/260322-codex-app-server.md](./docs/codex-app-server/260322-codex-app-server.md)

## Tasks

* Below is for [xc](https://github.com/joerdav/xc)

### sync-codex-app-schema

* Sync codex app schema to [api/codex-app-server](./api/codex-app-server)

```
#!/usr/bin/env bash

codex app-server generate-json-schema --out api/codex-app-server
```
