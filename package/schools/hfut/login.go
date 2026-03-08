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
	"strings"
	"time"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/schools"
)

const (
	code       = "hfut"
	name       = "合肥工业大学"
	casBase    = "https://cas.hfut.edu.cn"
	loginPath  = "/cas/login"
	vercodePath = "/cas/vercode"
)

// HFUT 合肥工业大学 CAS 登录
type HFUT struct{}

func (h *HFUT) Code() string { return code }
func (h *HFUT) Name() string { return name }

// Login 仅需账号密码，内部自动处理验证码（OCR）
func (h *HFUT) Login(ctx context.Context, username, password string) (*schools.LoginResult, error) {
	if username == "" || password == "" {
		return &schools.LoginResult{Success: false, Message: "账号或密码不能为空"}, nil
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	const maxAttempts = 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		res, err := h.doLogin(ctx, client, username, password)
		if err != nil {
			return &schools.LoginResult{Success: false, Message: err.Error()}, nil
		}
		if res != nil {
			return res, nil
		}
		// res==nil 表示验证码错误，重试
		time.Sleep(time.Second)
	}
	return &schools.LoginResult{Success: false, Message: "验证码识别失败，请稍后重试"}, nil
}

func (h *HFUT) doLogin(ctx context.Context, client *http.Client, username, password string) (*schools.LoginResult, error) {
	loginURL := casBase + loginPath + "?service=https%3A%2F%2Fcas.hfut.edu.cn%2Fcas%2Foauth2.0%2FcallbackAuthorize%3Fclient_id%3DBsHfutEduPortal%26redirect_uri%3Dhttps%253A%252F%252Fone.hfut.edu.cn%252Fhome%252Findex%26response_type%3Dcode%26client_name%3DCasOAuthClient"

	jar := &cookieJar{cookies: make(map[string]string)}

	// 1. 获取登录页，拿到 session
	req1, _ := http.NewRequestWithContext(ctx, "GET", loginURL, nil)
	req1.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res1, err := client.Do(req1)
	if err != nil {
		return nil, fmt.Errorf("请求登录页失败: %w", err)
	}
	res1.Body.Close()
	jar.collect(res1.Cookies())
	cookieStr := jar.string()

	if cookieStr == "" {
		return &schools.LoginResult{Success: false, Message: "登录太过频繁，请稍后再试"}, nil
	}

	// 2. 获取验证码图片
	req2, _ := http.NewRequestWithContext(ctx, "GET", casBase+vercodePath, nil)
	req2.Header.Set("Cookie", cookieStr)
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/100.0.4896.75")
	res2, err := client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("获取验证码失败: %w", err)
	}
	imgData, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	jar.collect(res2.Cookies())
	cookieStr = jar.string()

	// 3. OCR 识别验证码
	captcha, err := recognizeCaptcha(imgData)
	if err != nil {
		return nil, fmt.Errorf("验证码识别失败: %w", err)
	}

	// 4. checkInitVercode 获取加密盐
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

	// 5. checkUserIdenty 验证身份
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
		msg := checkResp.Data.Message
		if strings.Contains(msg, "验证码") || strings.Contains(msg, "不正确") {
			return nil, nil // 验证码错误，返回 nil 触发重试
		}
		return &schools.LoginResult{Success: false, Message: msg}, nil
	}

	// 6. POST 提交登录
	form := url.Values{
		"username":    {username},
		"password":   {encPwd},
		"capcha":     {captcha},
		"execution":  {"e1s1"},
		"_eventId":   {"submit"},
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
	res6.Body.Close()

	if res6.StatusCode == 302 {
		loc := res6.Header.Get("Location")
		if strings.Contains(loc, "ticket=") {
			return &schools.LoginResult{
				Success:   true,
				StudentID: username,
			}, nil
		}
	}
	return &schools.LoginResult{Success: false, Message: "登录失败"}, nil
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
