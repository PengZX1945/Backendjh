# 校园失物招领系统 — 后端

基于 Go + Gin + GORM + MySQL 8 的 RESTful API 服务。

- 接口契约：见仓库根目录《接口文档.md》
- 架构与数据库设计：见《项目设计文档.md》
- 用户板块手写教程：见《用户板块开发指南.md》

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
├── cmd/server/main.go        程序入口：加载配置、初始化 DB、注册路由
├── internal/                 私有代码（Go 编译器强制，外部项目无法导入）
│   ├── config/               配置读取（环境变量 + 默认值）
│   ├── model/                GORM 模型、数据库初始化、枚举常量
│   ├── repository/           数据访问层（只碰数据库）
│   ├── service/              业务逻辑层（业务规则与状态流转）
│   ├── handler/              控制器（参数解析、调用 service、统一响应）
│   ├── middleware/           auth（JWT 鉴权）、error（统一错误响应）、cors
│   ├── router/               路由注册，按模块分组挂载中间件
│   └── pkg/
│       ├── errcode/          错误码定义（一个 code 对应一个 msg）
│       ├── response/         统一响应结构 {code, msg, data}
│       └── jwt/              token 生成与解析
├── uploads/                  图片存储目录（本地运行时使用）
├── deployments/              init.sql、Dockerfile 等部署相关文件
└── go.mod
```

## 快速开始

### 前置要求

- Go ≥ 1.22
- MySQL 8（本机运行或 Docker 均可）

### 1. 初始化数据库

```sql
CREATE DATABASE lost_found CHARACTER SET utf8mb4;
```

表结构由程序启动时 GORM `AutoMigrate` 自动创建，无需手动建表。

### 2. 配置环境变量（可选，有默认值）

| 变量 | 默认值 | 说明 |
|---|---|---|
| `DB_DSN` | `root:root@tcp(127.0.0.1:3306)/lost_found?charset=utf8mb4&parseTime=True&loc=Local` | MySQL 连接串 |
| `JWT_SECRET` | `dev-secret-change-me` | JWT 签名密钥，**生产环境必须修改** |
| `PORT` | `8080` | 服务监听端口 |

### 3. 安装依赖并启动

```bash
cd server
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

返回 `{"code":0,"msg":"success","data":{...}}` 即正常。日常调试建议用 Apifox，接口与仓库根目录《接口文档.md》一一对应。

## 开发约定

### 统一响应

所有接口（无论成败）HTTP 状态码均为 200，业务结果看 `code`：

```json
{ "code": 0, "msg": "success", "data": {} }
```

错误码集中在 `internal/pkg/errcode`，一个 code 唯一对应一个 msg，新增错误码必须先在此处定义，不允许在 handler 里手写错误文案。

### 分层规则

- handler：只做参数绑定/校验和响应，不写业务逻辑
- service：业务规则所在地，错误一律返回 `*errcode.Error`
- repository：只做 CRUD，"查不到"返回 `nil` 而非报错，交由 service 判断

### Git 规范

- 分支：`feat/server-xxx` 从 `develop` 切出，PR 合回
- Commit：`type(scope): 描述`，如 `feat(server): 用户注册接口`
- 小步提交，一个接口/一个中间件一次 commit

## 常见问题

**Q: 启动报 `数据库连接失败`？**
先确认 MySQL 已启动、已执行 `CREATE DATABASE`、DSN 里的用户名密码正确。

**Q: 接口返回 `{"code":10002}`？**
未携带或携带了过期的 token。先调 `/api/auth/login` 拿 token，再在请求头加 `Authorization: Bearer <token>`。

**Q: 改了 model 但表结构没变化？**
`AutoMigrate` 只会新增列，不会修改/删除已有列。开发期可直接删表让程序重建，或手动 `ALTER TABLE`。
