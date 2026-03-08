//go:build !tesseract
// +build !tesseract

package hfut

import (
	"fmt"
)

func recognizeCaptcha(imgData []byte) (string, error) {
	return "", fmt.Errorf("验证码识别需要 tesseract：编译时加 -tags tesseract，并安装 tesseract-ocr（brew install tesseract / apt install tesseract-ocr）")
}
