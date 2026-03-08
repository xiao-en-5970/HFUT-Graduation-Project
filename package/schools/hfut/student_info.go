package hfut

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/schools"
)

// fetchStudentInfo 使用 CAS cookie 获取 EAM session，再拉取学生信息（参考 hfut-api）
// opts 从 schools 表配置传入，eam_service_url、info_url 禁止写死；未配置时跳过
// 不解析响应，直接存响应体：若为 JSON 则原样存入 cert_info，否则以 {"raw": "..."} 形式存储
func fetchStudentInfo(ctx context.Context, cookieStr string, opts *schools.LoginOptions) (map[string]interface{}, error) {
	if opts == nil || opts.InfoURL == "" || opts.EAMServiceURL == "" || opts.CaptchaURL == "" {
		return nil, nil
	}
	casBase := deriveBaseFromURL(opts.CaptchaURL)
	if casBase == "" {
		return nil, nil
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: 15 * time.Second,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	casURL, _ := url.Parse(casBase)
	client.Jar.SetCookies(casURL, parseCookieString(cookieStr))

	// 1. 用 CAS cookie 访问 EAM service，跟随重定向获取 EAM session
	req1, _ := http.NewRequestWithContext(ctx, "GET", opts.EAMServiceURL, nil)
	req1.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res1, err := client.Do(req1)
	if err != nil {
		return nil, fmt.Errorf("EAM 登录失败: %w", err)
	}
	res1.Body.Close()

	// 2. 访问学生信息页（参考 hfut-api：首次可能 302 到 /info/{code}，从 Location 取 code）
	req2, _ := http.NewRequestWithContext(ctx, "GET", opts.InfoURL, nil)
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res2, err := client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("获取学生信息失败: %w", err)
	}
	defer res2.Body.Close()

	code := ""
	if res2.StatusCode == 302 {
		loc := res2.Header.Get("Location")
		if loc != "" {
			// Location 形如 /eams5-student/for-std/student-info/info/2020123456
			parts := strings.Split(strings.TrimSuffix(loc, "/"), "/")
			if len(parts) > 0 {
				code = parts[len(parts)-1]
			}
		}
	}

	// 3. 若有 code，请求 /info/{code} 获取完整信息（hfut-api 流程）
	var body []byte
	if code != "" {
		infoURL := strings.TrimSuffix(opts.InfoURL, "/") + "/info/" + code
		req3, _ := http.NewRequestWithContext(ctx, "GET", infoURL, nil)
		req3.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
		res3, err := client.Do(req3)
		if err != nil {
			return nil, fmt.Errorf("获取学生详情失败: %w", err)
		}
		defer res3.Body.Close()
		body, err = io.ReadAll(res3.Body)
		if err != nil {
			return nil, err
		}
	} else {
		body, err = io.ReadAll(res2.Body)
		if err != nil {
			return nil, err
		}
	}

	return rawBodyToCertInfo(body)
}

// rawBodyToCertInfo 将响应体原样存入 cert_info：若为有效 JSON 则解析后返回，否则以 {"raw": "..."} 存储
func rawBodyToCertInfo(body []byte) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err == nil && m != nil {
		return m, nil
	}
	return map[string]interface{}{"raw": string(body)}, nil
}

func parseCookieString(s string) []*http.Cookie {
	var cookies []*http.Cookie
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx <= 0 {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name:  strings.TrimSpace(part[:idx]),
			Value: strings.TrimSpace(part[idx+1:]),
		})
	}
	return cookies
}
