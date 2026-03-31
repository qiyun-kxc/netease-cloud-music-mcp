# NetEase Cloud Music MCP Server

A lightweight MCP (Model Context Protocol) server for NetEase Cloud Music, written from scratch in Go. Enables AI agents to search songs, artists, and playlists, fetch lyrics with translations, read user comments, get audio URLs, and discover new music through random recommendations.

## Features

| Tool | Description |
|------|-------------|
| `search_song` | Search songs by keyword |
| `search_artist` | Search artists by keyword |
| `search_playlist` | Search playlists by keyword |
| `get_playlist` | Get playlist details and full track list |
| `get_lyrics` | Get song lyrics (original + translation) |
| `get_comments` | Get hot and recent comments on a song |
| `random_recommend` | Random songs from public charts (热歌榜, 新歌榜, 民谣榜, ACG榜, etc.) |
| `get_song_url` | Get audio playback URL, bitrate, format, and file size |

## Changelog

### 2026-03-31

**New tools:**

- `random_recommend` — picks a random public chart (out of 10: 热歌榜, 新歌榜, 原创榜, 飙升榜, 古典音乐榜, 华语金曲榜, 网络热歌榜, 后摇榜, ACG音乐榜, 民谣榜), then randomly draws N songs from it. No login required.
- `get_song_url` — fetches audio playback link, bitrate (kbps), file size, and format. Some songs may be unavailable due to copyright restrictions (returns a note instead of failing).

**Bug fix:**

- `toInt` helper — MCP clients (notably Claude Code via SSE) sometimes pass numeric parameters as strings instead of float64. The old `.(float64)` type assertion caused a panic. New `toInt()` function handles both types gracefully. Affects: `get_lyrics`, `get_comments`, `get_playlist`, `get_song_url`, `random_recommend`.

## Why We Built This

Existing NetEase Cloud Music API wrappers are mostly Python-based and designed for web apps, not MCP. We needed a Go server that speaks SSE (Server-Sent Events) natively and can be deployed behind an Nginx reverse proxy alongside other MCP servers.

This was written from scratch using the [mcp-go](https://github.com/mark3labs/mcp-go) library and NetEase's public API endpoints.

## Technical Notes

- **Runtime**: Go 1.23+
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

### Known Issue: Type Assertion Panic (fixed 2026-03-31)

MCP clients may send numeric tool arguments as strings. Direct `.(float64)` assertion panics when this happens. Fixed by introducing a `toInt()` helper:

```go
func toInt(v interface{}) (int, error) {
    switch val := v.(type) {
    case float64:
        return int(val), nil
    case string:
        return strconv.Atoi(val)
    default:
        return 0, fmt.Errorf("unsupported type: %T", v)
    }
}
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

### Local deployment (e.g. for Claude Code)

If running on the same machine as Claude Code, change the `WithBaseURL` in `main.go` from the production domain to `http://localhost:8081/netease`, then register as an SSE MCP server:

```bash
claude mcp add --transport sse -s user netease http://localhost:8081/netease/sse
```

## Part of CloudPerch

This server is one component of [CloudPerch](https://github.com/qiyun-kxc/cloudperch), an MCP gateway platform. See the main repo for full architecture and deployment guide.

## License

MIT
