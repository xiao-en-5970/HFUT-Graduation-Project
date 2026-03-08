//go:build tesseract
// +build tesseract

package hfut

import (
	"os"
	"regexp"
	"strings"

	"github.com/otiai10/gosseract/v2"
)

func recognizeCaptcha(imgData []byte) (string, error) {
	tmp, err := os.CreateTemp("", "hfut-captcha-*.png")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.Write(imgData); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()

	client := gosseract.NewClient()
	defer client.Close()
	client.SetImage(path)
	client.SetLanguage("eng")
	client.SetWhitelist("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	text, err := client.Text()
	if err != nil {
		return "", err
	}
	// 清理：去空格、非字母数字，取前4位
	re := regexp.MustCompile(`[^A-Za-z0-9]`)
	cleaned := re.ReplaceAllString(text, "")
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) > 4 {
		cleaned = cleaned[:4]
	}
	if cleaned == "" {
		cleaned = "0000"
	}
	return cleaned, nil
}
