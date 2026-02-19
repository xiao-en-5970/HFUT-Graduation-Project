package image

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
)

// SmallSuffix 压缩图后缀
const SmallSuffix = ".small"

// CompressToSmall 将图片压缩到指定最大边长（720p=1280x720 的短边为 720，这里用最大边长）
// maxPx: 长边不超过此像素，如 720 或 540
// 从 srcPath 读取，写入 dstPath
func CompressToSmall(srcPath, dstPath string, maxPx uint) error {
	if maxPx == 0 {
		return fmt.Errorf("maxPx must be > 0")
	}
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	w, h := uint(bounds.Dx()), uint(bounds.Dy())
	if w <= maxPx && h <= maxPx {
		// 无需压缩，直接复制
		return copyFile(srcPath, dstPath)
	}

	resized := resize.Thumbnail(maxPx, maxPx, img, resize.Lanczos3)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}
	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return jpeg.Encode(out, resized, &jpeg.Options{Quality: 85})
	case "png":
		return png.Encode(out, resized)
	case "gif":
		// GIF 需特殊处理，resize 后可能丢失调色板，转为 PNG 更稳妥
		return png.Encode(out, resized)
	default:
		return jpeg.Encode(out, resized, &jpeg.Options{Quality: 85})
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// SmallPath 获取原路径对应的 .small 压缩图路径
func SmallPath(relPath string) string {
	if relPath == "" || strings.HasSuffix(relPath, SmallSuffix) {
		return relPath
	}
	return relPath + SmallSuffix
}

// StripSmallSuffix 去掉 .small 后缀得到原图路径
func StripSmallSuffix(path string) string {
	return strings.TrimSuffix(path, SmallSuffix)
}
