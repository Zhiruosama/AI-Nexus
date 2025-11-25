// Package logger 日志模块的实现
package logger

import (
	"fmt"
	"log"
	"time"

	"github.com/Zhiruosama/ai_nexus/internal/middleware"
	"github.com/gin-gonic/gin"
)

func formatLog(level string, requestID any, message string) string {
	return fmt.Sprintf("%s [%s] RequestID=%v %s",
		time.Now().Format("2006-01-02 15:04:05"),
		level,
		requestID,
		message,
	)
}

// Info 记录一条 Info 级别的日志
func Info(c *gin.Context, format string, args ...any) {
	requestID, _ := c.Get(middleware.RequestIDKey)
	message := fmt.Sprintf(format, args...)
	log.Println(formatLog("[INFO]", requestID, message))
}

// Warn 记录一条 Warn 级别的日志
func Warn(c *gin.Context, format string, args ...any) {
	requestID, _ := c.Get(middleware.RequestIDKey)
	message := fmt.Sprintf(format, args...)
	log.Println(formatLog("[WARN]", requestID, message))
}

// Error 记录一条 Error 级别的日志
func Error(c *gin.Context, format string, args ...any) {
	requestID, _ := c.Get(middleware.RequestIDKey)
	message := fmt.Sprintf(format, args...)
	log.Println(formatLog("[ERROR]", requestID, message))
}
