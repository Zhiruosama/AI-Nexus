// Package internal 包含文生图和图生图的 Worker
package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Zhiruosama/ai_nexus/configs"
	image_generation_dao "github.com/Zhiruosama/ai_nexus/internal/dao/image-generation"
	image_generation_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/image-generation"
	"github.com/Zhiruosama/ai_nexus/internal/pkg"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/queue"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/third"
	ws "github.com/Zhiruosama/ai_nexus/internal/pkg/ws"
)

// StartWorker 启动 Worker
func StartWorker(count int, fun func()) {
	for range count {
		go fun()
	}
}

// StartText2ImgWorker 启动文生图 Worker
func StartText2ImgWorker() {
	log.Println("[Worker] Text2Img worker starting")

	err := queue.Consume(queue.QueueText2Img, handleText2ImgTask)
	if err != nil {
		log.Printf("[Worker] Text2Img worker stopped: %v\n", err)
	}
}

// StartImg2ImgWorker 启动图生图 Worker
func StartImg2ImgWorker() {
	log.Println("[Worker] Img2Img worker starting")

	err := queue.Consume(queue.QueueImg2Img, handleImg2ImgTask)
	if err != nil {
		log.Printf("[Worker] Img2Img worker stopped: %v\n", err)
	}
}

// StartDeadLetterWorker 启动死信消费 Worker
func StartDeadLetterWorker() {
	log.Println("[Worker] DeadLetter worker starting")

	err := queue.ConsumeDeadLetter(queue.QueueDeadLetter, handleDeadLetterTask)
	if err != nil {
		log.Printf("[Worker] DeadLetter worker stopped: %v\n", err)
	}
}

// handleText2ImgTask 处理文生图任务
func handleText2ImgTask(msg *queue.TaskMessage) (bool, int8, int8, error) {
	log.Printf("[Worker] Processing text2img task: %s\n", msg.TaskID)

	dao := &image_generation_dao.DAO{}

	// 转换 Payload 为具体类型
	var payload queue.Text2ImgPayload
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		return false, 0, 0, fmt.Errorf("marshal payload error: %w", err)
	}
	if err = json.Unmarshal(payloadBytes, &payload); err != nil {
		return false, 0, 0, fmt.Errorf("unmarshal payload error: %w", err)
	}

	// 获取重试次数
	maxRetries, err := image_generation_dao.GetTaskInfo[int8](dao, "max_retry", msg.TaskID)
	if err != nil {
		return false, 0, 0, err
	}

	retryCount, err := image_generation_dao.GetTaskInfo[int8](dao, "retry_count", msg.TaskID)
	if err != nil {
		return false, 0, 0, err
	}

	// 判断任务是否在队列中
	status, err := image_generation_dao.GetTaskInfo[int8](dao, "status", msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	// 如果已完成或者取消了，直接返回不处理了
	if status == 3 || status == 5 {
		return false, 0, 0, nil
	}

	// 更新状态为处理中
	if err = dao.UpdateTaskParams("status", 2, msg.TaskID); err != nil {
		return true, retryCount, maxRetries, err
	}

	if err = dao.UpdateTaskParams("started_at", time.Now(), msg.TaskID); err != nil {
		return true, retryCount, maxRetries, err
	}

	// 推送给前端我在处理了
	ws.GlobalHub.SendToUser(msg.UserUUID, ws.MessageTypeTaskProgress, ws.TaskProgressData{
		TaskID: msg.TaskID,
		Status: "processing",
	})

	// 调用 第三方API 进行生图
	baseURL, err := image_generation_dao.GetInfoFromModel[string](dao, "base_url", payload.ModelID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	thirdPartyModelID, err := image_generation_dao.GetInfoFromModel[string](dao, "third_party_model_id", payload.ModelID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	apiKey := os.Getenv("MODELSCOPE_API_KEY")
	client := third.NewModelScopeClient(baseURL, apiKey)

	taskID, err := client.CreateText2ImgTask(thirdPartyModelID, payload)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	taskResp, err := client.WaitForTaskCompletion(taskID, 10, 5*time.Second)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	// 保存图片到本地
	path, err := pkg.DownloadAndSaveImages(taskResp.OutputImages[0], 80)
	if err != nil {
		return true, retryCount, maxRetries, fmt.Errorf("DownloadAndSaveImages error: %s", err.Error())
	}

	// 更新数据库
	err = dao.UpdateTaskParams("status", 3, msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, fmt.Errorf("UpdateTaskParams error: %s", err.Error())
	}

	err = dao.UpdateTaskParams("output_image_url", "/"+path, msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	err = dao.UpdateTaskParams("actual_seed", payload.Seed, msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	err = dao.UpdateTaskParams("generation_time_ms", int64(taskResp.TimeTaken), msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	err = dao.UpdateTaskParams("completed_at", time.Now(), msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	// 推送给前端我完成了
	ws.GlobalHub.SendToUser(msg.UserUUID, ws.MessageTypeTaskCompleted, ws.TaskCompletedData{
		TaskID:           msg.TaskID,
		Status:           "completed",
		OutputImageURL:   "http://" + configs.GlobalConfig.Server.SerialStringPublic() + "/" + path,
		GenerationTimeMs: int64(taskResp.TimeTaken),
	})

	log.Printf("[Worker] Text2Img task completed: %s\n", msg.TaskID)
	return false, 0, 0, nil
}

// handleImg2ImgTask 处理图生图任务
func handleImg2ImgTask(msg *queue.TaskMessage) (bool, int8, int8, error) {
	log.Printf("[Worker] Processing img2img task: %s\n", msg.TaskID)

	dao := &image_generation_dao.DAO{}

	// 1. 解析 payload
	var payload queue.Img2ImgPayload

	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		return false, 0, 0, fmt.Errorf("marshal payload error: %w", err)
	}
	if err = json.Unmarshal(payloadBytes, &payload); err != nil {
		return false, 0, 0, fmt.Errorf("unmarshal payload error: %w", err)
	}

	// 2. 检查任务状态和重试次数
	maxRetries, err := image_generation_dao.GetTaskInfo[int8](dao, "max_retry", msg.TaskID)
	if err != nil {
		return false, 0, 0, err
	}

	retryCount, err := image_generation_dao.GetTaskInfo[int8](dao, "retry_count", msg.TaskID)
	if err != nil {
		return false, 0, 0, err
	}

	status, err := image_generation_dao.GetTaskInfo[int8](dao, "status", msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	if status == 3 || status == 5 {
		return false, 0, 0, nil
	}

	// 3. 更新状态为"处理中"
	if err = dao.UpdateTaskParams("status", 2, msg.TaskID); err != nil {
		return true, retryCount, maxRetries, err
	}

	if err = dao.UpdateTaskParams("started_at", time.Now(), msg.TaskID); err != nil {
		return true, retryCount, maxRetries, err
	}

	// 4. 推送消息
	ws.GlobalHub.SendToUser(msg.UserUUID, ws.MessageTypeTaskProgress, ws.TaskProgressData{
		TaskID: msg.TaskID,
		Status: "processing",
	})

	// 5. 调用 ModelScope Img2Img API
	baseURL, err := image_generation_dao.GetInfoFromModel[string](dao, "base_url", payload.ModelID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	thirdPartyModelID, err := image_generation_dao.GetInfoFromModel[string](dao, "third_party_model_id", payload.ModelID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	apiKey := os.Getenv("MODELSCOPE_API_KEY")
	client := third.NewModelScopeClient(baseURL, apiKey)

	taskID, err := client.CreateImg2ImgTask(thirdPartyModelID, payload)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	taskResp, err := client.WaitForTaskCompletion(taskID, 30, 5*time.Second)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	// 6. 保存生成的图片
	path, err := pkg.DownloadAndSaveImages(taskResp.OutputImages[0], 80)
	if err != nil {
		return true, retryCount, maxRetries, fmt.Errorf("DownloadAndSaveImages error: %s", err.Error())
	}

	// 7. 更新数据库
	err = dao.UpdateTaskParams("status", 3, msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, fmt.Errorf("UpdateTaskParams error: %s", err.Error())
	}

	err = dao.UpdateTaskParams("output_image_url", "/"+path, msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	err = dao.UpdateTaskParams("actual_seed", payload.Seed, msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	err = dao.UpdateTaskParams("generation_time_ms", int64(taskResp.TimeTaken), msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	err = dao.UpdateTaskParams("completed_at", time.Now(), msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	// 8. WebSocket 推送通知
	ws.GlobalHub.SendToUser(msg.UserUUID, ws.MessageTypeTaskCompleted, ws.TaskCompletedData{
		TaskID:           msg.TaskID,
		Status:           "completed",
		OutputImageURL:   "http://" + configs.GlobalConfig.Server.SerialStringPublic() + "/" + path,
		GenerationTimeMs: int64(taskResp.TimeTaken),
	})

	log.Printf("[Worker] Img2Img task completed: %s\n", msg.TaskID)
	return false, 0, 0, nil
}

// handleDeadLetterTask 处理死信队列中的任务
func handleDeadLetterTask(msg *queue.TaskMessage, xDeathInfo map[string]any) error {
	log.Printf("[Worker] Processing dead letter task: %s\n", msg.TaskID)

	dao := &image_generation_dao.DAO{}

	// 查询任务信息
	taskType, err := image_generation_dao.GetTaskInfo[int8](dao, "task_type", msg.TaskID)
	if err != nil {
		return err
	}

	status, err := image_generation_dao.GetTaskInfo[int8](dao, "status", msg.TaskID)
	if err != nil {
		return err
	}

	// 检查是否已记录过死信
	exist, err := dao.CheckDeadLetterExists(msg.TaskID)
	if err != nil {
		return err
	}
	if exist {
		log.Printf("[Worker] Dead letter task %s already recorded, skipping\n", msg.TaskID)
		return nil
	}

	// 解析死信原因
	deadReason := parseDeadLetterReason(xDeathInfo, status)
	xDeathInfoJson, err := json.Marshal(xDeathInfo)
	if err != nil {
		log.Printf("[Worker] Failed to marshal x-death info for task %s: %v\n", msg.TaskID, err)
		xDeathInfoJson = []byte("{}")
	}
	xDeathInfoStr := string(xDeathInfoJson)

	log.Printf("[Worker] Dead letter reason for task %s: %s\n", msg.TaskID, xDeathInfoStr)

	// 插入死信记录
	do := image_generation_do.TableDeadLetterTasksDO{
		UserID:         msg.UserUUID,
		TaskID:         msg.TaskID,
		TaskType:       taskType,
		DeadReason:     deadReason,
		OriginalStatus: status,
	}

	err = dao.InsertDeadLetterTask(&do)
	if err != nil {
		return err
	}

	// 更新原任务状态为失败
	if status != 4 {
		if err := dao.UpdateTaskParams("status", 4, msg.TaskID); err != nil {
			log.Printf("[Worker] Failed to update task status for %s: %v\n", msg.TaskID, err)
		}

		if err := dao.UpdateTaskParams("error_message", xDeathInfoStr, msg.TaskID); err != nil {
			log.Printf("[Worker] Failed to update error_message for %s: %v\n", msg.TaskID, err)
		}

		if err := dao.UpdateTaskParams("completed_at", time.Now(), msg.TaskID); err != nil {
			log.Printf("[Worker] Failed to update completed_at for %s: %v\n", msg.TaskID, err)
		}
	}

	// 通知前端任务失败
	ws.GlobalHub.SendToUser(msg.UserUUID, ws.MessageTypeTaskFailed, ws.TaskFailedData{
		TaskID:       msg.TaskID,
		Status:       "failed",
		ErrorMessage: deadReason,
	})

	log.Printf("[Worker] Dead letter task %s processed successfully\n", msg.TaskID)
	return nil
}

// parseDeadLetterReason 解析死信原因
func parseDeadLetterReason(xDeathInfo map[string]any, status int8) string {
	if reason, ok := xDeathInfo["reason"].(string); ok {
		switch reason {
		case "rejected":
			return "任务重试次数耗尽,被拒绝处理"
		case "expired":
			return "任务在队列中超时(TTL过期)"
		case "maxlen":
			return "队列长度超过限制"
		default:
			return fmt.Sprintf("RabbitMQ死信原因: %s", reason)
		}
	}

	switch status {
	case 1:
		return "任务在队列中超时未被处理"
	case 2:
		return "Worker处理异常或崩溃"
	case 4:
		return "任务已标记为失败"
	default:
		return "未知原因进入死信队列"
	}
}
