// Package middleware Recovery中间件 用于捕获全局的panic
package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	logFileMutex sync.Mutex
	logFilePath  = "./logs/error.log"
)

// PanicLog 结构化的 panic 日志
type PanicLog struct {
	// 基础信息
	Timestamp  string `json:"timestamp"`
	PanicValue string `json:"panic_value"`
	StackTrace string `json:"stack_trace"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	ClientIP   string `json:"client_ip"`
	UserAgent  string `json:"user_agent"`

	// gin 链信息
	RequestID string `json:"request_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
}

// Recovery 用于捕获所有 panic 并将详细信息落盘
func Recovery() gin.HandlerFunc {
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[RECOVERY] Failed to create painc dir: %v\n", err)
	}

	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				panicLog := PanicLog{
					Timestamp:  time.Now().Format("2006-01-02 15:04:05.000"),
					PanicValue: fmt.Sprintf("%v", r),
					StackTrace: string(debug.Stack()),
					Method:     c.Request.Method,
					Path:       c.Request.URL.Path,
					ClientIP:   c.ClientIP(),
					UserAgent:  c.Request.UserAgent(),
				}

				if requestID, exists := c.Get(RequestIDKey); exists {
					panicLog.RequestID = fmt.Sprintf("%v", requestID)
				}
				if userID, exists := c.Get(UserIDKey); exists {
					panicLog.UserID = fmt.Sprintf("%v", userID)
				}

				writeErrorLog(panicLog)

				if !c.Writer.Written() {
					c.JSON(http.StatusInternalServerError, gin.H{
						"code":    http.StatusInternalServerError,
						"message": "Server internal error",
						"request": fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
					})
				}
			}
		}()

		c.Next()
	}
}

// writeErrorLog 将 panic 日志写入文件
func writeErrorLog(log PanicLog) {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RECOVERY] Failed to open log file: %v\n", err)
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "[RECOVERY] Failed to close log file: %v\n", closeErr)
		}
	}()

	logBytes, err := json.Marshal(log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RECOVERY] Serialization log failed: %v\n", err)
		return
	}

	if _, err := file.Write(append(logBytes, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "[RECOVERY] Failed to write to log file: %v\n", err)
	}
}
