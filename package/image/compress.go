package image

// image.Decode 需导入格式包才会注册解码器，否则报 "unknown format"。支持的格式应与 oss.imageExts 一致
import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // 注册解码器，image.Decode 需导入格式包才能识别
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/webp" // 同上，imageExts 支持的格式需全部注册

	"github.com/nfnt/resize"
	"golang.org/x/image/draw"
)

// SmallSuffix 压缩图后缀
const SmallSuffix = ".small"

// CompressToSmall 将图片压缩到指定最大边长，体积尽量不超过 maxKB
// maxPx: 长边不超过此像素；maxKB: 体积上限（KB），如 200
func CompressToSmall(srcPath, dstPath string, maxPx uint, maxKB int) error {
	if maxPx == 0 {
		return fmt.Errorf("maxPx must be > 0")
	}
	if maxKB <= 0 {
		maxKB = 200
	}
	maxBytes := maxKB * 1024
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	// 统一转为可 JPEG 编码的格式（处理透明通道）
	img = flattenAlpha(img)

	bounds := img.Bounds()
	w, h := uint(bounds.Dx()), uint(bounds.Dy())
	if w > maxPx || h > maxPx {
		img = resize.Thumbnail(maxPx, maxPx, img, resize.Lanczos3)
	}

	// 迭代降质直至体积不超过 200KB
	qualities := []int{78, 65, 52, 40, 30, 22}
	var best []byte
	var bestQ int
	for _, q := range qualities {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return err
		}
		if buf.Len() <= maxBytes {
			best = buf.Bytes()
			bestQ = q
			break
		}
		best = buf.Bytes()
		bestQ = q
	}

	// 若仍超限，缩小尺寸再压
	if len(best) > maxBytes && (img.Bounds().Dx() > 200 || img.Bounds().Dy() > 200) {
		sz := img.Bounds().Size()
		nw, nh := sz.X*3/4, sz.Y*3/4
		if nw < 200 {
			nw = 200
		}
		if nh < 200 {
			nh = 200
		}
		img = resize.Resize(uint(nw), uint(nh), img, resize.Lanczos3)
		for _, q := range []int{65, 50, 35} {
			var buf bytes.Buffer
			if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
				return err
			}
			if buf.Len() <= maxBytes {
				best = buf.Bytes()
				break
			}
			best = buf.Bytes()
		}
	}

	if best == nil {
		best, _ = encodeJPEG(img, 35)
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}
	_ = bestQ
	return os.WriteFile(dstPath, best, 0644)
}

func encodeJPEG(img image.Image, q int) ([]byte, error) {
	var b bytes.Buffer
	err := jpeg.Encode(&b, img, &jpeg.Options{Quality: q})
	return b.Bytes(), err
}

// flattenAlpha 将带透明通道的图片绘制到白色底上，供 JPEG 编码（PNG/GIF 透明 → 白底）
func flattenAlpha(img image.Image) image.Image {
	// YCbCr 等无透明通道，直接返回
	if _, ok := img.(*image.YCbCr); ok {
		return img
	}
	if _, ok := img.(*image.Gray); ok {
		return img
	}
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, &image.Uniform{C: color.White}, bounds.Min, draw.Src)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Over)
	return dst
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
