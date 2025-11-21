package pkg

import (
	"fmt"
	"image"

	// 导入图片解码器以支持 gif, jpeg, png 格式
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
)

// ProcessImageToWebP 异步处理图片转webp格式并压缩
func ProcessImageToWebP(ctx *gin.Context, srcPath string, quality int) bool {
	file, err := os.Open(srcPath)
	if err != nil {
		logger.Error(ctx, "Open source image error: %s", err.Error())
		return false
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logger.Error(ctx, "Close source file error: %s", closeErr.Error())
		}
	}()

	img, _, err := image.Decode(file)
	if err != nil {
		logger.Error(ctx, "Decode image error: %s", err.Error())
		return false
	}

	ext := filepath.Ext(srcPath)
	webpPath := strings.TrimSuffix(srcPath, ext) + ".webp"

	outFile, err := os.Create(webpPath)
	if err != nil {
		logger.Error(ctx, "Create webp file error: %s", err.Error())
		return false
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			logger.Error(ctx, "Close output file error: %s", closeErr.Error())
		}
	}()

	if quality <= 0 || quality > 100 {
		quality = 80
	}

	err = webp.Encode(outFile, img, &webp.Options{
		Lossless: false,
		Quality:  float32(quality),
	})
	if err != nil {
		logger.Error(ctx, "Encode webp error: %s", err.Error())
		return false
	}

	if err := os.Remove(srcPath); err != nil {
		logger.Error(ctx, "Remove original file error: %s", err.Error())
		return false
	}
	return true
}

// DownloadAndSaveImages 从URL切片下载图片并保存到本地,转换为WebP格式
func DownloadAndSaveImages(imgURL string, quality int) (string, error) {
	if len(imgURL) == 0 {
		return "", fmt.Errorf("image URL is empty")
	}

	savePath := "static/images/"

	// 确保保存目录存在
	if err := os.MkdirAll(savePath, 0755); err != nil {
		return "", fmt.Errorf("create save directory error: %w", err)
	}

	var outputPath string
	client := &http.Client{}

	// 下载图片
	resp, err := client.Get(imgURL)
	if err != nil {
		return "", fmt.Errorf("download image error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errs := resp.Body.Close()
		if errs != nil {
			logger.Error(nil, "Close response body error: %s", errs.Error())
		}
		return "", fmt.Errorf("download image failed with status: %d", resp.StatusCode)
	}

	// 从URL中提取文件名
	urlPath := filepath.Base(imgURL)
	ext := filepath.Ext(urlPath)
	if ext == "" {
		ext = ".png"
	}

	// 生成临时文件路径
	tempFilePath := filepath.Join(savePath, urlPath)
	if filepath.Ext(tempFilePath) == "" {
		tempFilePath += ext
	}

	// 保存临时图片
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		errs := resp.Body.Close()
		if errs != nil {
			logger.Error(nil, "Close response body error: %s", errs.Error())
		}
		return "", fmt.Errorf("create temp file error: %w", err)
	}

	_, err = io.Copy(tempFile, resp.Body)
	resp.Body.Close()
	tempFile.Close()

	if err != nil {
		errs := os.Remove(tempFilePath)
		if errs != nil {
			logger.Error(nil, "Remove temp file error: %s", errs.Error())
		}
		return "", fmt.Errorf("save image to temp file error: %w", err)
	}

	// 转换为WebP
	file, err := os.Open(tempFilePath)
	if err != nil {
		errs := os.Remove(tempFilePath)
		if errs != nil {
			logger.Error(nil, "Remove temp file error: %s", errs.Error())
		}
		return "", fmt.Errorf("open temp file error: %w", err)
	}

	img, _, err := image.Decode(file)
	file.Close()
	if err != nil {
		errs := os.Remove(tempFilePath)
		if errs != nil {
			logger.Error(nil, "Remove temp file error: %s", errs.Error())
		}
		return "", fmt.Errorf("decode image error: %w", err)
	}

	// 生成WebP文件路径
	webpPath := strings.TrimSuffix(tempFilePath, filepath.Ext(tempFilePath)) + ".webp"
	outFile, err := os.Create(webpPath)
	if err != nil {
		errs := os.Remove(tempFilePath)
		if errs != nil {
			logger.Error(nil, "Remove temp file error: %s", errs.Error())
		}
		return "", fmt.Errorf("create webp file error: %w", err)
	}

	if quality <= 0 || quality > 100 {
		quality = 80
	}

	err = webp.Encode(outFile, img, &webp.Options{
		Lossless: false,
		Quality:  float32(quality),
	})
	outFile.Close()

	if err != nil {
		os.Remove(tempFilePath)
		os.Remove(webpPath)
		return "", fmt.Errorf("encode webp error: %w", err)
	}

	// 删除原始临时文件
	os.Remove(tempFilePath)

	outputPath = webpPath

	return outputPath, nil
}
