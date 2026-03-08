package hfut

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	eamServiceURL  = "https://cas.hfut.edu.cn/cas/login?service=http://jxglstu.hfut.edu.cn/eams5-student/neusoft-sso/login"
	studentInfoURL = "http://jxglstu.hfut.edu.cn/eams5-student/for-std/student-info"
)

// fetchStudentInfo 使用 CAS cookie 获取 EAM session，再拉取学生信息页并解析
func fetchStudentInfo(ctx context.Context, cookieStr string) (map[string]interface{}, error) {
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

	// 注入 CAS cookie 到 jar
	casURL, _ := url.Parse("https://cas.hfut.edu.cn")
	client.Jar.SetCookies(casURL, parseCookieString(cookieStr))

	// 1. 用 CAS cookie 访问 EAM service，跟随重定向获取 EAM session
	req1, _ := http.NewRequestWithContext(ctx, "GET", eamServiceURL, nil)
	req1.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res1, err := client.Do(req1)
	if err != nil {
		return nil, fmt.Errorf("EAM 登录失败: %w", err)
	}
	res1.Body.Close()

	// 2. 访问学生信息页（可能重定向到 /info/{studentCode}），jar 已含 EAM cookie
	req2, _ := http.NewRequestWithContext(ctx, "GET", studentInfoURL, nil)
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res2, err := client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("获取学生信息失败: %w", err)
	}
	defer res2.Body.Close()

	body, err := io.ReadAll(res2.Body)
	if err != nil {
		return nil, err
	}

	return parseStudentInfoHTML(string(body))
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

// parseStudentInfoHTML 解析 EAMS5 学生信息页 HTML，提取 key-value 到 map
func parseStudentInfoHTML(html string) (map[string]interface{}, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	info := make(map[string]interface{})

	// 常见结构：table tr 内 th/td 或 label+span，或 dl dt dd
	doc.Find("table.student-info tr, table tr, .info-table tr, .form-table tr").Each(func(_ int, tr *goquery.Selection) {
		label := strings.TrimSpace(tr.Find("th, td:first-child, .label, dt").First().Text())
		value := strings.TrimSpace(tr.Find("td:last-child, td:nth-child(2), .value, dd").First().Text())
		if label != "" && value != "" {
			key := toSnakeCase(label)
			if key != "" {
				info[key] = value
			}
		}
	})

	// 备用：dl dt dd
	doc.Find("dl").Each(func(_ int, dl *goquery.Selection) {
		dl.Find("dt").Each(func(i int, dt *goquery.Selection) {
			label := strings.TrimSpace(dt.Text())
			dd := dt.NextFiltered("dd")
			value := strings.TrimSpace(dd.First().Text())
			if label != "" && value != "" {
				key := toSnakeCase(label)
				if key != "" {
					info[key] = value
				}
			}
		})
	})

	// 映射常见中文标签到标准字段
	labelMap := map[string]string{
		"学号": "student_id", "姓名": "username_zh", "英文名": "username_en",
		"性别": "sex", "院系": "department", "专业": "major",
		"班级": "class", "校区": "campus", "学院": "department",
	}
	for cn, en := range labelMap {
		for k, v := range info {
			if k == cn || strings.Contains(k, cn) {
				info[en] = v
				break
			}
		}
	}

	return info, nil
}

func toSnakeCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32)
		} else if r > 127 {
			b.WriteString(s)
			return s
		}
	}
	return b.String()
}
