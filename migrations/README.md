# 数据库迁移

## 当前策略：AutoMigrate

启动时由 `internal/database.Migrate` 调用 GORM 的 `AutoMigrate`，按 `internal/model`
里的结构体定义建表或补齐缺失字段。

选它而不是手写 SQL 迁移，是因为本项目实体少（4 张表）而字段改动频繁：
AutoMigrate 保证表结构与 model 定义始终一致，省去维护两份真相、以及
「改了结构体忘了改迁移脚本」这类只在特定环境才暴露的偏差。

代价要说清楚：AutoMigrate **只加不减** —— 删字段、改类型、改约束这类破坏性变更
它不做，也不会报错。真要做这类变更，按下文的手写 DDL 处理。

## 表结构

| 表 | 实体 | 说明 |
|---|---|---|
| `users` | `model.User` | 用户账号，`username` 唯一索引；`password` 存 bcrypt 摘要 |
| `items` | `model.Item` | 失物/招领帖子，`type` / `item_status` / `poster_id` 建索引 |
| `claims` | `model.Claim` | 认领申请，`item_id` / `applicant_id` / `claim_status` 建索引 |
| `announcements` | `model.Announcement` | 公告 |

索引都建在过滤与排序真正用到的列上：信息流按 `type + item_status` 过滤、按
`happen_time` 排序；我的发布按 `poster_id` 过滤、按 `created_time` 排序。

## 从 SQLite 切到 MySQL

改环境变量即可，业务代码无需改动：

```bash
DB_DRIVER=mysql \
DB_DSN='root:pass@tcp(127.0.0.1:3306)/lostfound?charset=utf8mb4&parseTime=True&loc=Local' \
go run ./cmd/server
```

两处需要留意：

1. **建库**：MySQL 需要先建好库（`CREATE DATABASE lostfound`），AutoMigrate 只建表。
2. **时间字段**：本项目的时间统一以 `VARCHAR(20)` 存 `YYYY-MM-DD HH:mm:ss`，
   不随数据库变化。好处是字典序即时间序、跨数据库行为完全一致；
   代价是不能直接用数据库的日期函数，需要时用 `STR_TO_DATE`。

## 破坏性变更

AutoMigrate 不处理删列/改类型。这类变更的流程：

1. 写一份编号 DDL 放在本目录（如 `0001_drop_items_legacy_column.sql`）；
2. 在变更说明里写清**回滚**方式与受影响的数据量；
3. 手工在对应环境执行，并在提交信息里注明已执行。

生产环境建议把 `DB_SEED` 关掉（`DB_SEED=false`），演示数据只用于本地与评分环境。
