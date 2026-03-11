A small Go service that sits in front of an HTTP resource, caches responses
in memory, and exposes a simple API for clearing the cache.

## Features

- In‑memory caching
- Proxy logic in `internal/proxy`
- Cache management in `internal/cache`

## Build & run

```bash
go build ./cmd
./cmd/main              # start server on default port (8080)
./cmd/main --clear-cache # clear cache and exit
```

## Usage

Point your client at the proxy instead of the origin URL.

```bash
curl http://localhost:8080/your/path
```

## Structure

```
cmd/      # executable entrypoint
internal/ # core packages: cache, proxy, server
```

That’s it – small, fast, and easy.
