// Package graphhopper 调用自建 GraphHopper 服务计算路径距离（与 OSM 路网一致，坐标为 WGS84）。
package graphhopper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// FootRouteDistanceMeters 返回两点间按 profile=foot 路径距离（米）。baseURL 示例 http://host:50002
func FootRouteDistanceMeters(ctx context.Context, baseURL string, lat1, lng1, lat2, lng2 float64) (int, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return 0, fmt.Errorf("GraphHopper 基地址为空")
	}
	u, err := url.Parse(baseURL + "/route")
	if err != nil {
		return 0, err
	}
	q := url.Values{}
	q.Add("point", fmt.Sprintf("%g,%g", lat1, lng1))
	q.Add("point", fmt.Sprintf("%g,%g", lat2, lng2))
	q.Set("profile", "foot")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("GraphHopper HTTP %d: %s", res.StatusCode, truncate(string(body), 200))
	}
	var out struct {
		Paths []struct {
			Distance float64 `json:"distance"`
		} `json:"paths"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, fmt.Errorf("解析 GraphHopper 响应: %w", err)
	}
	if len(out.Paths) == 0 {
		if out.Message != "" {
			return 0, fmt.Errorf("GraphHopper: %s", out.Message)
		}
		return 0, fmt.Errorf("无路径结果")
	}
	d := out.Paths[0].Distance
	if d <= 0 {
		return 0, fmt.Errorf("无效距离")
	}
	return int(d + 0.5), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
