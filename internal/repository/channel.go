// internal/repository/channel.go
package repository

import (
	"strings"

	"github.com/example/epay-go/internal/database"
	"github.com/example/epay-go/internal/model"
	"gorm.io/gorm"
)

type ChannelRepository struct {
	db *gorm.DB
}

func NewChannelRepository() *ChannelRepository {
	return &ChannelRepository{db: database.Get()}
}

// Create 创建通道
func (r *ChannelRepository) Create(channel *model.Channel) error {
	return r.db.Create(channel).Error
}

// GetByID 根据ID获取通道
func (r *ChannelRepository) GetByID(id int64) (*model.Channel, error) {
	var channel model.Channel
	err := r.db.First(&channel, id).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// Update 更新通道
func (r *ChannelRepository) Update(channel *model.Channel) error {
	return r.db.Save(channel).Error
}

// Delete 删除通道
func (r *ChannelRepository) Delete(id int64) error {
	return r.db.Delete(&model.Channel{}, id).Error
}

// List 分页查询通道列表
func (r *ChannelRepository) List(page, pageSize int) ([]model.Channel, int64, error) {
	var channels []model.Channel
	var total int64

	err := r.db.Model(&model.Channel{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = r.db.Offset(offset).Limit(pageSize).Order("sort ASC, id ASC").Find(&channels).Error
	if err != nil {
		return nil, 0, err
	}

	return channels, total, nil
}

// ListEnabled 获取所有启用的通道
func (r *ChannelRepository) ListEnabled() ([]model.Channel, error) {
	var channels []model.Channel
	err := r.db.Where("status = ?", 1).Order("sort ASC, id ASC").Find(&channels).Error
	return channels, err
}

// GetByPluginAndPayType 根据插件和支付类型获取可用通道
func (r *ChannelRepository) GetByPluginAndPayType(plugin, payType string) (*model.Channel, error) {
	var channel model.Channel
	query := r.db.Where("plugin = ? AND status = 1", plugin)
	if strings.TrimSpace(payType) != "" {
		query = query.Where("pay_types LIKE ?", "%"+payType+"%")
	}
	err := query.Order("sort ASC").First(&channel).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// GetAvailableByPayType 根据支付类型获取可用通道
func (r *ChannelRepository) GetAvailableByPayType(payType string) (*model.Channel, error) {
	var channel model.Channel
	normalizedPayType := strings.ToLower(strings.TrimSpace(payType))
	query := r.db.Where("status = 1")

	switch normalizedPayType {
	case "wxpay", "wechat":
		// 官方微信(plugin=wechat) 或 汇付微信(plugin=hf-wxpay)
		query = query.Where("plugin IN ? OR pay_types LIKE ?",
			[]string{"wechat", "hf-wxpay"}, "%"+normalizedPayType+"%")
	case "alipay":
		// 官方支付宝(plugin=alipay) 或 汇付支付宝(plugin=hf-alipay)
		query = query.Where("plugin IN ? OR pay_types LIKE ?",
			[]string{"alipay", "hf-alipay"}, "%"+normalizedPayType+"%")
	default:
		query = query.Where("pay_types LIKE ?", "%"+normalizedPayType+"%")
	}

	err := query.Order("sort ASC").First(&channel).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}
