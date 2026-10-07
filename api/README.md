# 接口清单

Base URL 为 `/api`。需要登录的接口在请求头携带 `Authorization: Bearer <token>`。
所有响应统一为 `{code, msg, data}` 信封，`code = 0` 表示成功。

权限一栏：`公开` / `登录` / `失物招领管理员+`（finder_admin 与 sys_admin） / `系统管理员`（sys_admin）。

## 上传

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| POST | `/upload` | 登录 | 单张图片，`multipart/form-data`，字段名 `file`。jpg/jpeg/png/webp，≤ 5MB |

失败一律返回错误码 `8`：不区分「格式不支持」与「超过大小限制」——
对上传方而言两者的处理动作相同（换一张）。

## 用户与鉴权

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| POST | `/auth/register` | 公开 | 注册。落 `role = user`；注册接口不接受角色入参 |
| POST | `/auth/login` | 公开 | 登录，凭证在 `data.token`。密码错 `6`，账号禁用 `10` |
| POST | `/auth/logout` | 登录 | 退出。JWT 无状态，服务端无可销毁的会话 |
| GET | `/auth/profile` | 登录 | 当前用户档案 |
| PUT | `/auth/profile` | 登录 | 改昵称与联系方式，`?user_id=` 须与登录身份一致 |
| PUT | `/auth/password` | 登录 | 改密码。旧密码错 `1`，新旧相同 `11` |
| GET | `/admin/users` | 系统管理员 | 用户列表，支持 `keyword` / `role` / 分页 |
| PUT | `/admin/users/{user_id}/role` | 系统管理员 | 调整角色（`finder_admin` ↔ `user`）。降级自己 `7`，改其他 sys_admin `3` |

## 物品

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| GET | `/items/list/{type}/` | 公开 | 信息流。仅 `item_status = 1`，按 `happen_time` 倒序 |
| GET | `/items/{item_id}` | 公开（可选鉴权） | 详情。待审核/已驳回仅本人与管理员可见，其余访问者返回 `4` |
| POST | `/items/{type}` | 登录 | 发布，落 `item_status = 0` |
| PUT | `/items/{item_id}` | 登录 | 修改。**全段更新**，改完状态重置为 `0` 并清空驳回理由 |
| DELETE | `/items/{item_id}` | 登录 | 删除。本人或任一后台角色 |
| GET | `/my/items` | 登录 | 我的发布，含全部状态与驳回理由 |
| POST | `/items/{item_id}/close` | 登录 | 关闭认领通道（`item_status = 3`）。本人或任一后台角色 |

`{type}` 为 `lost`（寻物启事）或 `found`（失物招领）。
信息流的可选筛选：`category` / `keyword`（匹配名称与描述）/ `location` / `start_time` / `end_time` / `page` / `page_size`。

## 认领申请

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| POST | `/claims?item_id=` | 登录 | 提交申请。仅 `found` 且已发布的帖子可申请 |
| GET | `/my/claims/` | 登录 | 我的认领申请 |
| GET | `/claims/{claim_id}` | 登录 | 详情，内联对应物品。本人或后台角色 |
| PUT | `/claims/{claim_id}` | 登录 | 改理由与联系方式，仅申请人本人 |
| DELETE | `/claims/{claim_id}/` | 登录 | 取消/删除。本人，或 sys_admin 删任意 |
| GET | `/admin/claims` | 失物招领管理员+ | 申请列表，支持 `claim_status` |
| POST | `/admin/claims/{claim_id}/{option}` | 失物招领管理员+ | 审批。`option` 为 `approve` / `reject` |

提交申请的错误码按此优先级判定：
物品不存在 `4` → 不可认领 `7` → 申请自己的帖子 `7` → 重复申请 `9` → 理由为空 `1`。

## 发布审核

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| GET | `/admin/items/pending/{type}/` | 失物招领管理员+ | 待审核队列，仅 `item_status = 0` |
| POST | `/admin/items/{item_id}/{option}` | 失物招领管理员+ | 审核。通过置 `1`，驳回置 `2` 并记录理由。非待审核状态返回 `7` |
| GET | `/admin/items/` | 系统管理员 | 全校总览，可按 `type` / `status` / `category` / `keyword` 过滤 |
| PUT | `/admin/items/{item_id}/` | 系统管理员 | 仅支持把状态置为 `3`（线下确认后关闭） |

驳回理由走请求体 `{"reject_reason": "..."}`；通过时不带请求体。

## 公告

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| GET | `/announcements/` | 公开 | 公告列表，仅 `announcement_status = 0` |
| GET | `/announcements/{announcement_id}` | 公开 | 公告详情 |
| GET | `/admin/announcements/` | 系统管理员 | 全部公告（含已下线） |
| POST | `/admin/announcements/` | 系统管理员 | 发布公告，标题与正文必填 |
| PUT | `/admin/announcements/{announcement_id}` | 系统管理员 | 修改。字段可选，**上下线也走这里**（体里带 `announcement_status`） |
| DELETE | `/admin/announcements/{announcement_id}` | 系统管理员 | 删除 |

## 错误码表

一个 code 唯一对应一条 msg，也唯一对应一个用于承载它的 HTTP 状态码。

| code | msg | HTTP | 典型场景 |
|---|---|---|---|
| 0 | success | 200 | 成功 |
| 1 | 参数错误 | 400 | 请求参数缺失或格式不合法 |
| 2 | 未登录或登录已过期 | 401 | token 缺失/非法/过期，前端应跳登录页 |
| 3 | 无权限操作 | 403 | 角色不满足接口要求 |
| 4 | 资源不存在 | 404 | 帖子/用户/申请不存在；也包括「待审核帖对无权限访问者」 |
| 5 | 用户名已存在 | 409 | 注册冲突 |
| 6 | 用户名或密码错误 | 401 | 登录失败 |
| 7 | 当前状态不允许该操作 | 405 | 重复审核、审核非待审核帖、申请自己的帖子、降级自己 |
| 8 | 文件上传失败 | 406 | 格式不支持或超过大小限制 |
| 9 | 请勿重复提交 | 409 | 重复认领同一帖子 |
| 10 | 账号已被禁用 | 423 | 登录时被禁用的账号；既有凭证也会被拦下 |
| 11 | 新密码不能与旧密码相同 | 400 | 改密码 |
| 114 | 服务器内部错误 | 500 | 未预期异常 |

未命中的路径与错误的方法同样返回信封 + code `4`，不会漏出框架自带的纯文本 404。

## 枚举取值

- 角色 `role`：`user` / `finder_admin` / `sys_admin`
- 物品大类 `type`：`found`（失物招领）/ `lost`（寻物启事）
- 物品状态 `item_status`：`0` 待审核 / `1` 已发布 / `2` 已驳回 / `3` 已关闭
- 认领状态 `claim_status`：`0` 待审批 / `1` 已通过 / `2` 已驳回 / `3` 已取消
- 公告状态 `announcement_status`：`0` 公开 / `1` 已下线
- 分类 `category`：卡证 / 数码电子 / 挂饰饰品 / 箱包 / 钥匙 / 书籍文具 / 衣物 / 现金钱包 / 其他
