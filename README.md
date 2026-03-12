# NetEase Cloud Music MCP Server

A lightweight MCP (Model Context Protocol) server for NetEase Cloud Music, written from scratch in Go. Enables AI agents to search songs, artists, and playlists, fetch lyrics with translations, and read user comments.

## Features

| Tool | Description |
|------|-------------|
| `search_song` | Search songs by keyword |
| `search_artist` | Search artists by keyword |
| `search_playlist` | Search playlists by keyword |
| `get_playlist` | Get playlist details and full track list |
| `get_lyrics` | Get song lyrics (original + translation) |
| `get_comments` | Get hot and recent comments on a song |

## Why We Built This

Existing NetEase Cloud Music API wrappers are mostly Python-based and designed for web apps, not MCP. We needed a Go server that speaks SSE (Server-Sent Events) natively and can be deployed behind an Nginx reverse proxy alongside other MCP servers.

This was written from scratch using the [mcp-go](https://github.com/mark3labs/mcp-go) library and NetEase's public API endpoints.

## Technical Notes

- **Runtime**: Go 1.22+
- **Transport**: SSE (Server-Sent Events)
- **Default port**: 8081
- **API**: Uses NetEase Cloud Music's public web API (`music.163.com`)
- **No authentication required** for public search and metadata

### Known Issue: POST vs GET

NetEase's `/api/search/get` endpoint expects GET requests with query parameters, not POST with form body. The original implementation used POST, which caused search results to return unrelated content. Fix:

```go
// Before (broken):
data, err := apiPost("/api/search/get", params)

// After (working):
data, err := apiGet("/api/search/get?" + params.Encode())
```

## Deployment

```bash
# Build
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o netease-mcp-sse main.go

# Run with PM2
pm2 start ./netease-mcp-sse --name netease-mcp

# Or with systemd
sudo systemctl start netease-mcp
```

### Nginx Configuration

```nginx
location /netease/ {
    proxy_pass http://127.0.0.1:8081/;
    proxy_http_version 1.1;
    proxy_set_header Connection '';
    proxy_buffering off;
    proxy_cache off;
}
```

## Part of CloudPerch

This server is one component of [CloudPerch](https://github.com/qiyun-kxc/cloudperch), an MCP gateway platform. See the main repo for full architecture and deployment guide.

## License

MIT
