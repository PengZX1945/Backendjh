package router_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lostfound/internal/config"
	"lostfound/internal/database"
	"lostfound/internal/model"
	"lostfound/internal/router"
)

/*
接口契约测试。

对着一个装配完整、数据已初始化的引擎逐条核对：错误码、角色权限、物品与认领的
状态机、分页与信封结构。这些断言直接抄自接口文档与前端交付说明里列出的口径，
因此它们同时也是「后端与前端约定一致」的可执行证据。

设计取舍：用真实的路由 + 真实的 SQLite（临时文件）+ 真实的种子数据，而不是打桩。
打桩只能证明「handler 调了 service」，证明不了「整条链路上的错误码、权限与状态流转
是否符合约定」—— 而这恰恰是最容易出错、也最需要被守住的部分。
*/

/* ────────────────────────── 测试脚手架 ────────────────────────── */

// envelope 是统一响应信封，用来断言 code 与 msg。
type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// newTestServer 起一个装好种子数据的引擎。
func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	uploadDir := filepath.Join(t.TempDir(), "uploads")
	cfg := &config.Config{
		Server:   config.Server{Port: 0, Mode: "release"},
		Database: config.Database{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "test.db"), Seed: true},
		Auth:     config.Auth{JWTSecret: "test-secret", JWTExpireHours: 1, Issuer: "lostfound-test"},
		Upload:   config.Upload{Dir: uploadDir, URLPrefix: "/api/uploads", MaxSizeMB: 5, AllowedExts: []string{"jpg", "jpeg", "png", "webp"}},
		Logger:   config.Logger{Level: "error"},
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	if err := model.Seed(db); err != nil {
		t.Fatalf("写入测试数据失败: %v", err)
	}

	// 上传目录必须存在，静态路由才能命中真实文件。
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		t.Fatalf("创建上传目录失败: %v", err)
	}

	return router.New(cfg, db)
}

// do 发一个 JSON 请求，返回 HTTP 状态码与解析后的信封。
func do(t *testing.T, engine http.Handler, method, path, token string, body any) (int, envelope) {
	t.Helper()

	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		payload = encoded
	}

	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	var parsed envelope
	if recorder.Body.Len() > 0 {
		if err := json.Unmarshal(recorder.Body.Bytes(), &parsed); err != nil {
			t.Fatalf("%s %s 的响应不是合法 JSON: %s", method, path, recorder.Body.String())
		}
	}
	return recorder.Code, parsed
}

// login 登录并返回凭证。
func login(t *testing.T, engine http.Handler, username, password string) string {
	t.Helper()

	status, response := do(t, engine, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": username,
		"password": password,
	})
	if status != http.StatusOK || response.Code != 0 {
		t.Fatalf("登录 %s 失败: status=%d code=%d msg=%s", username, status, response.Code, response.Msg)
	}

	var session struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(response.Data, &session); err != nil {
		t.Fatalf("解析登录响应失败: %v", err)
	}
	if session.Token == "" {
		t.Fatal("登录响应未返回 token")
	}
	return session.Token
}

// itemList 是物品列表响应的解析目标。
type itemList struct {
	Items []struct {
		ItemID       uint     `json:"item_id"`
		Type         string   `json:"type"`
		ItemName     string   `json:"item_name"`
		HappenTime   string   `json:"happen_time"`
		ItemStatus   int      `json:"item_status"`
		Image        []string `json:"image"`
		PosterID     uint     `json:"poster_id"`
		RejectReason string   `json:"reject_reason"`
	} `json:"items"`
}

// listItems 拉取一个列表接口。
func listItems(t *testing.T, engine http.Handler, path, token string) itemList {
	t.Helper()

	status, response := do(t, engine, http.MethodGet, path, token, nil)
	if status != http.StatusOK || response.Code != 0 {
		t.Fatalf("拉取 %s 失败: status=%d code=%d msg=%s", path, status, response.Code, response.Msg)
	}

	var parsed itemList
	if err := json.Unmarshal(response.Data, &parsed); err != nil {
		t.Fatalf("解析 %s 响应失败: %v", path, err)
	}
	return parsed
}

/* ────────────────────────── 断言用的小工具 ────────────────────────── */

// expectCode 断言业务错误码与 HTTP 状态码同时符合预期。
func expectCode(t *testing.T, label string, status int, response envelope, wantCode, wantStatus int) {
	t.Helper()
	if response.Code != wantCode || status != wantStatus {
		t.Fatalf("%s: 期望 code=%d http=%d，实际 code=%d http=%d (msg=%s)",
			label, wantCode, wantStatus, response.Code, status, response.Msg)
	}
}

// expectOK 断言成功响应。
func expectOK(t *testing.T, label string, status int, response envelope) {
	t.Helper()
	if status != http.StatusOK || response.Code != 0 {
		t.Fatalf("%s: 期望成功，实际 http=%d code=%d msg=%s", label, status, response.Code, response.Msg)
	}
}

/* ────────────────────────── 用例 ────────────────────────── */

// TestEnvelopeShape 校验统一信封：字段齐全、data 为 null 时不被省略、空列表为 []。
func TestEnvelopeShape(t *testing.T) {
	engine := newTestServer(t)

	status, response := do(t, engine, http.MethodGet, "/api/items/list/lost?page_size=1", "", nil)
	expectOK(t, "公开列表", status, response)

	if string(response.Data) == "null" {
		t.Fatal("列表接口的 data 不应为 null")
	}

	items := listItems(t, engine, "/api/items/list/lost?page_size=1", "")
	if len(items.Items) != 1 {
		t.Fatalf("期望 1 条，实际 %d 条", len(items.Items))
	}
	// 空图集必须是 []，前端会对它直接做遍历。
	if items.Items[0].Image == nil {
		t.Fatal("image 字段应为 []，不应为 null")
	}

	// 退出登录无请求体，data 必须显式为 null。
	status, response = do(t, engine, http.MethodPost, "/api/auth/logout", login(t, engine, "student001", "abc123"), nil)
	expectOK(t, "退出登录", status, response)
	if string(response.Data) != "null" {
		t.Fatalf("退出登录的 data 应为 null，实际 %s", response.Data)
	}
}

// TestPublicListOnlyPublished 公开信息流只返回已发布，且按发现/遗失时间倒序。
func TestPublicListOnlyPublished(t *testing.T) {
	engine := newTestServer(t)

	items := listItems(t, engine, "/api/items/list/lost?page_size=100", "")
	if len(items.Items) == 0 {
		t.Fatal("公开列表不应为空")
	}
	for _, item := range items.Items {
		if item.ItemStatus != 1 {
			t.Fatalf("公开列表混入了非已发布帖子: item_id=%d status=%d", item.ItemID, item.ItemStatus)
		}
		if item.Type != "lost" {
			t.Fatalf("lost 列表混入了 %s 类型", item.Type)
		}
	}
	for index := 1; index < len(items.Items); index++ {
		if items.Items[index-1].HappenTime < items.Items[index].HappenTime {
			t.Fatalf("未按 happen_time 倒序: %s 排在 %s 之前",
				items.Items[index-1].HappenTime, items.Items[index].HappenTime)
		}
	}

	// 待审核与已驳回的两条不应出现在公开列表里。
	for _, item := range items.Items {
		if item.ItemName == "白色蓝牙音箱" {
			t.Fatal("待审核帖子出现在了公开列表")
		}
	}

	// 分类筛选与关键词筛选
	filtered := listItems(t, engine, "/api/items/list/found?page_size=100&category=卡证", "")
	for _, item := range filtered.Items {
		if item.Image == nil {
			t.Fatal("image 应为空切片")
		}
	}
	keyword := listItems(t, engine, "/api/items/list/lost?page_size=100&keyword=雨伞", "")
	if len(keyword.Items) == 0 {
		t.Fatal("关键词筛选「雨伞」应命中至少一条")
	}
}

// TestPagination 分页口径：第二页与第一页不重复，超出范围返回空数组。
func TestPagination(t *testing.T) {
	engine := newTestServer(t)

	first := listItems(t, engine, "/api/items/list/lost?page=1&page_size=5", "")
	second := listItems(t, engine, "/api/items/list/lost?page=2&page_size=5", "")
	if len(first.Items) != 5 || len(second.Items) != 5 {
		t.Fatalf("分页条数不符: 第一页 %d 条，第二页 %d 条", len(first.Items), len(second.Items))
	}
	seen := map[uint]bool{}
	for _, item := range first.Items {
		seen[item.ItemID] = true
	}
	for _, item := range second.Items {
		if seen[item.ItemID] {
			t.Fatalf("第二页与第一页重复: item_id=%d", item.ItemID)
		}
	}

	empty := listItems(t, engine, "/api/items/list/lost?page=99&page_size=5", "")
	if len(empty.Items) != 0 {
		t.Fatalf("越界页应返回空数组，实际 %d 条", len(empty.Items))
	}
}

// TestTrailingSlash 尾斜杠兼容：文档里带尾斜杠的路径，两种写法都应直达处理器，
// 而不是走 307 重定向。
func TestTrailingSlash(t *testing.T) {
	engine := newTestServer(t)

	paths := []string{
		"/api/items/list/lost",
		"/api/items/list/lost/",
		"/api/announcements",
		"/api/announcements/",
	}
	for _, path := range paths {
		status, response := do(t, engine, http.MethodGet, path, "", nil)
		if status == http.StatusTemporaryRedirect || status == http.StatusMovedPermanently {
			t.Fatalf("%s 被重定向了（http=%d），应直接命中路由", path, status)
		}
		expectOK(t, path, status, response)
	}
}

// TestAuthFlow 注册冲突、登录失败、未登录与档案读取。
func TestAuthFlow(t *testing.T) {
	engine := newTestServer(t)

	// 注册：字段缺失 → 1/400
	status, response := do(t, engine, http.MethodPost, "/api/auth/register", "", map[string]string{
		"username": "newuser",
	})
	expectCode(t, "注册缺字段", status, response, 1, 400)

	// 注册：成功
	status, response = do(t, engine, http.MethodPost, "/api/auth/register", "", map[string]string{
		"username": "newuser", "password": "abc123", "nickname": "新同学", "contact": "new@campus.edu",
	})
	expectOK(t, "注册成功", status, response)

	// 注册：用户名已存在 → 5/409
	status, response = do(t, engine, http.MethodPost, "/api/auth/register", "", map[string]string{
		"username": "newuser", "password": "abc123", "nickname": "新同学", "contact": "new@campus.edu",
	})
	expectCode(t, "重复注册", status, response, 5, 409)

	// 登录：密码错误 → 6/401
	status, response = do(t, engine, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "newuser", "password": "wrong-password",
	})
	expectCode(t, "密码错误", status, response, 6, 401)

	// 登录：用户名不存在 → 6/401（不区分「不存在」与「密码错」）
	status, response = do(t, engine, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "nobody", "password": "abc123",
	})
	expectCode(t, "用户不存在", status, response, 6, 401)

	// 未登录访问需登录接口 → 2/401
	status, response = do(t, engine, http.MethodGet, "/api/auth/profile", "", nil)
	expectCode(t, "未登录取档案", status, response, 2, 401)

	// 伪造凭证 → 2/401
	status, response = do(t, engine, http.MethodGet, "/api/auth/profile", "not-a-real-token", nil)
	expectCode(t, "非法凭证", status, response, 2, 401)

	// 正常取档案
	status, response = do(t, engine, http.MethodGet, "/api/auth/profile", login(t, engine, "student001", "abc123"), nil)
	expectOK(t, "取档案", status, response)
	var profile struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if err := json.Unmarshal(response.Data, &profile); err != nil {
		t.Fatalf("解析档案失败: %v", err)
	}
	if profile.ID == 0 || profile.Username != "student001" || profile.Role != "user" {
		t.Fatalf("档案字段不符: %+v", profile)
	}
}

// TestProfileAndPassword 修改资料与改密码的错误码。
func TestProfileAndPassword(t *testing.T) {
	engine := newTestServer(t)
	token := login(t, engine, "student001", "abc123")

	// 改资料：正常
	status, response := do(t, engine, http.MethodPut, "/api/auth/profile", token, map[string]string{
		"nickname": "张同学（改）", "contact": "zhang@campus.edu",
	})
	expectOK(t, "改资料", status, response)

	// 改资料：越权改他人（query 里给另一个 user_id）→ 3/403
	status, response = do(t, engine, http.MethodPut, "/api/auth/profile?user_id=999", token, map[string]string{
		"nickname": "越权", "contact": "x@campus.edu",
	})
	expectCode(t, "越权改资料", status, response, 3, 403)

	// 改密码：旧密码错误 → 1/400
	status, response = do(t, engine, http.MethodPut, "/api/auth/password", token, map[string]string{
		"old_password": "wrong", "new_password": "another",
	})
	expectCode(t, "旧密码错误", status, response, 1, 400)

	// 改密码：新旧相同 → 11/400
	status, response = do(t, engine, http.MethodPut, "/api/auth/password", token, map[string]string{
		"old_password": "abc123", "new_password": "abc123",
	})
	expectCode(t, "新旧密码相同", status, response, 11, 400)

	// 改密码：正常，且旧密码随即失效
	status, response = do(t, engine, http.MethodPut, "/api/auth/password", token, map[string]string{
		"old_password": "abc123", "new_password": "abc12345",
	})
	expectOK(t, "改密码", status, response)

	status, response = do(t, engine, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "student001", "password": "abc123",
	})
	expectCode(t, "旧密码已失效", status, response, 6, 401)
}

// TestRoleGuards 角色权限：普通用户进不了后台，失物招领管理员进不了系统管理。
func TestRoleGuards(t *testing.T) {
	engine := newTestServer(t)

	userToken := login(t, engine, "student001", "abc123")
	finderToken := login(t, engine, "finder001", "abc123")

	// 普通用户访问后台审核台 → 3/403
	status, response := do(t, engine, http.MethodGet, "/api/admin/items/pending/lost", userToken, nil)
	expectCode(t, "普通用户进审核台", status, response, 3, 403)

	// 普通用户访问全校总览 → 3/403
	status, response = do(t, engine, http.MethodGet, "/api/admin/items", userToken, nil)
	expectCode(t, "普通用户进总览", status, response, 3, 403)

	// 失物招领管理员进得了审核台
	status, response = do(t, engine, http.MethodGet, "/api/admin/items/pending/lost", finderToken, nil)
	expectOK(t, "管理员进审核台", status, response)

	// 但进不了系统管理（总览 / 用户 / 公告管理）→ 3/403
	for _, path := range []string{"/api/admin/items", "/api/admin/users", "/api/admin/announcements"} {
		status, response = do(t, engine, http.MethodGet, path, finderToken, nil)
		expectCode(t, "失物招领管理员访问 "+path, status, response, 3, 403)
	}

	// 未登录访问后台 → 2/401
	status, response = do(t, engine, http.MethodGet, "/api/admin/items", "", nil)
	expectCode(t, "未登录进后台", status, response, 2, 401)
}

// TestAdminRoleRules 角色调整的两条特殊规则：降级自己 7，改其他系统管理员 3。
func TestAdminRoleRules(t *testing.T) {
	engine := newTestServer(t)
	adminToken := login(t, engine, "admin", "admin123")

	// 取 admin 自己的 ID 与 admin2 的 ID
	status, response := do(t, engine, http.MethodGet, "/api/admin/users?page_size=100", adminToken, nil)
	expectOK(t, "用户列表", status, response)

	var users struct {
		Users []struct {
			UserID   uint   `json:"user_id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"users"`
	}
	if err := json.Unmarshal(response.Data, &users); err != nil {
		t.Fatalf("解析用户列表失败: %v", err)
	}
	if len(users.Users) == 0 {
		t.Fatal("用户列表为空")
	}

	ids := map[string]uint{}
	for _, user := range users.Users {
		ids[user.Username] = user.UserID
	}

	// 降级自己 → 7/405
	status, response = do(t, engine, http.MethodPut,
		fmt.Sprintf("/api/admin/users/%d/role", ids["admin"]), adminToken, map[string]string{"role": "user"})
	expectCode(t, "降级自己", status, response, 7, 405)

	// 改另一个系统管理员 → 3/403
	status, response = do(t, engine, http.MethodPut,
		fmt.Sprintf("/api/admin/users/%d/role", ids["admin2"]), adminToken, map[string]string{"role": "user"})
	expectCode(t, "改其他系统管理员", status, response, 3, 403)

	// 角色取值非法 → 1/400
	status, response = do(t, engine, http.MethodPut,
		fmt.Sprintf("/api/admin/users/%d/role", ids["student002"]), adminToken, map[string]string{"role": "super_admin"})
	expectCode(t, "角色取值非法", status, response, 1, 400)

	// 正常提为失物招领管理员
	status, response = do(t, engine, http.MethodPut,
		fmt.Sprintf("/api/admin/users/%d/role", ids["student002"]), adminToken, map[string]string{"role": "finder_admin"})
	expectOK(t, "提升为管理员", status, response)

	// 角色调整立即生效：该用户现有凭证马上就能进审核台（角色取自数据库，非凭证载荷）
	status, response = do(t, engine, http.MethodGet, "/api/admin/items/pending/lost", login(t, engine, "student002", "abc123"), nil)
	expectOK(t, "新管理员立即生效", status, response)

	// 改回普通用户
	status, response = do(t, engine, http.MethodPut,
		fmt.Sprintf("/api/admin/users/%d/role", ids["student002"]), adminToken, map[string]string{"role": "user"})
	expectOK(t, "改回普通用户", status, response)
}

// TestItemVisibility 待审核/已驳回的帖子仅本人与管理员可见。
func TestItemVisibility(t *testing.T) {
	engine := newTestServer(t)

	finderToken := login(t, engine, "finder001", "abc123")
	ownerToken := login(t, engine, "student001", "abc123")

	pending := listItems(t, engine, "/api/admin/items/pending/lost", finderToken)
	if len(pending.Items) == 0 {
		t.Fatal("审核台应有至少一条待审核帖子（种子数据）")
	}
	pendingID := pending.Items[0].ItemID
	detailPath := fmt.Sprintf("/api/items/%d", pendingID)

	// 游客 → 4/404（用 404 而非 403，避免探测出「该 ID 存在但未过审」）
	status, response := do(t, engine, http.MethodGet, detailPath, "", nil)
	expectCode(t, "游客看待审核帖子", status, response, 4, 404)

	// 非本人的普通用户 → 4/404
	status, response = do(t, engine, http.MethodGet, detailPath, login(t, engine, "student002", "abc123"), nil)
	expectCode(t, "他人看待审核帖子", status, response, 4, 404)

	// 本人 → 200
	status, response = do(t, engine, http.MethodGet, detailPath, ownerToken, nil)
	expectOK(t, "本人看待审核帖子", status, response)

	// 管理员 → 200
	status, response = do(t, engine, http.MethodGet, detailPath, finderToken, nil)
	expectOK(t, "管理员看待审核帖子", status, response)

	// 不存在的 ID → 4/404
	status, response = do(t, engine, http.MethodGet, "/api/items/999999", "", nil)
	expectCode(t, "不存在的帖子", status, response, 4, 404)
}

// TestItemLifecycle 发布 → 审核 → 驳回/通过 → 修改重置状态 → 关闭。
func TestItemLifecycle(t *testing.T) {
	engine := newTestServer(t)

	ownerToken := login(t, engine, "student001", "abc123")
	finderToken := login(t, engine, "finder001", "abc123")

	// 发布：必填缺失 → 1/400
	status, response := do(t, engine, http.MethodPost, "/api/items/lost", ownerToken, map[string]any{
		"item_name": "缺字段的帖子",
	})
	expectCode(t, "发布缺必填", status, response, 1, 400)

	// 发布：图片超过 5 张 → 1/400
	tooManyImages := []string{}
	for index := 0; index < 6; index++ {
		tooManyImages = append(tooManyImages, fmt.Sprintf("/api/uploads/%d.png", index))
	}
	status, response = do(t, engine, http.MethodPost, "/api/items/lost", ownerToken, map[string]any{
		"item_name": "图太多", "category": "其他", "get_location": "值班室",
		"get_contact": "a@b.c", "image": tooManyImages,
	})
	expectCode(t, "图片超过 5 张", status, response, 1, 400)

	// 发布：成功，落待审核
	status, response = do(t, engine, http.MethodPost, "/api/items/lost", ownerToken, map[string]any{
		"item_name": "接口测试用的黑色钢笔", "category": "书籍文具", "location": "一号教学楼 101",
		"happen_time": "2026-10-01 10:00:00", "description": "笔帽有一圈磨白",
		"image": []string{"/api/uploads/demo.png"}, "get_location": "一号教学楼值班室",
		"get_contact": "pzx@campus.edu",
	})
	expectOK(t, "发布成功", status, response)

	// 我的发布里能查到，状态为待审核
	mine := listItems(t, engine, "/api/my/items?page_size=100", ownerToken)
	var createdID uint
	for _, item := range mine.Items {
		if item.ItemName == "接口测试用的黑色钢笔" {
			createdID = item.ItemID
			if item.ItemStatus != 0 {
				t.Fatalf("新发布的帖子状态应为 0，实际 %d", item.ItemStatus)
			}
		}
	}
	if createdID == 0 {
		t.Fatal("我的发布里找不到刚提交的帖子")
	}

	// 审核通过 → 出现在公开列表
	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/admin/items/%d/approve", createdID), finderToken, nil)
	expectOK(t, "审核通过", status, response)

	found := false
	for _, item := range listItems(t, engine, "/api/items/list/lost?page_size=100", "").Items {
		if item.ItemID == createdID {
			found = true
		}
	}
	if !found {
		t.Fatal("审核通过后帖子应出现在公开列表")
	}

	// 重复审核 → 7/405
	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/admin/items/%d/approve", createdID), finderToken, nil)
	expectCode(t, "重复审核", status, response, 7, 405)

	// 修改（全段更新）→ 状态重置为待审核，且驳回理由被清空
	status, response = do(t, engine, http.MethodPut, fmt.Sprintf("/api/items/%d", createdID), ownerToken, map[string]any{
		"type": "lost", "item_name": "接口测试用的黑色钢笔（已改）", "category": "书籍文具",
		"location": "一号教学楼 102", "happen_time": "2026-10-02 09:00:00",
		"description": "改了描述", "image": []string{}, "get_location": "一号教学楼值班室",
		"get_contact": "pzx@campus.edu",
	})
	expectOK(t, "修改帖子", status, response)

	reset := false
	for _, item := range listItems(t, engine, "/api/my/items?page_size=100", ownerToken).Items {
		if item.ItemID == createdID {
			reset = item.ItemStatus == 0 && item.RejectReason == ""
		}
	}
	if !reset {
		t.Fatal("修改后状态应重置为待审核且清空驳回理由")
	}

	// 驳回（带理由）→ 我的发布里能看到理由
	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/admin/items/%d/reject", createdID), finderToken,
		map[string]string{"reject_reason": "描述还不够具体"})
	expectOK(t, "驳回", status, response)

	rejected := false
	for _, item := range listItems(t, engine, "/api/my/items?page_size=100", ownerToken).Items {
		if item.ItemID == createdID {
			rejected = item.ItemStatus == 2 && item.RejectReason == "描述还不够具体"
		}
	}
	if !rejected {
		t.Fatal("驳回后应能在我的发布里读到驳回理由")
	}

	// 他人在未获管理权限时改不了我的帖子 → 3/403
	status, response = do(t, engine, http.MethodPut, fmt.Sprintf("/api/items/%d", createdID),
		login(t, engine, "student002", "abc123"), map[string]any{
			"item_name": "越权修改", "category": "其他", "get_location": "x", "get_contact": "y",
		})
	expectCode(t, "越权修改他人帖子", status, response, 3, 403)

	// 删除：他人 → 3/403；本人 → 成功
	status, response = do(t, engine, http.MethodDelete, fmt.Sprintf("/api/items/%d", createdID),
		login(t, engine, "student002", "abc123"), nil)
	expectCode(t, "越权删除", status, response, 3, 403)

	status, response = do(t, engine, http.MethodDelete, fmt.Sprintf("/api/items/%d", createdID), ownerToken, nil)
	expectOK(t, "删除自己的帖子", status, response)

	status, response = do(t, engine, http.MethodGet, fmt.Sprintf("/api/items/%d", createdID), "", nil)
	expectCode(t, "删除后查询", status, response, 4, 404)
}

// TestClaimRulesAndCascade 认领申请的提交规则，以及审批通过后的连带变更。
func TestClaimRulesAndCascade(t *testing.T) {
	engine := newTestServer(t)

	finderToken := login(t, engine, "finder001", "abc123")
	studentOneToken := login(t, engine, "student001", "abc123")
	studentTwoToken := login(t, engine, "student002", "abc123")

	// 挑一条招领处发布的失物招领作为认领对象。
	// 刻意避开「白色无线鼠标」：种子数据里 student002 已对它有一条待审批申请，
	// 用它会让「第二人提交」直接撞上重复提交(9)，反而测不出自动驳回那条链路。
	found := listItems(t, engine, "/api/items/list/found?page_size=100", "")
	var targetID uint
	for _, item := range found.Items {
		if item.ItemName != "白色无线鼠标" {
			targetID = item.ItemID
			break
		}
	}
	if targetID == 0 {
		t.Fatal("未找到可用的失物招领目标帖")
	}

	// 申请寻物启事 → 7/405（只有失物招领可认领）
	lostItems := listItems(t, engine, "/api/items/list/lost?page_size=1", "")
	status, response := do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/claims?item_id=%d", lostItems.Items[0].ItemID), studentOneToken, map[string]string{"reason": "这是我的东西"})
	expectCode(t, "认领寻物启事", status, response, 7, 405)

	// 申请不存在的物品 → 4/404
	status, response = do(t, engine, http.MethodPost, "/api/claims?item_id=999999", studentOneToken,
		map[string]string{"reason": "不存在"})
	expectCode(t, "认领不存在的物品", status, response, 4, 404)

	// 发布者申请自己的帖子 → 7（拿招领处的凭证申请招领处发布的帖子）
	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/claims?item_id=%d", targetID), finderToken, map[string]string{"reason": "我自己发的"})
	expectCode(t, "申请自己的帖子", status, response, 7, 405)

	// 理由为空 → 1/400
	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/claims?item_id=%d", targetID), studentOneToken, map[string]string{"reason": "   "})
	expectCode(t, "空理由", status, response, 1, 400)

	// 正常提交
	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/claims?item_id=%d", targetID), studentOneToken, map[string]string{"reason": "描述的特征与我的一致"})
	expectOK(t, "提交认领", status, response)

	// 重复提交 → 9/409
	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/claims?item_id=%d", targetID), studentOneToken, map[string]string{"reason": "再提一次"})
	expectCode(t, "重复认领", status, response, 9, 409)

	// 第二个申请人提交同一物品，用于验证自动驳回
	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/claims?item_id=%d", targetID), studentTwoToken, map[string]string{"reason": "我也觉得是我的"})
	expectOK(t, "第二人提交认领", status, response)

	// 找到第一条待审批申请并审批通过
	claims := fetchClaims(t, engine, "/api/admin/claims?claim_status=0&page_size=100", finderToken)
	var approvedClaimID uint
	var siblingClaimID uint
	for _, claim := range claims {
		if claim.ItemID == targetID {
			if approvedClaimID == 0 {
				approvedClaimID = claim.ClaimID
			} else {
				siblingClaimID = claim.ClaimID
			}
		}
	}
	if approvedClaimID == 0 || siblingClaimID == 0 {
		t.Fatalf("未找到同一物品上的两条待审批申请: %+v", claims)
	}

	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/admin/claims/%d/approve", approvedClaimID), finderToken, nil)
	expectOK(t, "审批通过", status, response)

	// 物品应变为已关闭
	status, response = do(t, engine, http.MethodGet, fmt.Sprintf("/api/items/%d", targetID), "", nil)
	expectOK(t, "审批后查物品", status, response)
	var detail struct {
		ItemStatus int `json:"item_status"`
	}
	if err := json.Unmarshal(response.Data, &detail); err != nil {
		t.Fatalf("解析物品详情失败: %v", err)
	}
	if detail.ItemStatus != 3 {
		t.Fatalf("审批通过后物品状态应为 3，实际 %d", detail.ItemStatus)
	}

	// 同物品的另一条申请应被自动驳回
	rejected := fetchClaims(t, engine, "/api/admin/claims?claim_status=2&page_size=100", finderToken)
	cascade := false
	for _, claim := range rejected {
		if claim.ClaimID == siblingClaimID {
			cascade = true
		}
	}
	if !cascade {
		t.Fatal("审批通过后，同物品的其他待审批申请应被自动驳回")
	}

	// 重复审批 → 7/405
	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/admin/claims/%d/approve", approvedClaimID), finderToken, nil)
	expectCode(t, "重复审批", status, response, 7, 405)

	// 已关闭的物品不能再被认领 → 7/405
	status, response = do(t, engine, http.MethodPost,
		fmt.Sprintf("/api/claims?item_id=%d", targetID), login(t, engine, "student001", "abc123"),
		map[string]string{"reason": "关闭后还想认领"})
	expectCode(t, "认领已关闭物品", status, response, 7, 405)
}

// claimRow 是认领列表里的一行。
type claimRow struct {
	ClaimID     uint   `json:"claim_id"`
	ItemID      uint   `json:"item_id"`
	Reason      string `json:"reason"`
	ClaimStatus int    `json:"claim_status"`
	ApplicantID uint   `json:"applicant_id"`
}

// claimList 是认领列表响应的解析目标。
type claimList struct {
	Claims []claimRow `json:"claims"`
}

// fetchClaims 拉取认领申请列表。
func fetchClaims(t *testing.T, engine http.Handler, path, token string) []claimRow {
	t.Helper()

	status, response := do(t, engine, http.MethodGet, path, token, nil)
	expectOK(t, path, status, response)

	var parsed claimList
	if err := json.Unmarshal(response.Data, &parsed); err != nil {
		t.Fatalf("解析 %s 失败: %v", path, err)
	}
	return parsed.Claims
}

// TestClaimPermissions 认领申请的可见性与越权。
func TestClaimPermissions(t *testing.T) {
	engine := newTestServer(t)

	studentToken := login(t, engine, "student001", "abc123")
	finderToken := login(t, engine, "finder001", "abc123")

	// 种子数据里 student001 有一条已通过的申请
	mine := fetchClaims(t, engine, "/api/my/claims?page_size=100", studentToken)
	if len(mine) == 0 {
		t.Fatal("我的认领申请不应为空（种子数据）")
	}
	claimID := mine[0].ClaimID

	// 本人可看详情
	status, response := do(t, engine, http.MethodGet, fmt.Sprintf("/api/claims/%d", claimID), studentToken, nil)
	expectOK(t, "本人看申请详情", status, response)

	// 详情里应内联物品
	var detail struct {
		ClaimID uint `json:"claim_id"`
		Item    *struct {
			ItemID uint `json:"item_id"`
		} `json:"item"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(response.Data, &detail); err != nil {
		t.Fatalf("解析申请详情失败: %v", err)
	}
	if detail.Item == nil || detail.Item.ItemID == 0 {
		t.Fatal("申请详情应内联对应的物品")
	}

	// 他人看 → 3/403
	status, response = do(t, engine, http.MethodGet, fmt.Sprintf("/api/claims/%d", claimID),
		login(t, engine, "student002", "abc123"), nil)
	expectCode(t, "他人看申请详情", status, response, 3, 403)

	// 管理员可看
	status, response = do(t, engine, http.MethodGet, fmt.Sprintf("/api/claims/%d", claimID), finderToken, nil)
	expectOK(t, "管理员看申请详情", status, response)

	// 他人改 → 3/403
	status, response = do(t, engine, http.MethodPut, fmt.Sprintf("/api/claims/%d", claimID),
		login(t, engine, "student002", "abc123"), map[string]string{"reason": "改成我的"})
	expectCode(t, "他人改申请", status, response, 3, 403)

	// 他人删 → 3/403（普通用户不能删别人的申请）
	status, response = do(t, engine, http.MethodDelete, fmt.Sprintf("/api/claims/%d", claimID),
		login(t, engine, "student002", "abc123"), nil)
	expectCode(t, "他人删申请", status, response, 3, 403)

	// 不存在 → 4/404
	status, response = do(t, engine, http.MethodGet, "/api/claims/999999", studentToken, nil)
	expectCode(t, "不存在的申请", status, response, 4, 404)
}

// TestAnnouncementFlow 公告的公开列表只给公开项，管理端能看到已下线项。
func TestAnnouncementFlow(t *testing.T) {
	engine := newTestServer(t)
	adminToken := login(t, engine, "admin", "admin123")

	type announcementList struct {
		Announcements []struct {
			AnnouncementID     uint   `json:"announcement_id"`
			Title              string `json:"title"`
			AnnouncementStatus int    `json:"announcement_status"`
		} `json:"announcements"`
	}

	// 公开列表：全部为公开状态
	status, response := do(t, engine, http.MethodGet, "/api/announcements/", "", nil)
	expectOK(t, "公告公开列表", status, response)
	var public announcementList
	if err := json.Unmarshal(response.Data, &public); err != nil {
		t.Fatalf("解析公告列表失败: %v", err)
	}
	if len(public.Announcements) == 0 {
		t.Fatal("公开公告列表不应为空（种子数据）")
	}
	for _, item := range public.Announcements {
		if item.AnnouncementStatus != 0 {
			t.Fatalf("公开列表混入了已下线公告: %d", item.AnnouncementID)
		}
	}

	// 管理端：能看到已下线的那条
	status, response = do(t, engine, http.MethodGet, "/api/admin/announcements/", adminToken, nil)
	expectOK(t, "公告管理列表", status, response)
	var all announcementList
	if err := json.Unmarshal(response.Data, &all); err != nil {
		t.Fatalf("解析管理端公告列表失败: %v", err)
	}
	if len(all.Announcements) <= len(public.Announcements) {
		t.Fatalf("管理端公告数(%d)应多于公开列表(%d)", len(all.Announcements), len(public.Announcements))
	}

	// 发布公告：缺正文 → 1/400
	status, response = do(t, engine, http.MethodPost, "/api/admin/announcements/", adminToken,
		map[string]string{"title": "只有标题"})
	expectCode(t, "公告缺正文", status, response, 1, 400)

	// 发布公告：成功
	status, response = do(t, engine, http.MethodPost, "/api/admin/announcements/", adminToken,
		map[string]string{"title": "接口测试公告", "content": "这是接口测试写入的公告正文。"})
	expectOK(t, "发布公告", status, response)

	// 找到刚发布的公告
	var createdID uint
	status, response = do(t, engine, http.MethodGet, "/api/admin/announcements/?page_size=100", adminToken, nil)
	expectOK(t, "重新拉取公告", status, response)
	if err := json.Unmarshal(response.Data, &all); err != nil {
		t.Fatalf("解析公告列表失败: %v", err)
	}
	for _, item := range all.Announcements {
		if item.Title == "接口测试公告" {
			createdID = item.AnnouncementID
		}
	}
	if createdID == 0 {
		t.Fatal("找不到刚发布的公告")
	}

	// 下线（借 PUT 携带 announcement_status）→ 公开列表看不到，管理端仍看得到
	status, response = do(t, engine, http.MethodPut,
		fmt.Sprintf("/api/admin/announcements/%d", createdID), adminToken,
		map[string]any{"title": "接口测试公告", "content": "这是接口测试写入的公告正文。", "announcement_status": 1})
	expectOK(t, "公告下线", status, response)

	var offlineFound bool
	status, response = do(t, engine, http.MethodGet, "/api/announcements/?page_size=100", "", nil)
	expectOK(t, "下线后公开列表", status, response)
	if err := json.Unmarshal(response.Data, &public); err != nil {
		t.Fatalf("解析公告列表失败: %v", err)
	}
	for _, item := range public.Announcements {
		if item.AnnouncementID == createdID {
			offlineFound = true
		}
	}
	if offlineFound {
		t.Fatal("已下线公告不应出现在公开列表")
	}

	// 删除
	status, response = do(t, engine, http.MethodDelete,
		fmt.Sprintf("/api/admin/announcements/%d", createdID), adminToken, nil)
	expectOK(t, "删除公告", status, response)

	// 公共用户不能改公告 → 3/403
	status, response = do(t, engine, http.MethodPost, "/api/admin/announcements/",
		login(t, engine, "student001", "abc123"), map[string]string{"title": "越权", "content": "越权"})
	expectCode(t, "普通用户发公告", status, response, 3, 403)
}

// TestAdminItemStatusAndOverview 全校总览的筛选，以及管理端置为已关闭。
func TestAdminItemStatusAndOverview(t *testing.T) {
	engine := newTestServer(t)
	adminToken := login(t, engine, "admin", "admin123")

	// 总览应包含非已发布的帖子（与公开列表的区别所在）
	all := listItems(t, engine, "/api/admin/items?page_size=100", adminToken)
	public := listItems(t, engine, "/api/items/list/lost?page_size=100", "")
	if len(all.Items) <= len(public.Items) {
		t.Fatalf("总览(%d)应多于公开列表(%d)", len(all.Items), len(public.Items))
	}

	// 按状态筛选 pending
	pending := listItems(t, engine, "/api/admin/items?status=0&page_size=100", adminToken)
	for _, item := range pending.Items {
		if item.ItemStatus != 0 {
			t.Fatalf("状态筛选失效: item_id=%d status=%d", item.ItemID, item.ItemStatus)
		}
	}
	if len(pending.Items) == 0 {
		t.Fatal("应至少有一条待审核帖子（种子数据）")
	}

	// 置为已关闭
	targetID := pending.Items[0].ItemID
	status, response := do(t, engine, http.MethodPut,
		fmt.Sprintf("/api/admin/items/%d", targetID), adminToken, map[string]int{"status": 3})
	expectOK(t, "管理端关闭物品", status, response)

	// 只允许改成 3
	status, response = do(t, engine, http.MethodPut,
		fmt.Sprintf("/api/admin/items/%d", targetID), adminToken, map[string]int{"status": 1})
	expectCode(t, "非法状态值", status, response, 1, 400)
}

// TestUpload 上传的格式与体积限制。
func TestUpload(t *testing.T) {
	engine := newTestServer(t)
	token := login(t, engine, "student001", "abc123")

	// 不支持的格式 → 8
	status, response := uploadFile(t, engine, token, "notes.txt", []byte("我不是图片"))
	expectCode(t, "上传 txt", status, response, 8, 406)

	// 改后缀的伪装文件 → 8（内容嗅探拦下）
	status, response = uploadFile(t, engine, token, "fake.png", []byte("我不是图片，只是改了个后缀"))
	expectCode(t, "伪装成 png 的文本", status, response, 8, 406)

	// 合法 PNG（仅签名，够内容嗅探识别）→ 成功并返回 URL
	pngSignature := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	status, response = uploadFile(t, engine, token, "real.png", pngSignature)
	expectOK(t, "上传 png", status, response)

	var payload struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(response.Data, &payload); err != nil {
		t.Fatalf("解析上传响应失败: %v", err)
	}
	if !strings.HasPrefix(payload.URL, "/api/uploads/") {
		t.Fatalf("上传返回的 URL 前缀不符: %q", payload.URL)
	}

	// 未登录上传 → 2/401
	status, response = uploadFile(t, engine, "", "real.png", pngSignature)
	expectCode(t, "未登录上传", status, response, 2, 401)
}

// uploadFile 构造一个 multipart 请求完成上传。
func uploadFile(t *testing.T, engine http.Handler, token, filename string, content []byte) (int, envelope) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("构造 multipart 失败: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("写入 multipart 失败: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭 multipart 失败: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	var parsed envelope
	if recorder.Body.Len() > 0 {
		if err := json.Unmarshal(recorder.Body.Bytes(), &parsed); err != nil {
			t.Fatalf("上传响应不是合法 JSON: %s", recorder.Body.String())
		}
	}
	return recorder.Code, parsed
}

// TestUnknownRouteUnknownMethod 未知路径与错误方法都回 4/404。
func TestUnknownRouteUnknownMethod(t *testing.T) {
	engine := newTestServer(t)

	status, response := do(t, engine, http.MethodGet, "/api/nope", "", nil)
	expectCode(t, "未知路径", status, response, 4, 404)

	// 已知路径用错方法（PUT 一条只支持 GET 的路径）
	status, response = do(t, engine, http.MethodPut, "/api/items/list/lost", "", nil)
	expectCode(t, "错误方法", status, response, 4, 404)
}
