# 精弘校园失物招领系统 · 后端

Go 技术栈实现的后端服务，与《精弘大作业（失物招领系统）》接口文档逐条对齐，
可直接对接 `_Frontendjh_副本` 里那套 Vue 3 前端。

## 技术栈

| 关注点 | 选型 | 为什么 |
|---|---|---|
| HTTP 框架 | Gin v1.12 | 中间件链清晰，「业务异常统一响应中间件」正落在它的 `c.Errors` 机制上 |
| ORM | GORM v1.31 | `AutoMigrate` 让表结构与 model 定义始终一致，不必维护两份真相 |
| 数据库 | SQLite（默认）/ MySQL | 默认走纯 Go 驱动，评分环境 `go run` 直接起，不需要装库；切 MySQL 只改配置 |
| 鉴权 | JWT / HS256（golang-jwt/v5） | 无状态、自包含，天然适配前后端分离与前端「登录状态持久化」 |
| 口令 | bcrypt | 库中只存摘要，明文永不落库 |
| 配置 | yaml.v3 + 环境变量覆盖 | 配置项是封闭的，用不上 viper 那类重型配置库 |
| 日志 | 标准库 `log/slog` | 结构化、可分级，零依赖 |
| 参数校验 | 手写在 service 层 | 文档对错误码的**优先级**有要求（如「认领已关闭的帖子」要回 7 而不是 1），声明式 tag 表达不了先后关系 |

## 快速开始

```bash
cd _Backendjh_副本

# 默认监听 :8080 —— 与前端 vite 代理的默认目标一致
go run ./cmd/server
```

对接前端：

```bash
cd _Frontendjh_副本
npm install
npm run dev                        # 默认就指向 http://localhost:8080

# 后端若改在别的端口（例如本机 8080 被占用）：
VITE_API_TARGET=http://127.0.0.1:8095 npm run dev
```

首次启动会自动建表并写入演示数据。**演示数据是幂等的**：库里已有用户则跳过，
因此反复重启既不会累积重复记录，也不会覆盖你在界面上改过的内容。

## 演示账号

| 用户名 | 密码 | 角色 |
|---|---|---|
| `admin` | `admin123` | 系统管理员 |
| `admin2` | `admin123` | 系统管理员（副号，用于验证「不允许修改其他系统管理员的角色」） |
| `finder001` | `abc123` | 失物招领管理员 |
| `student001` | `abc123` | 普通用户 |
| `student002` | `abc123` | 普通用户 |

## 目录结构

```
cmd/server/            服务入口：读配置 → 连库迁移 → 起 HTTP → 优雅退出
config/config.yaml     配置（可用环境变量覆盖）
internal/
├── config/            配置加载与默认值
├── database/          连接与建表（SQLite / MySQL 的差异隔离在此）
├── model/             实体、状态常量与领域判断（IsVisibleTo / CanBeManagedBy）
│   └── seed.go        演示数据
├── dto/               请求体、响应体与查询条件（接口字段名与库字段名的唯一映射点）
├── repository/        数据访问层，唯一与 GORM 打交道的地方
├── service/           业务规则：状态机、权限判定、错误码选择
├── handler/           HTTP 适配层：解析 → 调 service → 写响应
├── middleware/        统一响应 / 鉴权 / 角色 / CORS / 访问日志
├── router/            组合根与路由表
└── token/             JWT 签发与校验
pkg/                   与业务无关的通用件
├── apperr/            业务异常类型
├── errcode/           错误码表（code ↔ msg ↔ HTTP 状态码）
├── response/          统一响应信封
├── logger/            日志出口
└── timeutil/          时间串格式
api/README.md          接口清单
docs/后端说明.md        设计决策与契约说明
```

数据流只有一条：
**handler → service → repository → 数据库**。handler 不做业务判断，
service 不碰 gin 与 gorm，repository 不返回裸 `*gorm.DB`。

## 配置

优先级：`config/config.yaml` 的默认值 → 环境变量覆盖。
部署时改端口或密钥不必动文件。

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `SERVER_PORT` | `8080` | 监听端口 |
| `SERVER_MODE` | `release` | `debug` / `release` |
| `DB_DRIVER` | `sqlite` | `sqlite` / `mysql` |
| `DB_DSN` | `data/lostfound.db` | sqlite 为文件路径；mysql 形如 `user:pass@tcp(127.0.0.1:3306)/lostfound?charset=utf8mb4&parseTime=True&loc=Local` |
| `DB_SEED` | `true` | 是否写入演示数据 |
| `JWT_SECRET` | 内置默认值 | **生产必须覆盖** |
| `JWT_EXPIRE_HOURS` | `72` | 凭证有效期 |
| `UPLOAD_DIR` | `data/uploads` | 图片落盘目录 |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `APP_CONFIG` | `config/config.yaml` | 配置文件路径 |

切到 MySQL 只需两步：

```bash
DB_DRIVER=mysql \
DB_DSN='root:pass@tcp(127.0.0.1:3306)/lostfound?charset=utf8mb4&parseTime=True&loc=Local' \
go run ./cmd/server
```

## 测试

```bash
go test ./...           # 16 个契约用例
go vet ./...
```

契约测试跑的是**真实路由 + 真实 SQLite + 真实种子数据**（不起桩），
逐条核对错误码、角色权限、物品与认领的状态机、分页口径与信封结构 ——
这些断言抄自接口文档与前端交付说明，因此它们同时是「前后端约定一致」的可执行证据。

## 接口

完整清单（33 个端点、权限、错误码）见 [`api/README.md`](api/README.md)，
设计取舍与契约说明见 [`docs/后端说明.md`](docs/后端说明.md)。
