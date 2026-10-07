package model

import (
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"lostfound/pkg/timeutil"
)

/*
演示数据。

选这套数据而不是随手造几条，是为了让前端开箱就能看到真实的分页与筛选行为：
两类各超过 12 条（前端每页 12 条），滚动加载才有第二页可翻；另备一条待审核与
一条已驳回，审核台与「我的发布」的驳回理由展示才不是空态。
*/

// seedUserDefinition 是一个种子账号。
type seedUserDefinition struct {
	Username string
	Password string
	Nickname string
	Contact  string
	Role     string
}

// seedItemDefinition 是一条种子帖子。字段顺序即 seedItemDefinitions 里的取值顺序：
// 大类、名称、分类、地点、描述、领取地点、领取联系方式、几天前。
type seedItemDefinition struct {
	Type        string
	ItemName    string
	Category    string
	Location    string
	Description string
	GetLocation string
	GetContact  string
	DaysAgo     int
}

// Seed 写入演示数据，让服务首次启动就有一屏可看的内容。
//
// 幂等：库里已有用户则直接返回。这样反复重启不会累积重复数据，
// 也不会覆盖使用者已经在界面上改过的内容。
func Seed(db *gorm.DB) error {
	var existing int64
	if err := db.Model(&User{}).Count(&existing).Error; err != nil {
		return fmt.Errorf("统计用户失败: %w", err)
	}
	if existing > 0 {
		return nil
	}

	userIDs, err := seedUsers(db)
	if err != nil {
		return err
	}
	if err := seedItems(db, userIDs); err != nil {
		return err
	}
	if err := seedClaims(db, userIDs); err != nil {
		return err
	}
	return seedAnnouncements(db)
}

// seedUsers 写入演示账号，返回「用户名 → 用户 ID」的映射。
// 后续数据一律按用户名解析发布者，避免往数组中间插入新账号后写死的数字整体位移。
func seedUsers(db *gorm.DB) (map[string]uint, error) {
	definitions := []seedUserDefinition{
		{"admin", "admin123", "系统管理员", "13800000000", RoleSysAdmin},
		// 第二个系统管理员：文档「不允许修改其他系统管理员的角色」这条规则，
		// 要有两个系统管理员才验证得了。
		{"admin2", "admin123", "系统管理员（副）", "13800000099", RoleSysAdmin},
		{"finder001", "abc123", "招领处・李同学", "liming@campus.edu", RoleFinderAdmin},
		{"student001", "abc123", "张同学", "pzx@campus.edu", RoleUser},
		{"student002", "abc123", "王同学", "wang@campus.edu", RoleUser},
	}

	ids := make(map[string]uint, len(definitions))
	for _, definition := range definitions {
		digest, err := bcrypt.GenerateFromPassword([]byte(definition.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("生成口令摘要失败: %w", err)
		}
		user := &User{
			Username:    definition.Username,
			Password:    string(digest),
			Nickname:    definition.Nickname,
			Contact:     definition.Contact,
			Role:        definition.Role,
			CreatedTime: timeutil.Now(),
		}
		if err := db.Create(user).Error; err != nil {
			return nil, fmt.Errorf("写入种子用户 %s 失败: %w", definition.Username, err)
		}
		ids[definition.Username] = user.UserID
	}
	return ids, nil
}

// seedItemDefinitions 是手工编写的 20 条帖子（两类各 10 条），描述都写得比较具体，
// 用于展示详情页与认领理由核对的实际观感。
var seedItemDefinitions = []seedItemDefinition{
	// ── 寻物启事 ──
	{ItemTypeLost, "黑色长柄雨伞", "其他", "三号教学楼 2 层自习室", "伞柄有一圈磨白的胶布，伞骨内侧写了名字缩写", "图书馆一层服务台", "asd@campus.edu", 2},
	{ItemTypeLost, "AirPods Pro 充电盒", "数码电子", "体育馆羽毛球场 4 号场", "外壳有一处磕碰凹痕，内侧刻了手机号后四位", "体育馆器材室", "pasd@campus.edu", 5},
	{ItemTypeLost, "蓝色学生卡（章同学）", "卡证", "二食堂二楼", "卡面右下角贴了一枚蓝色贴纸", "二食堂失物窗", "13800000001", 1},
	{ItemTypeLost, "黑色双肩包", "箱包", "东门校车站台", "包内有线装笔记本与一支钢笔，拉链头换过", "东门保安亭", "pasd@campus.edu", 3},
	{ItemTypeLost, "银色保温杯", "其他", "四号教学楼 501 阶梯教室", "杯盖内圈有茶渍，杯身贴着实验室标签", "四号教学楼值班室", "13500000002", 8},
	{ItemTypeLost, "宿舍钥匙（三把一串）", "钥匙", "五号宿舍楼电梯口", "钥匙扣是橙色塑料圆牌，牌上写着宿舍号", "五号宿舍楼宿管处", "13600000003", 2},
	{ItemTypeLost, "蓝色文件夹（内含实验报告）", "书籍文具", "化学实验楼 B210", "封面贴着一张实验安排表，内有手写数据", "化学实验楼收发室", "pasd@campus.edu", 6},
	{ItemTypeLost, "灰色围巾", "衣物", "图书馆三层南侧靠窗座位", "羊绒材质，一端有勾丝", "图书馆一层服务台", "13700000004", 4},
	{ItemTypeLost, "棕色皮质钱包（少量现金）", "现金钱包", "操场跑道外侧看台", "内有校园卡与一张公交卡，现金不多", "操场管理室", "pasd@campus.edu", 7},
	{ItemTypeLost, "银戒指（素圈）", "挂饰饰品", "游泳馆更衣室 3 号柜前", "内圈刻字模糊，圈口偏细", "游泳馆前台", "13800000005", 9},

	// ── 失物招领 ──
	{ItemTypeFound, "白色无线鼠标", "数码电子", "一号教学楼机房 A", "底部有使用痕迹，接收器还收在电池仓里", "一号教学楼机房值班室", "机房管理员 13500000010", 1},
	{ItemTypeFound, "黑色雨伞（自动伞）", "其他", "三号教学楼大厅伞架", "伞面印有校徽，自动开合正常", "三号教学楼物业值班室", "物业 13500000011", 2},
	{ItemTypeFound, "学生证（张同学）", "卡证", "图书馆自助借还机旁", "证件照清晰，学号可见", "图书馆一层服务台", "前台 13500000012", 3},
	{ItemTypeFound, "深蓝色笔袋", "书籍文具", "四号教学楼 302 教室", "内含两支黑色签字笔与一把直尺", "四号教学楼值班室", "值班室 13500000013", 4},
	{ItemTypeFound, "保温杯（不锈钢）", "其他", "体育馆看台第三排", "杯身贴着一枚已经卷边的贴纸", "体育馆器材室", "器材室 13500000014", 5},
	{ItemTypeFound, "一串钥匙（带小黄鸭挂件）", "钥匙", "二食堂门口长椅", "挂件是一只小黄鸭，共四把钥匙", "二食堂失物窗", "窗口 13500000015", 6},
	{ItemTypeFound, "黑色有线耳机", "数码电子", "五号宿舍楼一楼洗衣房", "线材有一处缠胶带，耳塞套偏小号", "五号宿舍楼宿管处", "宿管 13500000016", 8},
	{ItemTypeFound, "米白色针织开衫", "衣物", "一号教学楼 208 教室椅背", "均码，左袖口有小块污渍", "一号教学楼值班室", "值班室 13500000017", 9},
	{ItemTypeFound, "帆布袋（印有社团标志）", "箱包", "操场入口台阶", "袋内有一本英语词汇书与一副眼镜布", "操场管理室", "管理室 13500000018", 7},
	{ItemTypeFound, "银色手链", "挂饰饰品", "游泳馆更衣区长凳", "链节处有一处断开后用线缠过", "游泳馆前台", "前台 13500000019", 10},
}

// seedExtraCategories 供批量生成的帖子循环取分类。
var seedExtraCategories = []string{"卡证", "数码电子", "钥匙", "书籍文具", "衣物", "其他", "箱包"}

// seedExtraLostNames 与 seedExtraFoundNames 用于把两类各自撑过一页（12 条），
// 让前端的「滚动加载下一页」有第二页可翻。
var seedExtraLostNames = []string{
	"黑色机械键盘腕托", "充电宝（白色 20000mAh）", "蓝色水彩笔盒", "黄色手机支架",
	"灰色运动水壶", "银色 U 盘（32G）", "红色棒球帽", "白色护腕（一对）",
	"黑色眼镜盒", "绿色环保袋", "棕色皮带", "粉色化妆包",
	"黑色耳机转接头", "蓝色游泳镜", "白色充电线（Type-C）", "灰色笔记本内胆包",
	"黑色计算器", "黄色雨衣",
}

var seedExtraFoundNames = []string{
	"黑色 U 盘（16G）", "蓝色运动手环", "白色充电器（65W）", "灰色钥匙包",
	"棕色皮质笔袋", "银色口琴", "红色围脖", "黑色刻度尺",
	"蓝色保温饭盒", "白色蓝牙耳机盒", "灰色鼠标垫", "黑色双肩电脑包",
}

// seedItems 写入全部演示帖子（手工 20 条 + 批量生成 30 条 + 特殊状态 2 条）。
func seedItems(db *gorm.DB, userIDs map[string]uint) error {
	lostPoster := userIDs["student001"]
	foundPoster := userIDs["finder001"]
	secondStudent := userIDs["student002"]

	items := make([]Item, 0, 64)
	for _, definition := range seedItemDefinitions {
		items = append(items, buildSeedItem(definition, lostPoster, foundPoster))
	}
	// 寻物启事轮流由两个学生发布，失物招领统一由招领处发布 —— 与实际使用情形一致。
	items = append(items, buildExtraSeedItems(seedExtraLostNames, ItemTypeLost, []uint{lostPoster, secondStudent}, 3)...)
	items = append(items, buildExtraSeedItems(seedExtraFoundNames, ItemTypeFound, []uint{foundPoster}, 1)...)
	items = append(items, buildPendingSeedItem(lostPoster), buildRejectedSeedItem(foundPoster))

	if err := db.Create(&items).Error; err != nil {
		return fmt.Errorf("写入种子帖子失败: %w", err)
	}
	return nil
}

// buildSeedItem 把一条定义翻译成实体。
func buildSeedItem(definition seedItemDefinition, lostPoster, foundPoster uint) Item {
	happenTime := daysAgo(definition.DaysAgo)

	posterID := lostPoster
	if definition.Type == ItemTypeFound {
		posterID = foundPoster
	}

	return Item{
		Type:          definition.Type,
		ItemName:      definition.ItemName,
		Category:      definition.Category,
		Location:      definition.Location,
		HappenTime:    happenTime,
		PosterContact: definition.GetContact,
		Description:   definition.Description,
		Image:         []string{placeholderImage(definition.ItemName, definition.Category)},
		ItemStatus:    ItemStatusPublished,
		CreatedTime:   happenTime,
		LastEditTime:  happenTime,
		PosterID:      posterID,
		GetLocation:   definition.GetLocation,
		GetContact:    definition.GetContact,
	}
}

// buildExtraSeedItems 批量生成帖子，把某一类撑过一页（前端每页 12 条），
// 让「滚动加载下一页」有第二页可翻。
//
// posters 里给出该类帖子的候选发布者，按序轮流取用：寻物启事在两个学生之间交替，
// 失物招领则只有招领处一个发布者。firstDayOffset 是这批帖子最早的时间（几天前），
// 之后每条再往后推 2 天，使时间自然铺开、互不重叠。
func buildExtraSeedItems(names []string, itemType string, posters []uint, firstDayOffset int) []Item {
	traits := []string{"有一处明显磨损", "贴了手写标签", "边角有磕碰痕迹", "颜色略有褪色", "带有挂饰"}

	items := make([]Item, 0, len(names))
	for index, name := range names {
		trait := traits[index%len(traits)]
		category := seedExtraCategories[index%len(seedExtraCategories)]
		posterID := posters[index%len(posters)]
		happenTime := daysAgo(index*2 + firstDayOffset)
		place := fmt.Sprintf("%d 号教学楼值班室", index%6+1)

		description := fmt.Sprintf("%s的特征：%s，如有拾到请与我联系。", name, trait)
		contact := "student@campus.edu"
		if itemType == ItemTypeFound {
			// 失物招领是「捡到者在等失主」，措辞与联系方式都该换成招领处口径。
			description = fmt.Sprintf("在 %d 号教学楼走廊拾到，%s，暂存于值班室。", index%5+1, trait)
			contact = "招领处 13500000010"
		}

		items = append(items, Item{
			Type:          itemType,
			ItemName:      name,
			Category:      category,
			Location:      fmt.Sprintf("%d 号教学楼 %d0%d 教室", index%6+1, index%4+1, index%9+1),
			HappenTime:    happenTime,
			PosterContact: contact,
			Description:   description,
			Image:         []string{placeholderImage(name, category)},
			ItemStatus:    ItemStatusPublished,
			CreatedTime:   happenTime,
			LastEditTime:  happenTime,
			PosterID:      posterID,
			GetLocation:   place,
			GetContact:    contact,
		})
	}
	return items
}

// buildPendingSeedItem 生成一条待审核帖子，让审核台开箱即有内容。
func buildPendingSeedItem(posterID uint) Item {
	happenTime := daysAgo(1)
	return Item{
		Type:          ItemTypeLost,
		ItemName:      "白色蓝牙音箱",
		Category:      "数码电子",
		Location:      "六号教学楼报告厅",
		HappenTime:    happenTime,
		PosterContact: "pzx@campus.edu",
		Description:   "音箱侧面有一道细小划痕，配黑色挂绳",
		Image:         []string{placeholderImage("白色蓝牙音箱", "数码电子")},
		ItemStatus:    ItemStatusPending,
		CreatedTime:   happenTime,
		LastEditTime:  happenTime,
		PosterID:      posterID,
		GetLocation:   "六号教学楼值班室",
		GetContact:    "pzx@campus.edu",
	}
}

// buildRejectedSeedItem 生成一条已驳回帖子，让「我的发布」能展示驳回理由。
func buildRejectedSeedItem(posterID uint) Item {
	return Item{
		Type:          ItemTypeFound,
		ItemName:      "保温饭盒",
		Category:      "其他",
		Location:      "二食堂三楼",
		HappenTime:    daysAgo(4),
		PosterContact: "窗口 13500000020",
		Description:   "饭盒",
		Image:         []string{placeholderImage("保温饭盒", "其他")},
		ItemStatus:    ItemStatusRejected,
		CreatedTime:   daysAgo(4),
		LastEditTime:  daysAgo(3),
		RejectReason:  "描述过于简单，请补充外观特征（颜色、品牌、有无贴纸）后重新提交",
		PosterID:      posterID,
		GetLocation:   "二食堂失物窗",
		GetContact:    "窗口 13500000020",
	}
}

// seedClaims 写入两条演示申请：一条待审批、一条已通过。
// 按物品名称反查 ID，避免依赖帖子在数组中的插入顺序。
func seedClaims(db *gorm.DB, userIDs map[string]uint) error {
	pendingItem, err := findItemByName(db, "白色无线鼠标")
	if err != nil {
		return err
	}
	approvedItem, err := findItemByName(db, "学生证（张同学）")
	if err != nil {
		return err
	}

	claims := []Claim{
		{
			ItemID:           pendingItem.ItemID,
			Reason:           "鼠标底部有一道我贴的防滑贴，接收器上写着我的名字缩写 WL",
			ApplicantContact: "wang@campus.edu",
			CreatedTime:      daysAgo(0),
			LastEditTime:     daysAgo(0),
			ClaimStatus:      ClaimStatusPending,
			ApplicantID:      userIDs["student002"],
		},
		{
			ItemID:           approvedItem.ItemID,
			Reason:           "学生证是我本人的，学号 2023xxxxxx，证件照可以核对",
			ApplicantContact: "pzx@campus.edu",
			CreatedTime:      daysAgo(1),
			LastEditTime:     daysAgo(1),
			ClaimStatus:      ClaimStatusApproved,
			ApplicantID:      userIDs["student001"],
		},
	}

	if err := db.Create(&claims).Error; err != nil {
		return fmt.Errorf("写入种子认领申请失败: %w", err)
	}
	return nil
}

// findItemByName 按名称取一条种子帖子。
func findItemByName(db *gorm.DB, name string) (*Item, error) {
	var item Item
	if err := db.Where("item_name = ?", name).First(&item).Error; err != nil {
		return nil, fmt.Errorf("查找种子物品 %s 失败: %w", name, err)
	}
	return &item, nil
}

// seedAnnouncements 写入公告：两条公开、一条已下线。
// 已下线的这条用于验证「公开列表看不到它，但管理端看得到」。
func seedAnnouncements(db *gorm.DB) error {
	announcements := []Announcement{
		{
			Title:              "失物招领处开放时间调整",
			Content:            "自本周起，图书馆一层失物招领服务台开放时间调整为 08:30–20:30，中午不休息。\n节假日期间请以门口告示为准，感谢配合。",
			AnnouncementStatus: AnnouncementStatusPublished,
			CreatedTime:        daysAgo(2),
		},
		{
			Title:              "关于认领流程的说明",
			Content:            "为提高核对准确率，认领申请需写清物品的可辨识特征。管理员核对通过后，会在 1 个工作日内联系申请人安排交接。\n请勿在申请中直接公开完整证件号码等敏感信息。",
			AnnouncementStatus: AnnouncementStatusPublished,
			CreatedTime:        daysAgo(6),
		},
		{
			Title:              "【已结束】开学季失物集中清点",
			Content:            "开学季集中清点活动已结束，未被认领的物品已统一移交各楼栋值班室保管。",
			AnnouncementStatus: AnnouncementStatusOffline,
			CreatedTime:        daysAgo(20),
		},
	}

	if err := db.Create(&announcements).Error; err != nil {
		return fmt.Errorf("写入种子公告失败: %w", err)
	}
	return nil
}

// categoryPalette 为每个分类配一组「底色 / 墨色」，让占位图有区分度。
var categoryPalette = map[string][2]string{
	"卡证":   {"#dbeafe", "#1e3a8a"},
	"数码电子": {"#e0e7ff", "#312e81"},
	"挂饰饰品": {"#fce7f3", "#831843"},
	"箱包":   {"#fef3c7", "#78350f"},
	"钥匙":   {"#d1fae5", "#064e3b"},
	"书籍文具": {"#ede9fe", "#4c1d95"},
	"衣物":   {"#ffe4e6", "#881337"},
	"现金钱包": {"#dcfce7", "#14532d"},
	"其他":   {"#f1f5f9", "#334155"},
}

// placeholderImage 生成内联 SVG 占位图（data URL），让演示数据离线也能渲染出图片。
//
// 用 base64 而非百分号编码：SVG 里含引号与大量中文，base64 不必逐字符转义，
// 也不受 URL 保留字符（如 #）影响，浏览器同样能直接渲染。
func placeholderImage(label, category string) string {
	colors, exists := categoryPalette[category]
	if !exists {
		colors = categoryPalette["其他"]
	}
	background, ink := colors[0], colors[1]

	// 名称过长时截断，避免文字溢出画布。
	runes := []rune(label)
	if len(runes) > 12 {
		runes = runes[:12]
	}

	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="640" height="480" viewBox="0 0 640 480">`+
			`<rect width="640" height="480" fill="%s"/>`+
			`<circle cx="500" cy="90" r="120" fill="%s" opacity="0.08"/>`+
			`<circle cx="120" cy="400" r="160" fill="%s" opacity="0.06"/>`+
			`<text x="48" y="250" font-family="PingFang SC, Helvetica, sans-serif" font-size="46" font-weight="600" fill="%s">%s</text>`+
			`<text x="48" y="300" font-family="PingFang SC, Helvetica, sans-serif" font-size="22" fill="%s" opacity="0.6">%s</text>`+
			`</svg>`,
		background, ink, ink, ink, string(runes), ink, category,
	)
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg))
}

// daysAgo 返回 n 天前的时间串，用于把种子帖子的时间摊开在最近两周内。
func daysAgo(days int) string {
	return time.Now().Add(-time.Duration(days) * 24 * time.Hour).Format(timeutil.Layout)
}
