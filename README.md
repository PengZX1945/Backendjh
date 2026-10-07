# 校园失物招领系统 — 后端

基于 Go + Gin + GORM + MySQL 8 的 RESTful API 服务。

## 功能板块

| 板块 | 接口 | 说明 |
|---|---|---|
| 用户 | 注册 / 登录 / 登出 / 查改资料 / 改密码 | JWT 鉴权，bcrypt 密码哈希 |
| 上传 | 图片上传 | 类型白名单 + 5MB 上限，`/uploads/` 静态访问 |
| 物品 | 发布 / 列表 / 详情 / 修改 / 删除 / 关闭 / 我的发布 | 状态机：待审核→已发布/已驳回→已认领 |
| 认领申请 | 提交 / 审批 / 详情 / 修改 / 删除 / 我的申请 / 管理列表 | 审批通过全联动：物品转已认领、驳回同物品其他申请 |
| 发布审核 | 待审列表 / 全部物品 / 审核 / 管理置 3 | finder_admin 及以上 |
| 公告 | 公开列表 / 发布 / 修改 / 删除 | 管理员维护，所有人可读 |
| 系统管理 | 用户列表 / 调整角色 / 封禁解封 | 仅 sys_admin；首个管理员由启动种子创建 |
| 统计 | 总览 / 趋势 | 管理端数据看板 |

## 技术栈

| 依赖 | 用途 |
|---|---|
| Gin | HTTP 路由与框架 |
| GORM + MySQL 驱动 | ORM 与数据库访问 |
| golang-jwt/v5 | JWT 鉴权 |
| golang.org/x/crypto | bcrypt 密码哈希 |

## 目录结构

```
Backendjh/
├── cmd/server/main.go        程序入口：加载配置、初始化 DB、种子管理员、注册路由
├── internal/                 私有代码（Go 编译器强制，外部项目无法导入）
│   ├── config/               配置读取（环境变量 + 默认值）
│   ├── model/                GORM 模型、数据库初始化、枚举常量
│   ├── repository/           数据访问层（只碰数据库）
│   ├── service/              业务逻辑层（业务规则与状态流转）
│   ├── handler/              控制器（参数解析、调用 service、统一响应）
│   ├── middleware/           auth（JWT 鉴权/可选鉴权）、error（panic 兜底）、cors
│   ├── router/               路由注册，按模块分组挂载中间件
│   └── pkg/
│       ├── errcode/          错误码定义（一个 code 对应一个 msg）
│       ├── response/         统一响应结构 {code, msg, data}
│       └── jwt/              token 生成与解析
├── uploads/                  图片存储目录（运行时自动创建）
└── go.mod
```

## 快速开始

### 前置要求

- Go ≥ 1.26
- MySQL 8（本机运行或 Docker 均可）

### 1. 初始化数据库

```sql
CREATE DATABASE lost_found CHARACTER SET utf8mb4;
```

表结构由程序启动时 `AutoMigrate` 自动创建，无需手动建表。

### 2. 配置环境变量（可选）

| 变量 | 默认值 | 说明 |
|---|---|---|
| `DB_DSN` | `root:@tcp(127.0.0.1:3306)/lost_found?charset=utf8mb4&parseTime=True&loc=Local` | MySQL 连接串，生产环境必须显式配置 |
| `JWT_SECRET` | `dev-secret-change-me` | JWT 签名密钥，**生产环境必须修改** |
| `PORT` | `8080` | 服务监听端口 |
| `ADMIN_USERNAME` | `admin` | 种子管理员用户名 |
| `ADMIN_PASSWORD` | 空 | 种子管理员密码；**为空则不创建**。配置了且系统无 sys_admin 时，启动自动创建一个 |

### 3. 安装依赖并启动

```bash
go mod tidy
go run ./cmd/server
```

看到 Gin 启动日志、服务监听 `:8080` 即成功。

### 4. 验证

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"zhangsan","password":"abc12345","nickname":"张三","contact":"13800001111"}'
```

返回 `{"code":0,"msg":"success",...}` 即正常。日常调试建议用 Apifox。

## 部署

生产环境标准形态：Nginx 反向代理（80 端口对外，`/api/` 与 `/uploads/` 转发给后端，前端静态文件同源托管）+ systemd 常驻运行。

```bash
# 本地交叉编译
GOOS=linux GOARCH=amd64 go build -o server ./cmd/server
# 上传后在服务器用 systemd 运行，环境变量在 service 文件里配置
```

## 开发约定

### 统一响应

所有接口（无论成败）HTTP 状态码均为 200，业务结果看 `code`：

```json
{ "code": 0, "msg": "success", "data": {} }
```

错误码集中在 `internal/pkg/errcode`，一个 code 唯一对应一个 msg，新增错误码必须先在此处定义，不允许在 handler 里手写错误文案。

### 分层规则

- handler：只做参数绑定/格式校验和响应，不写业务逻辑
- service：业务规则所在地，错误一律返回 `*errcode.Error`
- repository：只做 CRUD，"查不到"返回 `nil` 而非报错，交由 service 判断
- 身份信息（userID/role）一律从 JWT 上下文取，不信任请求体里的身份字段

### Git 规范

- Commit 遵循 Conventional Commits：`feat/fix/refactor/docs/chore: 描述`
- 小步提交：一个接口/一个方法一次 commit
- 协作通过 GitHub PR 合并进 main

## 常见问题

**Q: 启动报数据库连接失败？**
先确认 MySQL 已启动、已执行 `CREATE DATABASE`、DSN 里的用户名密码正确。

**Q: 接口返回 `{"code":10002}`？**
未携带或携带了过期的 token。先调 `/api/auth/login` 拿 token，再在请求头加 `Authorization: Bearer <token>`。

**Q: 改了 model 但表结构没变化？**
`AutoMigrate` 只会新增列，不会修改/删除已有列。开发期可直接删表让程序重建，或手动 `ALTER TABLE`。

**Q: 管理员账号怎么来？**
配置 `ADMIN_PASSWORD` 环境变量后启动，系统自动创建首个 sys_admin（用户名默认 `admin`）；已有 sys_admin 时跳过。接口不提供创建 sys_admin 的能力，防止权限提升。
