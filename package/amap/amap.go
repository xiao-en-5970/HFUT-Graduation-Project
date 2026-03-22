// Package amap 封装高德地图 Web 服务 API：地理编码、两点间步行规划距离
// 文档：https://lbs.amap.com/api/webservice/guide/api/georegeo 、https://lbs.amap.com/api/webservice/guide/api/distance
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
	"time"
)

const (
	geocodeURL  = "https://restapi.amap.com/v3/geocode/geo"
	distanceURL = "https://restapi.amap.com/v3/distance"
)

var httpClient = &http.Client{Timeout: 12 * time.Second}

// GeocodeResult 经纬度（高德坐标系 GCJ-02）
type GeocodeResult struct {
	Lng float64
	Lat float64
}

// Geocode 将地址解析为经纬度
func Geocode(ctx context.Context, key, address string) (*GeocodeResult, error) {
	address = strings.TrimSpace(address)
	if address == "" || key == "" {
		return nil, fmt.Errorf("地址或 key 为空")
	}
	u, _ := url.Parse(geocodeURL)
	q := u.Query()
	q.Set("key", key)
	q.Set("address", address)
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
		Status   string `json:"status"`
		Info     string `json:"info"`
		InfoCode string `json:"infocode"`
		Geocodes []struct {
			Location string `json:"location"`
		} `json:"geocodes"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("解析地理编码响应失败: %w", err)
	}
	if out.Status != "1" {
		return nil, fmt.Errorf("地理编码失败: %s (%s)", out.Info, out.InfoCode)
	}
	if len(out.Geocodes) == 0 || out.Geocodes[0].Location == "" {
		return nil, fmt.Errorf("未解析到坐标，请检查地址是否完整")
	}
	parts := strings.Split(out.Geocodes[0].Location, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("坐标格式异常")
	}
	lng, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return nil, err
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return nil, err
	}
	return &GeocodeResult{Lng: lng, Lat: lat}, nil
}

// WalkingDistanceMeters 步行规划距离（米）。高德 distance 接口 type=3 为步行
func WalkingDistanceMeters(ctx context.Context, key string, from, to *GeocodeResult) (int, error) {
	if key == "" || from == nil || to == nil {
		return 0, fmt.Errorf("参数不完整")
	}
	origins := fmt.Sprintf("%.6f,%.6f", from.Lng, from.Lat)
	destination := fmt.Sprintf("%.6f,%.6f", to.Lng, to.Lat)
	u, _ := url.Parse(distanceURL)
	q := u.Query()
	q.Set("key", key)
	q.Set("origins", origins)
	q.Set("destination", destination)
	q.Set("type", "3") // 3：步行规划距离（米）
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
	var out struct {
		Status   string `json:"status"`
		Info     string `json:"info"`
		InfoCode string `json:"infocode"`
		Results  []struct {
			Distance string `json:"distance"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, fmt.Errorf("解析距离响应失败: %w", err)
	}
	if out.Status != "1" {
		return 0, fmt.Errorf("距离查询失败: %s (%s)", out.Info, out.InfoCode)
	}
	if len(out.Results) == 0 {
		return 0, fmt.Errorf("无距离结果")
	}
	d, err := strconv.Atoi(strings.TrimSpace(out.Results[0].Distance))
	if err != nil {
		return 0, fmt.Errorf("距离数值无效: %w", err)
	}
	return d, nil
}

// DistanceBetweenAddresses 根据两段文字地址计算步行规划距离（米）。需配置高德 Web 服务 Key。
func DistanceBetweenAddresses(ctx context.Context, key, senderAddr, receiverAddr string) (meters int, err error) {
	senderAddr = strings.TrimSpace(senderAddr)
	receiverAddr = strings.TrimSpace(receiverAddr)
	if key == "" {
		return 0, fmt.Errorf("未配置 AMAP_KEY")
	}
	if senderAddr == "" || receiverAddr == "" {
		return 0, fmt.Errorf("发货地址与收货地址均需填写才能计算距离")
	}
	from, err := Geocode(ctx, key, senderAddr)
	if err != nil {
		return 0, fmt.Errorf("发货地地理编码: %w", err)
	}
	to, err := Geocode(ctx, key, receiverAddr)
	if err != nil {
		return 0, fmt.Errorf("收货地地理编码: %w", err)
	}
	return WalkingDistanceMeters(ctx, key, from, to)
}
