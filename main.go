package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func main() {
	// 1. 定义要计算哈希值的文件路径
	filePath := "./static/avatar/default.png"

	// 2. 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", filePath, err)
		return // 终止程序
	}
	// 3. 确保文件句柄在函数返回时关闭
	defer file.Close()

	// 4. 初始化 SHA256 哈希对象
	hash := sha256.New()

	// 5. 将文件内容复制到哈希对象中进行计算
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Printf("Error calculating sha256 for file %s: %v\n", filePath, err)
		return // 终止程序
	}

	// 6. 获取哈希结果（字节切片）并转换为十六进制字符串
	calculatedHash := hex.EncodeToString(hash.Sum(nil))

	// 7. 打印哈希值 (这是你唯一需要的输出)
	fmt.Println(calculatedHash)

	// 其他与你的原始逻辑相关的代码已移除（如 dto.Sha256 比较、os.Remove、webp 转换等）。
}
