// Package grpc grpc核心文件
package grpc

import (
	"context"
	"fmt"
	"log"
	sync "sync"
	"time"

	"github.com/Zhiruosama/ai_nexus/configs"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddress  string
	defaultTimeout time.Duration
	clientOnce     sync.Once
	conn           *grpc.ClientConn
	serviceClient  VarifyServiceClient
	initErr        error
)

// init 初始化 gRPC 连接和客户端存根
func init() {
	serverAddress = configs.GlobalConfig.GRPCClient.ServerAddress
	defaultTimeout = configs.GlobalConfig.GRPCClient.DefaultTimeout

	clientOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		done := make(chan struct{})
		go func() {
			conn, initErr = grpc.NewClient(
				serverAddress,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			close(done)
		}()

		select {
		case <-ctx.Done():
			initErr = fmt.Errorf(
				"gRPC 连接超时 at %s: %w",
				serverAddress,
				ctx.Err(),
			)

		case <-done: // NewClient 完成
			// 如果 initErr 已经在 goroutine 中被设置，它将被保留
		}

		if initErr != nil {
			return
		}

		// 创建存根 (Stub)
		serviceClient = NewVarifyServiceClient(conn)
		log.Printf("[gRPC Client] 已成功连接到: %s (Insecure)", serverAddress)
	})
}

// GetVerificationCode 调用gRPC外部方法
func GetVerificationCode(email string) (rspErrCode int32, rspEmail string, rspCode string, err error) {
	// 设置调用的上下文 (Context)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// 构造请求
	req := &GetVarifyReq{
		Email: email,
	}

	// 调用方法
	rsp, grpcErr := serviceClient.GetVarifyCode(ctx, req)
	if grpcErr != nil {
		return 0, "", "", fmt.Errorf("gRPC 调用 GetVarifyCode 失败: %w", grpcErr)
	}

	// 返回 GetVarifyRsp 中的所有数据
	return rsp.GetError(), rsp.GetEmail(), rsp.GetCode(), nil
}
