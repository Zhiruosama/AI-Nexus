package pkg

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
)

// ProcessImageToWebP 异步处理图片转webp格式并压缩
func ProcessImageToWebP(ctx *gin.Context, srcPath string, quality int) {
	go func() {
		file, err := os.Open(srcPath)
		if err != nil {
			logger.Error(ctx, "Open source image error: %s", err.Error())
			return
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			logger.Error(ctx, "Decode image error: %s", err.Error())
			return
		}

		ext := filepath.Ext(srcPath)
		webpPath := strings.TrimSuffix(srcPath, ext) + ".webp"

		outFile, err := os.Create(webpPath)
		if err != nil {
			logger.Error(ctx, "Create webp file error: %s", err.Error())
			return
		}
		defer outFile.Close()

		if quality <= 0 || quality > 100 {
			quality = 80
		}

		err = webp.Encode(outFile, img, &webp.Options{
			Lossless: false,
			Quality:  float32(quality),
		})
		if err != nil {
			logger.Error(ctx, "Encode webp error: %s", err.Error())
			return
		}

		if err := os.Remove(srcPath); err != nil {
			logger.Error(ctx, "Remove original file error: %s", err.Error())
			return
		}
	}()
}
