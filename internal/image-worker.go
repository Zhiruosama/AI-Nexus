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
	"github.com/Zhiruosama/ai_nexus/internal/pkg"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/queue"
	rabbitmq "github.com/Zhiruosama/ai_nexus/internal/pkg/queue"
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

// handleText2ImgTask 处理文生图任务
func handleText2ImgTask(msg *queue.TaskMessage) (bool, int8, int8, error) {
	log.Printf("[Worker] Processing text2img task: %s\n", msg.TaskID)

	dao := &image_generation_dao.DAO{}

	// 转换 Payload 为具体类型
	var payload rabbitmq.Text2ImgPayload
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		return false, 0, 0, fmt.Errorf("marshal payload error: %w", err)
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
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
	if err := dao.UpdateTaskParams("status", 2, msg.TaskID); err != nil {
		return true, retryCount, maxRetries, err
	}

	if err := dao.UpdateTaskParams("started_at", time.Now(), msg.TaskID); err != nil {
		return true, retryCount, maxRetries, err
	}

	// 推送给前端我在处理了
	ws.GlobalHub.SendToUser(msg.UserUUID, ws.MessageTypeTaskProgress, ws.TaskProgressData{
		TaskID: msg.TaskID,
		Status: "processing",
	})

	// 调用 第三方API 进行生图
	baseUrl, err := image_generation_dao.GetInfoFromModel[string](dao, "base_url", payload.ModelID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	thirdPartyModelId, err := image_generation_dao.GetInfoFromModel[string](dao, "third_party_model_id", payload.ModelID)
	if err != nil {
		return true, retryCount, maxRetries, err
	}

	apiKey := os.Getenv("MODELSCOPE_API_KEY")
	client := third.NewModelScopeClient(baseUrl, apiKey)

	taskID, err := client.CreateTask(thirdPartyModelId, payload)
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
		return true, retryCount, maxRetries, fmt.Errorf("DownloadAndSaveImages error: %s\n", err.Error())
	}

	// 更新数据库
	err = dao.UpdateTaskParams("status", 3, msg.TaskID)
	if err != nil {
		return true, retryCount, maxRetries, fmt.Errorf("UpdateTaskParams error: %s\n", err.Error())
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
		OutputImageURL:   "http://" + configs.GlobalConfig.Server.SerialString() + "/" + path,
		GenerationTimeMs: int64(taskResp.TimeTaken),
	})

	log.Printf("[Worker] Text2Img task completed: %s\n", msg.TaskID)
	return false, 0, 0, nil
}

// handleImg2ImgTask 处理图生图任务
func handleImg2ImgTask(msg *queue.TaskMessage) (bool, int8, int8, error) {
	log.Printf("[Worker] Processing img2img task: %s\n", msg.TaskID)

	// TODO: 实现图生图处理逻辑（与文生图类似，但需要处理输入图片）
	// 2. 检查任务状态和重试次数
	// 3. 更新状态为"处理中"
	// 4. 读取输入图片
	// 5. 调用 ModelScope Img2Img API
	// 6. 保存生成的图片
	// 7. 更新任务状态为"已完成"
	// 8. WebSocket 推送通知

	log.Printf("[Worker] Img2Img task completed: %s\n", msg.TaskID)
	return false, 0, 0, nil
}
