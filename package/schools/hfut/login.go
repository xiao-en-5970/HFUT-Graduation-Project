package hfut

import (
	"bytes"
	"context"
	"crypto/aes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/schools"
)

// deriveBaseFromURL 从 captcha_url 推导 CAS 根地址，如 https://cas.hfut.edu.cn/cas/vercode -> https://cas.hfut.edu.cn
func deriveBaseFromURL(captchaURL string) string {
	u, err := url.Parse(captchaURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

const (
	code = "hfut"
	name = "合肥工业大学"
)

// HFUT 合肥工业大学 CAS 登录，需验证码（前端先调 GetCaptcha 获取）
// login_url、captcha_url 必须从 schools 表配置，禁止写死
type HFUT struct{}

func (h *HFUT) Code() string { return code }
func (h *HFUT) Name() string { return name }

// GetCaptcha 获取验证码图片及 token，opts 从 schools 表配置传入
func (h *HFUT) GetCaptcha(ctx context.Context, opts *schools.CaptchaOptions) (image []byte, token string, err error) {
	if opts == nil || opts.LoginURL == "" || opts.CaptchaURL == "" {
		return nil, "", fmt.Errorf("未配置 login_url 或 captcha_url，请先在学校管理中配置")
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	jar := &cookieJar{cookies: make(map[string]string)}

	req1, _ := http.NewRequestWithContext(ctx, "GET", opts.LoginURL, nil)
	req1.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res1, err := client.Do(req1)
	if err != nil {
		return nil, "", fmt.Errorf("请求登录页失败: %w", err)
	}
	res1.Body.Close()
	jar.collect(res1.Cookies())
	cookieStr := jar.string()
	if cookieStr == "" {
		return nil, "", fmt.Errorf("登录太过频繁，请稍后再试")
	}

	req2, _ := http.NewRequestWithContext(ctx, "GET", opts.CaptchaURL, nil)
	req2.Header.Set("Cookie", cookieStr)
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res2, err := client.Do(req2)
	if err != nil {
		return nil, "", fmt.Errorf("获取验证码失败: %w", err)
	}
	imgData, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	jar.collect(res2.Cookies())
	cookieStr = jar.string()

	token, err = schools.StoreCaptchaSession(ctx, code, cookieStr)
	if err != nil {
		return nil, "", err
	}
	return imgData, token, nil
}

// Login 需 captcha 和 captchaToken（由 GetCaptcha 返回），opts 从 schools 表配置传入
func (h *HFUT) Login(ctx context.Context, username, password, captcha, captchaToken string, opts *schools.LoginOptions) (*schools.LoginResult, error) {
	if username == "" || password == "" {
		return &schools.LoginResult{Success: false, Message: "账号或密码不能为空"}, nil
	}
	if captcha == "" || captchaToken == "" {
		return &schools.LoginResult{Success: false, Message: "请先获取验证码并填写"}, nil
	}
	if opts == nil || opts.LoginURL == "" || opts.CaptchaURL == "" {
		return &schools.LoginResult{Success: false, Message: "未配置 login_url 或 captcha_url，请先在学校管理中配置"}, nil
	}

	cookieStr, err := schools.GetCaptchaSession(ctx, code, captchaToken)
	if err != nil {
		return &schools.LoginResult{Success: false, Message: err.Error()}, nil
	}
	if cookieStr == "" {
		return &schools.LoginResult{Success: false, Message: "验证码会话无效或已过期，请重新获取验证码"}, nil
	}

	casBase := deriveBaseFromURL(opts.CaptchaURL)
	if casBase == "" {
		return &schools.LoginResult{Success: false, Message: "captcha_url 格式无效，无法推导 CAS 地址"}, nil
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	loginURL := opts.LoginURL

	jar := &cookieJar{cookies: make(map[string]string)}
	jar.collect(parseCookieString(cookieStr))

	// checkInitVercode 获取加密盐
	req4, _ := http.NewRequestWithContext(ctx, "GET", casBase+"/cas/checkInitVercode?_="+fmt.Sprint(time.Now().UnixMilli()), nil)
	req4.Header.Set("Cookie", cookieStr)
	req4.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res4, err := client.Do(req4)
	if err != nil {
		return nil, fmt.Errorf("获取加密参数失败: %w", err)
	}
	res4.Body.Close()
	jar.collect(res4.Cookies())
	cookieStr = jar.string()

	salt := jar.get("LOGIN_FLAVORING")
	if salt == "" {
		return nil, fmt.Errorf("无法获取加密盐值")
	}

	encPwd, err := encryptPassword(password, salt)
	if err != nil {
		return nil, err
	}

	// checkUserIdenty 验证身份（cas_base 从 captcha_url 推导）
	checkURL := casBase + "/cas/policy/checkUserIdenty?" + url.Values{
		"username": {username},
		"password": {encPwd},
		"capcha":   {captcha},
		"_":        {fmt.Sprint(time.Now().UnixMilli())},
	}.Encode()
	req5, _ := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
	req5.Header.Set("Cookie", cookieStr)
	req5.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res5, err := client.Do(req5)
	if err != nil {
		return nil, fmt.Errorf("验证身份失败: %w", err)
	}
	body5, _ := io.ReadAll(res5.Body)
	res5.Body.Close()

	var checkResp struct {
		Data struct {
			AuthFlag bool   `json:"authFlag"`
			Message  string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body5, &checkResp); err == nil && !checkResp.Data.AuthFlag {
		msg := strings.TrimSpace(checkResp.Data.Message)
		if msg == "" {
			msg = "账号或密码错误"
		}
		return &schools.LoginResult{Success: false, Message: msg}, nil
	}

	// POST 提交登录
	form := url.Values{
		"username":    {username},
		"password":    {encPwd},
		"capcha":      {captcha},
		"execution":   {"e1s1"},
		"_eventId":    {"submit"},
		"geolocation": {""},
	}.Encode()
	req6, _ := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewBufferString(form))
	req6.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req6.Header.Set("Cookie", cookieStr)
	req6.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res6, err := client.Do(req6)
	if err != nil {
		return nil, fmt.Errorf("提交登录失败: %w", err)
	}
	body6, _ := io.ReadAll(res6.Body)
	res6.Body.Close()

	if res6.StatusCode == 302 {
		loc := res6.Header.Get("Location")
		if strings.Contains(loc, "ticket=") {
			res := &schools.LoginResult{Success: true, StudentID: username}
			if certInfo, err := fetchStudentInfo(ctx, jar.string()); err == nil && len(certInfo) > 0 {
				res.CertInfo = certInfo
				if n, ok := certInfo["username_zh"].(string); ok && n != "" {
					res.Name = n
				} else if n, ok := certInfo["姓名"].(string); ok && n != "" {
					res.Name = n
				}
			}
			return res, nil
		}
	}
	// 解析 CAS 返回的错误信息（参考 hfut-api：HTML 中 #errorpassword、#errorcode、.alert-error）
	msg := parseLoginErrorFromBody(body6)
	if msg == "" {
		msg = "账号或密码错误"
	}
	return &schools.LoginResult{Success: false, Message: msg}, nil
}

// parseLoginErrorFromBody 从 CAS 登录失败响应中解析错误信息（HTML 或 JSON）
func parseLoginErrorFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	// 尝试解析 JSON
	var jsonResp struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &jsonResp); err == nil {
		if s := strings.TrimSpace(jsonResp.Message); s != "" {
			return s
		}
		if s := strings.TrimSpace(jsonResp.Error); s != "" {
			return s
		}
	}
	// 解析 HTML：参考 hfut-api 的 #errorpassword、#errorcode、.alert-error
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return ""
	}
	var msg string
	doc.Find("#errorpassword, #errorcode, .alert-error, .error, [id^=error]").Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t != "" && msg == "" {
			msg = t
		}
	})
	if msg != "" {
		// 去除多余空白
		msg = regexp.MustCompile(`\s+`).ReplaceAllString(msg, " ")
		return strings.TrimSpace(msg)
	}
	return ""
}

func encryptPassword(pwd, salt string) (string, error) {
	key := []byte(salt)
	if len(key) < 16 {
		key = append(key, make([]byte, 16-len(key))...)
	} else if len(key) > 16 {
		key = key[:16]
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	padded := pkcs7Pad([]byte(pwd), block.BlockSize())
	dst := make([]byte, len(padded))
	for i := 0; i < len(padded); i += block.BlockSize() {
		block.Encrypt(dst[i:i+block.BlockSize()], padded[i:i+block.BlockSize()])
	}
	return base64.StdEncoding.EncodeToString(dst), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	n := blockSize - len(data)%blockSize
	for i := 0; i < n; i++ {
		data = append(data, byte(n))
	}
	return data
}

type cookieJar struct {
	cookies map[string]string
}

func (j *cookieJar) collect(cs []*http.Cookie) {
	for _, c := range cs {
		j.cookies[c.Name] = c.Value
	}
}

func (j *cookieJar) get(name string) string {
	return j.cookies[name]
}

func (j *cookieJar) string() string {
	var parts []string
	for k, v := range j.cookies {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}
