// Package chat 对话模块 DAO
package chat

import (
	"fmt"
	"slices"

	chat_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/chat"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/db"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// DAO 对话模块的数据访问层
type DAO struct{}

// ==================== API Key ====================

// GetAPIKeysByUser 获取用户所有 API Key
func (d *DAO) GetAPIKeysByUser(ctx *gin.Context, userUUID string) ([]*chat_do.TableUserAPIKeyDO, error) {
	var keys []*chat_do.TableUserAPIKeyDO
	result := db.GlobalDB.Where("user_uuid = ?", userUUID).Order("created_at DESC").Find(&keys)
	if result.Error != nil {
		logger.Error(ctx, "GetAPIKeysByUser error: %s", result.Error.Error())
		return nil, result.Error
	}
	return keys, nil
}

// CreateAPIKey 创建 API Key
func (d *DAO) CreateAPIKey(ctx *gin.Context, do *chat_do.TableUserAPIKeyDO) error {
	result := db.GlobalDB.Create(do)
	if result.Error != nil {
		logger.Error(ctx, "CreateAPIKey error: %s", result.Error.Error())
		return result.Error
	}
	return nil
}

// UpdateAPIKey 更新 API Key
func (d *DAO) UpdateAPIKey(ctx *gin.Context, id uint64, userUUID string, updates map[string]any) error {
	result := db.GlobalDB.Model(&chat_do.TableUserAPIKeyDO{}).Where("id = ? AND user_uuid = ?", id, userUUID).Updates(updates)
	if result.Error != nil {
		logger.Error(ctx, "UpdateAPIKey error: %s", result.Error.Error())
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("api key 不存在或无权操作")
	}
	return nil
}

// DeleteAPIKey 删除 API Key
func (d *DAO) DeleteAPIKey(ctx *gin.Context, id uint64, userUUID string) error {
	result := db.GlobalDB.Where("id = ? AND user_uuid = ?", id, userUUID).Delete(&chat_do.TableUserAPIKeyDO{})
	if result.Error != nil {
		logger.Error(ctx, "DeleteAPIKey error: %s", result.Error.Error())
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("api key 不存在或无权操作")
	}
	return nil
}

// GetAPIKeyByID 根据 ID 获取 API Key
func (d *DAO) GetAPIKeyByID(_ *gin.Context, id uint64, userUUID string) (*chat_do.TableUserAPIKeyDO, error) {
	var key chat_do.TableUserAPIKeyDO
	result := db.GlobalDB.Where("id = ? AND user_uuid = ?", id, userUUID).First(&key)
	if result.Error != nil {
		return nil, result.Error
	}
	return &key, nil
}

// ==================== Conversation ====================

// GetConversationsByUser 获取用户对话列表
func (d *DAO) GetConversationsByUser(ctx *gin.Context, userUUID string, offset, limit int) ([]*chat_do.TableConversationDO, int64, error) {
	var total int64
	db.GlobalDB.Model(&chat_do.TableConversationDO{}).Where("user_uuid = ?", userUUID).Count(&total)

	var convs []*chat_do.TableConversationDO
	query := db.GlobalDB.Where("user_uuid = ?", userUUID).Order("updated_at DESC")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	result := query.Find(&convs)
	if result.Error != nil {
		logger.Error(ctx, "GetConversationsByUser error: %s", result.Error.Error())
		return nil, 0, result.Error
	}
	return convs, total, nil
}

// CreateConversation 创建对话
func (d *DAO) CreateConversation(ctx *gin.Context, do *chat_do.TableConversationDO) error {
	result := db.GlobalDB.Create(do)
	if result.Error != nil {
		logger.Error(ctx, "CreateConversation error: %s", result.Error.Error())
		return result.Error
	}
	return nil
}

// GetConversationByID 根据 conv_id 获取对话
func (d *DAO) GetConversationByID(_ *gin.Context, convID, userUUID string) (*chat_do.TableConversationDO, error) {
	var conv chat_do.TableConversationDO
	result := db.GlobalDB.Where("conv_id = ? AND user_uuid = ?", convID, userUUID).First(&conv)
	if result.Error != nil {
		return nil, result.Error
	}
	return &conv, nil
}

// DeleteConversation 删除对话
func (d *DAO) DeleteConversation(_ *gin.Context, convID, userUUID string) error {
	tx := db.GlobalDB.Begin()
	tx.Where("conv_id = ?", convID).Delete(&chat_do.TableConversationMessageDO{})
	result := tx.Where("conv_id = ? AND user_uuid = ?", convID, userUUID).Delete(&chat_do.TableConversationDO{})
	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	return tx.Commit().Error
}

// UpdateConversationTitle 更新对话标题
func (d *DAO) UpdateConversationTitle(_ *gin.Context, convID, userUUID, title string) error {
	result := db.GlobalDB.Model(&chat_do.TableConversationDO{}).Where("conv_id = ? AND user_uuid = ?", convID, userUUID).Update("title", title)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("对话不存在或无权操作")
	}
	return nil
}

// TouchConversation 更新对话的 updated_at
func (d *DAO) TouchConversation(convID string) error {
	result := db.GlobalDB.Model(&chat_do.TableConversationDO{}).Where("conv_id = ?", convID).Update("updated_at", db.GlobalDB.NowFunc())
	return result.Error
}

// ==================== Message ====================

// GetMessagesByConvID 获取对话的消息列表
func (d *DAO) GetMessagesByConvID(ctx *gin.Context, convID string) ([]*chat_do.TableConversationMessageDO, error) {
	var msgs []*chat_do.TableConversationMessageDO
	result := db.GlobalDB.Where("conv_id = ?", convID).Order("created_at ASC").Find(&msgs)
	if result.Error != nil {
		logger.Error(ctx, "GetMessagesByConvID error: %s", result.Error.Error())
		return nil, result.Error
	}
	return msgs, nil
}

// GetRecentMessages 获取最近 N 条消息（用于上下文窗口）
func (d *DAO) GetRecentMessages(convID string, limit int) ([]*chat_do.TableConversationMessageDO, error) {
	var msgs []*chat_do.TableConversationMessageDO
	result := db.GlobalDB.Where("conv_id = ?", convID).Order("created_at DESC").Limit(limit).Find(&msgs)
	if result.Error != nil {
		return nil, result.Error
	}
	slices.Reverse(msgs)
	return msgs, nil
}

// CreateMessage 创建消息
func (d *DAO) CreateMessage(do *chat_do.TableConversationMessageDO) error {
	result := db.GlobalDB.Create(do)
	return result.Error
}

// GetMessageCount 获取对话消息数量
func (d *DAO) GetMessageCount(convID string) (int64, error) {
	var count int64
	result := db.GlobalDB.Model(&chat_do.TableConversationMessageDO{}).Where("conv_id = ?", convID).Count(&count)
	return count, result.Error
}

// ==================== Preset ====================

// CreatePreset 创建预设
func (d *DAO) CreatePreset(ctx *gin.Context, do *chat_do.TableChatPresetDO) error {
	result := db.GlobalDB.Create(do)
	if result.Error != nil {
		logger.Error(ctx, "CreatePreset error: %s", result.Error.Error())
		return result.Error
	}
	return nil
}

// GetPresetsByUser 获取用户预设列表
func (d *DAO) GetPresetsByUser(ctx *gin.Context, userUUID string) ([]*chat_do.TableChatPresetDO, error) {
	var presets []*chat_do.TableChatPresetDO
	result := db.GlobalDB.Where("user_uuid = ?", userUUID).Order("updated_at DESC").Find(&presets)
	if result.Error != nil {
		logger.Error(ctx, "GetPresetsByUser error: %s", result.Error.Error())
		return nil, result.Error
	}
	return presets, nil
}

// UpdatePreset 更新预设
func (d *DAO) UpdatePreset(ctx *gin.Context, id uint64, userUUID string, updates map[string]any) error {
	result := db.GlobalDB.Model(&chat_do.TableChatPresetDO{}).Where("id = ? AND user_uuid = ?", id, userUUID).Updates(updates)
	if result.Error != nil {
		logger.Error(ctx, "UpdatePreset error: %s", result.Error.Error())
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("预设不存在或无权操作")
	}
	return nil
}

// DeletePreset 删除预设
func (d *DAO) DeletePreset(ctx *gin.Context, id uint64, userUUID string) error {
	result := db.GlobalDB.Where("id = ? AND user_uuid = ?", id, userUUID).Delete(&chat_do.TableChatPresetDO{})
	if result.Error != nil {
		logger.Error(ctx, "DeletePreset error: %s", result.Error.Error())
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("预设不存在或无权操作")
	}
	return nil
}
