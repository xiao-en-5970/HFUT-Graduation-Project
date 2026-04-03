// 输入提示、POI 关键字搜索（GCJ-02）
// 文档：https://lbs.amap.com/api/webservice/guide/api/inputtips
// https://lbs.amap.com/api/webservice/guide/api/textsearch
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

const (
	inputTipsURL = "https://restapi.amap.com/v3/assistant/inputtips"
	placeTextURL = "https://restapi.amap.com/v3/place/text"
)

// PlaceTip 单条地点结果（坐标为高德 GCJ-02，与订单 receiver_lat/lng 一致）
type PlaceTip struct {
	ID       string  `json:"id,omitempty"`
	Name     string  `json:"name"`
	Address  string  `json:"address,omitempty"`
	District string  `json:"district,omitempty"`
	Adcode   string  `json:"adcode,omitempty"`
	Lng      float64 `json:"lng"`
	Lat      float64 `json:"lat"`
	HasCoord bool    `json:"has_coord"`
}

func parseLocationString(loc string) (lng, lat float64, ok bool) {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return 0, 0, false
	}
	parts := strings.Split(loc, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	lng, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, false
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, false
	}
	return lng, lat, true
}

// InputTips 输入提示（适合搜索框联想：用户输入关键字，返回候选地点）
func InputTips(ctx context.Context, key, keywords, city string, citylimit bool) ([]PlaceTip, error) {
	keywords = strings.TrimSpace(keywords)
	if key == "" || keywords == "" {
		return nil, fmt.Errorf("key 或 keywords 为空")
	}
	u, _ := url.Parse(inputTipsURL)
	q := u.Query()
	q.Set("key", key)
	q.Set("keywords", keywords)
	if city != "" {
		q.Set("city", city)
		if citylimit {
			q.Set("citylimit", "true")
		}
	}
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
		Status string `json:"status"`
		Info   string `json:"info"`
		Tips   []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			District string `json:"district"`
			Adcode   string `json:"adcode"`
			Location string `json:"location"`
			Address  string `json:"address"`
		} `json:"tips"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("解析输入提示响应失败: %w", err)
	}
	if out.Status != "1" {
		return nil, fmt.Errorf("输入提示失败: %s", out.Info)
	}
	list := make([]PlaceTip, 0, len(out.Tips))
	for _, t := range out.Tips {
		lng, lat, ok := parseLocationString(t.Location)
		list = append(list, PlaceTip{
			ID:       t.ID,
			Name:     t.Name,
			Address:  t.Address,
			District: t.District,
			Adcode:   t.Adcode,
			Lng:      lng,
			Lat:      lat,
			HasCoord: ok,
		})
	}
	return list, nil
}

// PlaceTextSearch POI 关键字搜索（适合「搜地名 → 选一条」；结果更偏完整 POI）
func PlaceTextSearch(ctx context.Context, key, keywords, city string, page, offset int) ([]PlaceTip, error) {
	keywords = strings.TrimSpace(keywords)
	if key == "" || keywords == "" {
		return nil, fmt.Errorf("key 或 keywords 为空")
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
	u, _ := url.Parse(placeTextURL)
	q := u.Query()
	q.Set("key", key)
	q.Set("keywords", keywords)
	if city != "" {
		q.Set("city", city)
	}
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
		Status string `json:"status"`
		Info   string `json:"info"`
		Pois   []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Address  string `json:"address"`
			Adname   string `json:"adname"`
			Cityname string `json:"cityname"`
			Adcode   string `json:"adcode"`
			Location string `json:"location"`
		} `json:"pois"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("解析地点搜索响应失败: %w", err)
	}
	if out.Status != "1" {
		return nil, fmt.Errorf("地点搜索失败: %s", out.Info)
	}
	list := make([]PlaceTip, 0, len(out.Pois))
	for _, p := range out.Pois {
		lng, lat, ok := parseLocationString(p.Location)
		addr := p.Address
		if addr == "" {
			addr = p.Adname
		}
		list = append(list, PlaceTip{
			ID:       p.ID,
			Name:     p.Name,
			Address:  addr,
			District: p.Cityname + p.Adname,
			Adcode:   p.Adcode,
			Lng:      lng,
			Lat:      lat,
			HasCoord: ok,
		})
	}
	return list, nil
}
