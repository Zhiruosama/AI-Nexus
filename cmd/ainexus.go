// Package main 应用的入口点，初始化 Config 和 DB 配置
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/Zhiruosama/ai_nexus/configs"
	app "github.com/Zhiruosama/ai_nexus/internal"
	"github.com/Zhiruosama/ai_nexus/internal/mailcallback"
	"github.com/Zhiruosama/ai_nexus/internal/mailgateway"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/db"
	rabbitmq "github.com/Zhiruosama/ai_nexus/internal/pkg/queue"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	websocket "github.com/Zhiruosama/ai_nexus/internal/pkg/ws"
	user_service "github.com/Zhiruosama/ai_nexus/internal/service/user"
	"github.com/Zhiruosama/ai_nexus/internal/verification"
)

func main() {
	// 命令行参数解析
	port := flag.Int("p", 0, "服务端口号，不指定则使用配置文件中的端口")
	flag.Parse()

	// 如果指定了端口，覆盖配置文件中的端口
	if *port > 0 {
		configs.GlobalConfig.Server.Port = *port
		log.Printf("[Main] 使用命令行指定端口: %d\n", *port)
	}
	defer rabbitmq.GlobalMQ.Close()
	defer websocket.GlobalHub.Close()

	mailCfg := configs.GlobalConfig.Mail
	mailClient, err := mailgateway.NewClient(mailgateway.Config{
		Address: mailCfg.Address, Timeout: mailCfg.Timeout, AllowInsecure: mailCfg.AllowInsecure,
		TLSCAFile: mailCfg.TLSCAFile, TLSServerName: mailCfg.TLSServerName,
		SenderIdentityKey: mailCfg.SenderIdentityKey, Locale: mailCfg.Locale,
		DispatchDeadline: mailCfg.DispatchDeadline,
	})
	if err != nil {
		log.Fatalf("[Main] Failed to initialize Mail Service client: %v\n", err)
	}
	defer mailClient.Close()

	verificationService, err := verification.NewService(
		verification.NewStore(db.GlobalDB), mailClient, rdb.Rdb,
		verification.Config{
			ValidFor: mailCfg.VerificationTTL, PendingTTL: mailCfg.PendingTTL,
			Cooldown: mailCfg.Cooldown, MaxAttempts: uint32(mailCfg.MaxAttempts),
			HMACSecret: mailCfg.HMACSecret, FingerprintSecret: mailCfg.EmailFingerprintSecret,
		},
	)
	if err != nil {
		log.Fatalf("[Main] Failed to initialize verification service: %v\n", err)
	}
	callbackServer, err := mailcallback.NewServer(mailcallback.Config{
		Address: mailCfg.CallbackAddress, AllowInsecure: mailCfg.CallbackAllowInsecure,
		TLSCertFile: mailCfg.CallbackTLSCertFile, TLSKeyFile: mailCfg.CallbackTLSKeyFile,
	}, verificationService)
	if err != nil {
		log.Fatalf("[Main] Failed to initialize mail callback server: %v\n", err)
	}
	defer callbackServer.Stop()
	go func() {
		log.Printf("[MailCallback] gRPC callback server listening on %s\n", mailCfg.CallbackAddress)
		if serveErr := callbackServer.Serve(); serveErr != nil {
			log.Printf("[MailCallback] server stopped unexpectedly: %v\n", serveErr)
		}
	}()

	reconcileCtx, stopReconcile := context.WithCancel(context.Background())
	defer stopReconcile()
	go runVerificationReconciler(reconcileCtx, verificationService, mailCfg.ReconcileInterval)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := rabbitmq.GlobalMQ.WaitForConnection(ctx); err != nil {
		log.Fatalf("[Main] Failed to wait for RabbitMQ connection: %v\n", err)
	}

	app.StartWorker(3, app.StartText2ImgWorker)
	app.StartWorker(2, app.StartImg2ImgWorker)
	app.StartWorker(2, app.StartDeadLetterWorker)

	app.Run(user_service.NewService(verificationService))
}

func runVerificationReconciler(ctx context.Context, service *verification.Service, interval time.Duration) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := service.Reconcile(ctx); err != nil {
				log.Printf("[MailReconcile] reconciliation failed: %v\n", err)
			}
		}
	}
}
