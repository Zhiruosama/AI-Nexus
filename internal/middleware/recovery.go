// Package middleware Recovery 中间件 用于捕获全局的panic
package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery 于捕获所有 panic
func Recovery() gin.HandlerFunc {
	log.Println("Recovery Middleware initialized")

	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// r是panic传出的信息
				log.Printf("PANIC RECOVERED!\n")
				log.Printf("Panic Value: %v\n", r)
				// 打印堆栈信息
				log.Printf("Stack Trace:\n%s\n", debug.Stack())
				log.Printf("--------------------------------\n")

				// 向客户端发送统一的错误响应
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"messgae": "服务器内部错误",
					"request": fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
				})

				// 终止后续处理链
				c.Abort()
			}
		}()

		c.Next()
	}
}
