package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ============================================================
// 网易云音乐 API 客户端
// ============================================================

const (
	baseURL   = "https://music.163.com"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// apiGet 发起 GET 请求
func apiGet(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", "https://music.163.com/")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// apiPost 发起 POST 请求（form 表单）
func apiPost(path string, params url.Values) ([]byte, error) {
	req, err := http.NewRequest("POST", baseURL+path, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", "https://music.163.com/")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ============================================================
// 搜索歌曲
// ============================================================

func searchSong(keyword string, limit int) (string, error) {
	params := url.Values{
		"s":      {keyword},
		"type":   {"1"}, // 1=歌曲
		"limit":  {strconv.Itoa(limit)},
		"offset": {"0"},
	}
	data, err := apiPost("/api/cloudsearch/pc", params)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	r, ok := result["result"].(map[string]interface{})
	if !ok {
		return "没有搜索到相关歌曲", nil
	}
	songs, ok := r["songs"].([]interface{})
	if !ok || len(songs) == 0 {
		return "没有搜索到相关歌曲", nil
	}

	var sb strings.Builder
	for i, s := range songs {
		song := s.(map[string]interface{})
		name := song["name"].(string)
		id := song["id"].(float64)

		// 提取歌手
		artists := song["ar"].([]interface{})
		artistNames := make([]string, 0)
		for _, a := range artists {
			artist := a.(map[string]interface{})
			artistNames = append(artistNames, artist["name"].(string))
		}

		// 提取专辑
		albumName := ""
		if album, ok := song["al"].(map[string]interface{}); ok {
			albumName = album["name"].(string)
		}

		// 时长
		duration := ""
		if d, ok := song["dt"].(float64); ok {
			mins := int(d) / 1000 / 60
			secs := (int(d) / 1000) % 60
			duration = fmt.Sprintf("%d:%02d", mins, secs)
		}

		sb.WriteString(fmt.Sprintf("%d. %s - %s\n", i+1, name, strings.Join(artistNames, "/")))
		sb.WriteString(fmt.Sprintf("   专辑: %s | 时长: %s | ID: %.0f\n", albumName, duration, id))
		sb.WriteString(fmt.Sprintf("   链接: https://music.163.com/song?id=%.0f\n\n", id))
	}
	return sb.String(), nil
}

// ============================================================
// 搜索歌手
// ============================================================

func searchArtist(keyword string, limit int) (string, error) {
	params := url.Values{
		"s":      {keyword},
		"type":   {"100"}, // 100=歌手
		"limit":  {strconv.Itoa(limit)},
		"offset": {"0"},
	}
	data, err := apiPost("/api/cloudsearch/pc", params)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	r, ok := result["result"].(map[string]interface{})
	if !ok {
		return "没有搜索到相关歌手", nil
	}
	artists, ok := r["artists"].([]interface{})
	if !ok || len(artists) == 0 {
		return "没有搜索到相关歌手", nil
	}

	var sb strings.Builder
	for i, a := range artists {
		artist := a.(map[string]interface{})
		name := artist["name"].(string)
		id := artist["id"].(float64)

		alias := ""
		if aliases, ok := artist["alias"].([]interface{}); ok && len(aliases) > 0 {
			aliasStrs := make([]string, 0)
			for _, al := range aliases {
				aliasStrs = append(aliasStrs, al.(string))
			}
			alias = " (" + strings.Join(aliasStrs, "/") + ")"
		}

		sb.WriteString(fmt.Sprintf("%d. %s%s | ID: %.0f\n", i+1, name, alias, id))
		sb.WriteString(fmt.Sprintf("   链接: https://music.163.com/artist?id=%.0f\n\n", id))
	}
	return sb.String(), nil
}

// ============================================================
// 获取歌词
// ============================================================

func getSongLyrics(songID int) (string, error) {
	path := fmt.Sprintf("/api/song/lyric?id=%d&lv=1&kv=1&tv=-1", songID)
	data, err := apiGet(path)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	var sb strings.Builder

	// 原歌词
	if lrc, ok := result["lrc"].(map[string]interface{}); ok {
		if lyric, ok := lrc["lyric"].(string); ok && lyric != "" {
			sb.WriteString("【歌词】\n")
			sb.WriteString(lyric)
			sb.WriteString("\n")
		}
	}

	// 翻译歌词
	if tlyric, ok := result["tlyric"].(map[string]interface{}); ok {
		if lyric, ok := tlyric["lyric"].(string); ok && lyric != "" {
			sb.WriteString("\n【翻译】\n")
			sb.WriteString(lyric)
		}
	}

	if sb.Len() == 0 {
		return "该歌曲暂无歌词", nil
	}
	return sb.String(), nil
}

// ============================================================
// 获取歌曲评论（热评 + 最新评论）
// ============================================================

func getSongComments(songID int, limit int) (string, error) {
	// 先获取热评
	hotPath := fmt.Sprintf("/api/v1/resource/hotcomments/R_SO_4_%d?limit=%d&offset=0", songID, limit)
	hotData, err := apiGet(hotPath)
	if err != nil {
		return "", err
	}

	// 再获取最新评论
	newPath := fmt.Sprintf("/api/v1/resource/comments/R_SO_4_%d?limit=%d&offset=0", songID, limit)
	newData, err := apiGet(newPath)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	// 解析热评
	var hotResult map[string]interface{}
	if err := json.Unmarshal(hotData, &hotResult); err == nil {
		if hotComments, ok := hotResult["hotComments"].([]interface{}); ok && len(hotComments) > 0 {
			sb.WriteString("🔥 热门评论\n")
			sb.WriteString(strings.Repeat("─", 40) + "\n\n")
			for i, c := range hotComments {
				comment := c.(map[string]interface{})
				content := comment["content"].(string)
				likedCount := comment["likedCount"].(float64)

				nickname := "匿名"
				if user, ok := comment["user"].(map[string]interface{}); ok {
					if n, ok := user["nickname"].(string); ok {
						nickname = n
					}
				}

				timeStr := ""
				if t, ok := comment["time"].(float64); ok {
					tm := time.Unix(int64(t)/1000, 0)
					timeStr = tm.Format("2006-01-02 15:04")
				}

				sb.WriteString(fmt.Sprintf("%d. 💬 %s\n", i+1, content))
				sb.WriteString(fmt.Sprintf("   —— %s | %s | 👍 %.0f\n\n", nickname, timeStr, likedCount))
			}
		}
	}

	// 解析最新评论
	var newResult map[string]interface{}
	if err := json.Unmarshal(newData, &newResult); err == nil {
		// 总评论数
		if total, ok := newResult["total"].(float64); ok {
			sb.WriteString(fmt.Sprintf("\n📊 总评论数: %.0f\n\n", total))
		}

		if comments, ok := newResult["comments"].([]interface{}); ok && len(comments) > 0 {
			sb.WriteString("💬 最新评论\n")
			sb.WriteString(strings.Repeat("─", 40) + "\n\n")
			for i, c := range comments {
				comment := c.(map[string]interface{})
				content := comment["content"].(string)
				likedCount := comment["likedCount"].(float64)

				nickname := "匿名"
				if user, ok := comment["user"].(map[string]interface{}); ok {
					if n, ok := user["nickname"].(string); ok {
						nickname = n
					}
				}

				timeStr := ""
				if t, ok := comment["time"].(float64); ok {
					tm := time.Unix(int64(t)/1000, 0)
					timeStr = tm.Format("2006-01-02 15:04")
				}

				sb.WriteString(fmt.Sprintf("%d. 💬 %s\n", i+1, content))
				sb.WriteString(fmt.Sprintf("   —— %s | %s | 👍 %.0f\n\n", nickname, timeStr, likedCount))
			}
		}
	}

	if sb.Len() == 0 {
		return "该歌曲暂无评论", nil
	}
	return sb.String(), nil
}

// ============================================================
// 获取歌单详情
// ============================================================

func getPlaylistDetail(playlistID int) (string, error) {
	path := fmt.Sprintf("/api/v6/playlist/detail?id=%d", playlistID)
	data, err := apiGet(path)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	playlist, ok := result["playlist"].(map[string]interface{})
	if !ok {
		return "未找到该歌单", nil
	}

	var sb strings.Builder

	name := playlist["name"].(string)
	sb.WriteString(fmt.Sprintf("🎵 歌单: %s\n", name))
	sb.WriteString(strings.Repeat("─", 40) + "\n")

	if desc, ok := playlist["description"].(string); ok && desc != "" {
		sb.WriteString(fmt.Sprintf("📝 简介: %s\n", desc))
	}

	if creator, ok := playlist["creator"].(map[string]interface{}); ok {
		if n, ok := creator["nickname"].(string); ok {
			sb.WriteString(fmt.Sprintf("👤 创建者: %s\n", n))
		}
	}

	if count, ok := playlist["trackCount"].(float64); ok {
		sb.WriteString(fmt.Sprintf("🎶 歌曲数: %.0f\n", count))
	}
	if playCount, ok := playlist["playCount"].(float64); ok {
		sb.WriteString(fmt.Sprintf("▶️  播放量: %.0f\n", playCount))
	}

	sb.WriteString("\n")

	// 歌曲列表（trackIds 里只有 id，需要用 tracks 如果有的话）
	if tracks, ok := playlist["tracks"].([]interface{}); ok && len(tracks) > 0 {
		sb.WriteString("📋 歌曲列表:\n\n")
		maxShow := 50
		if len(tracks) < maxShow {
			maxShow = len(tracks)
		}
		for i := 0; i < maxShow; i++ {
			track := tracks[i].(map[string]interface{})
			trackName := track["name"].(string)
			trackID := track["id"].(float64)

			artistNames := make([]string, 0)
			if ar, ok := track["ar"].([]interface{}); ok {
				for _, a := range ar {
					if artist, ok := a.(map[string]interface{}); ok {
						if n, ok := artist["name"].(string); ok {
							artistNames = append(artistNames, n)
						}
					}
				}
			}

			sb.WriteString(fmt.Sprintf("  %d. %s - %s (ID: %.0f)\n",
				i+1, trackName, strings.Join(artistNames, "/"), trackID))
		}
		if len(tracks) > maxShow {
			sb.WriteString(fmt.Sprintf("\n  ... 还有 %d 首歌曲\n", len(tracks)-maxShow))
		}
	}

	return sb.String(), nil
}

// ============================================================
// 搜索歌单
// ============================================================

func searchPlaylist(keyword string, limit int) (string, error) {
	params := url.Values{
		"s":      {keyword},
		"type":   {"1000"}, // 1000=歌单
		"limit":  {strconv.Itoa(limit)},
		"offset": {"0"},
	}
	data, err := apiPost("/api/cloudsearch/pc", params)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	r, ok := result["result"].(map[string]interface{})
	if !ok {
		return "没有搜索到相关歌单", nil
	}
	playlists, ok := r["playlists"].([]interface{})
	if !ok || len(playlists) == 0 {
		return "没有搜索到相关歌单", nil
	}

	var sb strings.Builder
	for i, p := range playlists {
		pl := p.(map[string]interface{})
		name := pl["name"].(string)
		id := pl["id"].(float64)

		creatorName := ""
		if creator, ok := pl["creator"].(map[string]interface{}); ok {
			if n, ok := creator["nickname"].(string); ok {
				creatorName = n
			}
		}

		trackCount := float64(0)
		if tc, ok := pl["trackCount"].(float64); ok {
			trackCount = tc
		}
		playCount := float64(0)
		if pc, ok := pl["playCount"].(float64); ok {
			playCount = pc
		}

		desc := ""
		if d, ok := pl["description"].(string); ok && d != "" {
			// 截取前80个字符
			r := []rune(d)
			if len(r) > 80 {
				desc = string(r[:80]) + "..."
			} else {
				desc = d
			}
		}

		sb.WriteString(fmt.Sprintf("%d. 🎵 %s\n", i+1, name))
		sb.WriteString(fmt.Sprintf("   创建者: %s | 歌曲数: %.0f | 播放: %.0f\n", creatorName, trackCount, playCount))
		if desc != "" {
			sb.WriteString(fmt.Sprintf("   简介: %s\n", desc))
		}
		sb.WriteString(fmt.Sprintf("   ID: %.0f | 链接: https://music.163.com/playlist?id=%.0f\n\n", id, id))
	}
	return sb.String(), nil
}

// ============================================================
// MCP 工具 Handler
// ============================================================

func searchSongHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword := request.Params.Arguments["keyword"].(string)
	result, err := searchSong(keyword, 20)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("搜索失败: %v", err)), nil
	}
	return mcp.NewToolResultText(result), nil
}

func searchArtistHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword := request.Params.Arguments["keyword"].(string)
	result, err := searchArtist(keyword, 15)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("搜索失败: %v", err)), nil
	}
	return mcp.NewToolResultText(result), nil
}

func getLyricsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	songID := int(request.Params.Arguments["song_id"].(float64))
	result, err := getSongLyrics(songID)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取歌词失败: %v", err)), nil
	}
	return mcp.NewToolResultText(result), nil
}

func getCommentsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	songID := int(request.Params.Arguments["song_id"].(float64))
	limit := 15
	if l, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(l)
	}
	result, err := getSongComments(songID, limit)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取评论失败: %v", err)), nil
	}
	return mcp.NewToolResultText(result), nil
}

func getPlaylistHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	playlistID := int(request.Params.Arguments["playlist_id"].(float64))
	result, err := getPlaylistDetail(playlistID)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取歌单失败: %v", err)), nil
	}
	return mcp.NewToolResultText(result), nil
}

func searchPlaylistHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword := request.Params.Arguments["keyword"].(string)
	result, err := searchPlaylist(keyword, 15)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("搜索失败: %v", err)), nil
	}
	return mcp.NewToolResultText(result), nil
}

// ============================================================
// 主入口
// ============================================================

func main() {
	s := server.NewMCPServer(
		"网易云音乐 MCP 服务器",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// 注册工具：搜索歌曲
	s.AddTool(
		mcp.NewTool("search_song",
			mcp.WithDescription("搜索网易云音乐歌曲，返回歌曲名、歌手、专辑、时长和链接"),
			mcp.WithString("keyword", mcp.Required(), mcp.Description("搜索关键词，可以是歌名、歌手名或歌词片段")),
		),
		searchSongHandler,
	)

	// 注册工具：搜索歌手
	s.AddTool(
		mcp.NewTool("search_artist",
			mcp.WithDescription("搜索网易云音乐歌手"),
			mcp.WithString("keyword", mcp.Required(), mcp.Description("歌手名")),
		),
		searchArtistHandler,
	)

	// 注册工具：获取歌词
	s.AddTool(
		mcp.NewTool("get_lyrics",
			mcp.WithDescription("获取歌曲的歌词（包括原文和翻译）"),
			mcp.WithNumber("song_id", mcp.Required(), mcp.Description("歌曲ID，可通过search_song获取")),
		),
		getLyricsHandler,
	)

	// 注册工具：获取评论
	s.AddTool(
		mcp.NewTool("get_comments",
			mcp.WithDescription("获取歌曲的热门评论和最新评论"),
			mcp.WithNumber("song_id", mcp.Required(), mcp.Description("歌曲ID")),
			mcp.WithNumber("limit", mcp.Description("返回评论数量，默认15")),
		),
		getCommentsHandler,
	)

	// 注册工具：获取歌单详情
	s.AddTool(
		mcp.NewTool("get_playlist",
			mcp.WithDescription("获取歌单详情，包括歌曲列表"),
			mcp.WithNumber("playlist_id", mcp.Required(), mcp.Description("歌单ID")),
		),
		getPlaylistHandler,
	)

	// 注册工具：搜索歌单
	s.AddTool(
		mcp.NewTool("search_playlist",
			mcp.WithDescription("搜索网易云音乐歌单"),
			mcp.WithString("keyword", mcp.Required(), mcp.Description("搜索关键词，如心情、场景、风格")),
		),
		searchPlaylistHandler,
	)

	// 以 SSE 模式启动
	sse := server.NewSSEServer(s, server.WithBaseURL("https://qiyun.cloud/netease"))
	fmt.Println("网易云音乐 MCP 服务器启动中... 端口 8081")
	if err := sse.Start(":8081"); err != nil {
		fmt.Printf("服务器错误: %v\n", err)
	}
}
