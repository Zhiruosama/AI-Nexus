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
	mu             sync.RWMutex
)

// init 初始化 gRPC 连接和客户端存根
func init() {
	serverAddress = configs.GlobalConfig.GRPCClient.ServerAddress
	defaultTimeout = configs.GlobalConfig.GRPCClient.DefaultTimeout

	clientOnce.Do(func() {
		// 懒加载服务
		conn, initErr = grpc.NewClient(
			serverAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)

		if initErr != nil {
			log.Printf("[gRPC] Create gRPC client failed: %v\n", initErr)
			return
		}

		serviceClient = NewVarifyServiceClient(conn)
		log.Printf("[gRPC] Create gRPC client success: %s\n", serverAddress)
	})
}

// GetVerificationCode 调用gRPC外部方法
func GetVerificationCode(email string) (rspErrCode int32, rspEmail string, rspCode string, err error) {
	mu.RLock()
	if initErr != nil {
		mu.RUnlock()
		return 0, "", "", fmt.Errorf("gRPC Client initialization failed: %w", initErr)
	}
	if serviceClient == nil {
		mu.RUnlock()
		return 0, "", "", fmt.Errorf("gRPC Client is not initialized")
	}
	client := serviceClient
	mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	req := &GetVarifyReq{
		Email: email,
	}

	rsp, grpcErr := client.GetVarifyCode(ctx, req)
	if grpcErr != nil {
		return 0, "", "", fmt.Errorf("gRPC 调用 GetVarifyCode 失败: %w", grpcErr)
	}

	return rsp.GetError(), rsp.GetEmail(), rsp.GetCode(), nil
}
