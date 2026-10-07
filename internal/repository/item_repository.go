package repository

import (
	"gorm.io/gorm"

	"lostfound/internal/model"
)

// ItemFilter 是物品查询条件。
//
// 用具名结构体而不是一串可变参数：调用方一眼能看出「能按哪些维度筛」，
// 指针字段表示「可选」，nil 即不过滤 —— 这与接口文档里可选查询参数语义一致。
type ItemFilter struct {
	Type      string
	Status    *int
	Category  string
	Keyword   string
	Location  string
	StartTime string
	EndTime   string
	PosterID  *uint
	// OrderBy 由本包调用方从固定枚举里给出（不接受用户输入），默认按 happen_time 倒序。
	OrderBy string
	Offset  int
	Limit   int
}

// ItemRepository 提供物品的数据访问。
type ItemRepository struct {
	db *gorm.DB
}

// NewItemRepository 构造物品仓储。
func NewItemRepository(db *gorm.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// Create 新增帖子。
func (r *ItemRepository) Create(item *model.Item) error {
	return r.db.Create(item).Error
}

// FindByID 按主键查询，不存在时返回 ErrNotFound。
func (r *ItemRepository) FindByID(itemID uint) (*model.Item, error) {
	var item model.Item
	if err := r.db.First(&item, "item_id = ?", itemID).Error; err != nil {
		return nil, translate(err)
	}
	return &item, nil
}

// Update 全量保存帖子字段。
//
// 用 Save（结构体更新）而不是 map 更新：Image 字段走 GORM 的 json serializer 落库，
// serializer 只在结构体维度的写入里生效，若把它塞进 map[string]any 会以 []string
// 原始类型直接交给驱动，SQLite 无法绑定该类型而报错。
//
// 语义上这也正是「全段更新」：调用方拼好完整实体后整体落库，不做部分更新，
// 免得出现「只改了名称但漏更新 last_edit_time」这类半更新状态。
func (r *ItemRepository) Update(item *model.Item) error {
	return r.db.Save(item).Error
}

// UpdateStatus 只改状态与最后修改时间，供审核、关闭等状态流转使用。
func (r *ItemRepository) UpdateStatus(itemID uint, status int, lastEditTime string) error {
	return r.db.Model(&model.Item{}).
		Where("item_id = ?", itemID).
		Updates(map[string]any{
			"item_status":    status,
			"last_edit_time": lastEditTime,
		}).Error
}

// Delete 删除帖子。
func (r *ItemRepository) Delete(itemID uint) error {
	return r.db.Delete(&model.Item{}, "item_id = ?", itemID).Error
}

// List 按条件分页查询。
func (r *ItemRepository) List(filter ItemFilter) ([]model.Item, error) {
	query := r.db.Model(&model.Item{})

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != nil {
		query = query.Where("item_status = ?", *filter.Status)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("item_name LIKE ? OR description LIKE ?", like, like)
	}
	if filter.Location != "" {
		query = query.Where("location LIKE ?", "%"+filter.Location+"%")
	}
	if filter.StartTime != "" {
		query = query.Where("happen_time >= ?", filter.StartTime)
	}
	if filter.EndTime != "" {
		query = query.Where("happen_time <= ?", filter.EndTime)
	}
	if filter.PosterID != nil {
		query = query.Where("poster_id = ?", *filter.PosterID)
	}

	orderBy := filter.OrderBy
	if orderBy == "" {
		orderBy = "happen_time desc"
	}

	var items []model.Item
	if err := query.Order(orderBy).Limit(filter.Limit).Offset(filter.Offset).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
