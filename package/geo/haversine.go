// Package geo 地理辅助（WGS84）
package geo

import "math"

// HaversineMeters 两点球面大圆距离（米），WGS84 经纬度。
func HaversineMeters(lat1, lng1, lat2, lng2 float64) int {
	const earthR = 6371000.0
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return int(earthR*c + 0.5)
}
