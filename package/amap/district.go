// 行政区域查询（省 / 市 / 区下级列表）
// 文档：https://lbs.amap.com/api/webservice/guide/api/district
package amap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const districtURL = "https://restapi.amap.com/v3/config/district"

// DistrictNode 行政区节点（用于级联选择）
type DistrictNode struct {
	Adcode string `json:"adcode"`
	Name   string `json:"name"`
	Center string `json:"center"`
	Level  string `json:"level"`
}

// DistrictChildren 查询某级行政区下一层子节点。keywords 可为 adcode（如 100000 中国）或行政区名称。
// subdistrict=1，只取一层子级。
func DistrictChildren(ctx context.Context, key, keywords string, page, offset int) ([]DistrictNode, error) {
	keywords = strings.TrimSpace(keywords)
	if key == "" {
		return nil, fmt.Errorf("key 为空")
	}
	if keywords == "" {
		keywords = "100000"
	}
	if page < 1 {
		page = 1
	}
	if offset < 1 {
		offset = 20
	}
	if offset > 50 {
		offset = 50
	}
	u, _ := url.Parse(districtURL)
	q := u.Query()
	q.Set("key", key)
	q.Set("keywords", keywords)
	q.Set("subdistrict", "1")
	q.Set("extensions", "base")
	q.Set("page", strconv.Itoa(page))
	q.Set("offset", strconv.Itoa(offset))
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var out struct {
		Status    string `json:"status"`
		Info      string `json:"info"`
		Districts []struct {
			Adcode    string `json:"adcode"`
			Name      string `json:"name"`
			Center    string `json:"center"`
			Level     string `json:"level"`
			Districts []struct {
				Adcode string `json:"adcode"`
				Name   string `json:"name"`
				Center string `json:"center"`
				Level  string `json:"level"`
			} `json:"districts"`
		} `json:"districts"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("解析行政区响应失败: %w", err)
	}
	if out.Status != "1" {
		return nil, fmt.Errorf("行政区查询失败: %s", out.Info)
	}
	if len(out.Districts) == 0 {
		return nil, nil
	}
	ch := out.Districts[0].Districts
	list := make([]DistrictNode, 0, len(ch))
	for _, c := range ch {
		list = append(list, DistrictNode{
			Adcode: c.Adcode,
			Name:   c.Name,
			Center: c.Center,
			Level:  c.Level,
		})
	}
	return list, nil
}
