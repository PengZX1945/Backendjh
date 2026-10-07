// Package router 是组合根与路由表。
//
// 装配顺序：仓储 → 服务 → 处理器 → 路由，全部收在本包的 New 里。这样 main 只负责
// 「读配置、连数据库、起 HTTP、优雅退出」，依赖关系在一处可见，测试也能直接拿到
// 一个装配完整的 *gin.Engine（见 router_test.go）。
package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lostfound/internal/config"
	"lostfound/internal/handler"
	"lostfound/internal/middleware"
	"lostfound/internal/model"
	"lostfound/internal/repository"
	"lostfound/internal/service"
	"lostfound/internal/token"
	"lostfound/pkg/apperr"
	"lostfound/pkg/errcode"
)

// 审核动作的两种取值。与接口文档的 {option} 路径段对应。
const (
	reviewApprove = "approve"
	reviewReject  = "reject"
)

// New 装配依赖与全部路由，返回可直接交给 http.Server 的引擎。
func New(cfg *config.Config, db *gorm.DB) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// —— 仓储层 ——
	userRepository := repository.NewUserRepository(db)
	itemRepository := repository.NewItemRepository(db)
	claimRepository := repository.NewClaimRepository(db)
	announcementRepository := repository.NewAnnouncementRepository(db)

	// —— 基础设施 ——
	tokenManager := token.NewManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpireHours, cfg.Auth.Issuer)

	// —— 服务层 ——
	authService := service.NewAuthService(userRepository, tokenManager)
	itemService := service.NewItemService(itemRepository)
	claimService := service.NewClaimService(claimRepository, itemRepository)
	announcementService := service.NewAnnouncementService(announcementRepository)
	uploadService := service.NewUploadService(cfg.Upload)

	// —— 处理器 ——
	authHandler := handler.NewAuthHandler(authService)
	itemHandler := handler.NewItemHandler(itemService)
	claimHandler := handler.NewClaimHandler(claimService)
	announcementHandler := handler.NewAnnouncementHandler(announcementService)
	uploadHandler := handler.NewUploadHandler(uploadService)

	engine := gin.New()

	// 中间件顺序即包裹层次（外 → 内）。两条讲究：
	//   - ErrorHandler 必须在所有路由之前，才能收口整条链上抛出的业务异常；
	//   - AccessLog 在 ErrorHandler 之外层，才记录得到被翻译后的最终状态码。
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS())
	engine.Use(middleware.AccessLog())
	engine.Use(middleware.ErrorHandler())

	// —— 三个鉴权档位 ——
	// RequireRole 自带凭证校验（见其注释），因此后台档位只需一个中间件；
	// 它内部也会把用户写进上下文，处理器仍可直接 currentUser(c) 取用。
	authenticated := middleware.Authenticate(tokenManager, userRepository)
	optionalAuth := middleware.OptionalAuthenticate(tokenManager, userRepository)
	backOffice := middleware.RequireRole(tokenManager, userRepository, model.RoleFinderAdmin, model.RoleSysAdmin)
	systemAdmin := middleware.RequireRole(tokenManager, userRepository, model.RoleSysAdmin)

	api := engine.Group("/api")

	registerAuthRoutes(api, authHandler, authenticated, systemAdmin)
	registerItemRoutes(api, itemHandler, authenticated, optionalAuth, backOffice, systemAdmin)
	registerClaimRoutes(api, claimHandler, authenticated, backOffice)
	registerAnnouncementRoutes(api, announcementHandler, systemAdmin)
	registerUploadRoutes(api, uploadHandler, authenticated)

	// 未命中的路径与方法也要走统一信封：gin 默认回一段纯文本 "404 page not found"，
	// 那会破坏「所有响应都是 {code,msg,data}」的约定，前端解析起来也会多一种形态。
	unmatched := func(c *gin.Context) {
		c.Error(apperr.New(errcode.NotFound))
	}
	engine.NoRoute(unmatched)
	engine.NoMethod(unmatched)

	// 上传的图片以静态目录对外提供。前缀挂在 /api 之下，前端开发态的 vite 代理
	// 与生产态的 Nginx 反代 /api 都能直接命中，无需额外配置一条转发规则。
	engine.Static(cfg.Upload.URLPrefix, cfg.Upload.Dir)

	return engine
}

// register 同时登记「带尾斜杠」与「不带尾斜杠」两种路径形态。
//
// 接口文档里多个路径带尾斜杠（/api/items/list/{type}/、/api/my/claims/…），前端也按
// 原样请求，而同一份文档里另一些路径又不带。若只登记其中一种，另一种会落到 gin 的
// 307 重定向：对 POST/PUT/DELETE 而言多一次往返，且不同代理对 307 的处理并不一致。
// 两种形态都登记，请求无论怎么写都能直达处理器。
func register(group *gin.RouterGroup, method, path string, handlers ...gin.HandlerFunc) {
	trimmed := strings.TrimSuffix(path, "/")
	group.Handle(method, trimmed, handlers...)
	group.Handle(method, trimmed+"/", handlers...)
}

// registerAuthRoutes 注册鉴权与系统管理（用户）路由。
func registerAuthRoutes(
	api *gin.RouterGroup,
	authHandler *handler.AuthHandler,
	authenticated, systemAdmin gin.HandlerFunc,
) {
	// 公开接口
	register(api, http.MethodPost, "/auth/register", authHandler.Register)
	register(api, http.MethodPost, "/auth/login", authHandler.Login)

	// 需登录
	register(api, http.MethodPost, "/auth/logout", authenticated, authHandler.Logout)
	register(api, http.MethodGet, "/auth/profile", authenticated, authHandler.Profile)
	register(api, http.MethodPut, "/auth/profile", authenticated, authHandler.UpdateProfile)
	register(api, http.MethodPut, "/auth/password", authenticated, authHandler.ChangePassword)

	// 系统管理：账户与角色
	register(api, http.MethodGet, "/admin/users", systemAdmin, authHandler.ListUsers)
	register(api, http.MethodPut, "/admin/users/:user_id/role", systemAdmin, authHandler.UpdateUserRole)
}

// registerItemRoutes 注册物品信息流、发布修改与发布审核路由。
func registerItemRoutes(
	api *gin.RouterGroup,
	itemHandler *handler.ItemHandler,
	authenticated, optionalAuth, backOffice, systemAdmin gin.HandlerFunc,
) {
	// 公开：信息流与详情
	register(api, http.MethodGet, "/items/list/"+model.ItemTypeLost, itemHandler.List(model.ItemTypeLost))
	register(api, http.MethodGet, "/items/list/"+model.ItemTypeFound, itemHandler.List(model.ItemTypeFound))
	// 详情挂可选鉴权：游客可看已发布的，待审核/已驳回则要靠身份判断是否放行。
	register(api, http.MethodGet, "/items/:item_id", optionalAuth, itemHandler.Detail)

	// 需登录：发布与维护
	//
	// 发布接口在文档里写作 /items/{type}，这里注册成两条静态路径而不是一个 :type。
	// 原因是 gin 的路由树要求「同一位置上出现的通配符必须同名」：POST /items/{type}
	// 与 POST /items/{item_id}/close 会在同一层出现 :type 与 :item_id 两个不同名的
	// 通配符，注册时直接 panic。拆成 lost / found 两条静态路径既绕开了这个限制，
	// 也顺带把「大类只能是这两个值」变成了路由层的约束。
	register(api, http.MethodPost, "/items/"+model.ItemTypeLost, authenticated, itemHandler.Create(model.ItemTypeLost))
	register(api, http.MethodPost, "/items/"+model.ItemTypeFound, authenticated, itemHandler.Create(model.ItemTypeFound))
	register(api, http.MethodPut, "/items/:item_id", authenticated, itemHandler.Update)
	register(api, http.MethodDelete, "/items/:item_id", authenticated, itemHandler.Delete)
	register(api, http.MethodPost, "/items/:item_id/close", authenticated, itemHandler.CloseClaim)

	// 需登录：我的发布
	register(api, http.MethodGet, "/my/items", authenticated, itemHandler.ListMine)

	// 发布审核（失物招领管理员及以上）
	register(api, http.MethodGet, "/admin/items/pending/"+model.ItemTypeLost, backOffice, itemHandler.ListPending(model.ItemTypeLost))
	register(api, http.MethodGet, "/admin/items/pending/"+model.ItemTypeFound, backOffice, itemHandler.ListPending(model.ItemTypeFound))
	register(api, http.MethodPost, "/admin/items/:item_id/"+reviewApprove, backOffice, itemHandler.Review(reviewApprove))
	register(api, http.MethodPost, "/admin/items/:item_id/"+reviewReject, backOffice, itemHandler.Review(reviewReject))

	// 全校总览与状态管理（系统管理员）
	register(api, http.MethodGet, "/admin/items", systemAdmin, itemHandler.ListAll)
	register(api, http.MethodPut, "/admin/items/:item_id", systemAdmin, itemHandler.SetStatus)
}

// registerClaimRoutes 注册认领申请路由。
func registerClaimRoutes(
	api *gin.RouterGroup,
	claimHandler *handler.ClaimHandler,
	authenticated, backOffice gin.HandlerFunc,
) {
	// 需登录：提交与自助管理
	register(api, http.MethodPost, "/claims", authenticated, claimHandler.Submit)
	register(api, http.MethodGet, "/my/claims", authenticated, claimHandler.ListMine)

	// 需登录：详情、修改、取消
	register(api, http.MethodGet, "/claims/:claim_id", authenticated, claimHandler.Detail)
	register(api, http.MethodPut, "/claims/:claim_id", authenticated, claimHandler.Update)
	register(api, http.MethodDelete, "/claims/:claim_id", authenticated, claimHandler.Delete)

	// 认领审批（失物招领管理员及以上）
	register(api, http.MethodGet, "/admin/claims", backOffice, claimHandler.ListForAdmin)
	register(api, http.MethodPost, "/admin/claims/:claim_id/"+reviewApprove, backOffice, claimHandler.Review(reviewApprove))
	register(api, http.MethodPost, "/admin/claims/:claim_id/"+reviewReject, backOffice, claimHandler.Review(reviewReject))
}

// registerAnnouncementRoutes 注册公告路由。
func registerAnnouncementRoutes(
	api *gin.RouterGroup,
	announcementHandler *handler.AnnouncementHandler,
	systemAdmin gin.HandlerFunc,
) {
	// 公开
	register(api, http.MethodGet, "/announcements", announcementHandler.ListPublic)
	register(api, http.MethodGet, "/announcements/:announcement_id", announcementHandler.Detail)

	// 系统管理
	register(api, http.MethodGet, "/admin/announcements", systemAdmin, announcementHandler.ListAll)
	register(api, http.MethodPost, "/admin/announcements", systemAdmin, announcementHandler.Create)
	register(api, http.MethodPut, "/admin/announcements/:announcement_id", systemAdmin, announcementHandler.Update)
	register(api, http.MethodDelete, "/admin/announcements/:announcement_id", systemAdmin, announcementHandler.Delete)
}

// registerUploadRoutes 注册图片上传。
func registerUploadRoutes(
	api *gin.RouterGroup,
	uploadHandler *handler.UploadHandler,
	authenticated gin.HandlerFunc,
) {
	register(api, http.MethodPost, "/upload", authenticated, uploadHandler.Upload)
}
