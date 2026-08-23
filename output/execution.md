# gb-527 真实执行与验收记录

## 基本信息

- 项目：`gb-527 satellite-contact-window-deconfliction`
- 验证时间：2026-08-22 03:25–04:03 CST
- 项目目录：`/Users/gaobo/repositories/gitlab/评审项目/0-1代码生成提示词/golang-改编提示词/2000档项目提示词-20260821-第二批/gb-527`
- Compose 项目名：`satellite-contact-window-deconfliction`
- 前端 / 后端 / PostgreSQL：`18527 / 19527 / 57527`
- runtime smoke：`20527`（GORM SQLite，仅用于启动冒烟）
- 实现提交：`6f213614e0ebe99f962a9257351e6223c2e2cc49`

## 构建、静态检查与规模

在项目根目录真实执行并通过：

| 命令 | 真实结果 |
| --- | --- |
| `go work sync` | 通过 |
| `go build ./backend/...` | 通过 |
| `go vet ./backend/...` | 通过 |
| `go test ./backend/...` | 通过；scheduler 与 conflict service 测试成功 |
| `npm --prefix frontend ci` | 通过；lockfile 可重复安装 |
| `npm --prefix frontend run typecheck` | 通过；严格 TypeScript 检查无错误 |
| `npm --prefix frontend run build` | 通过；Angular CLI/esbuild production build 成功，初始包 584.83 kB，五页均懒加载 |
| `project_scale.py .` | `Go 功能代码 3476 行 / 41 个 .go 文件`，处于 2500–4200 行、24–42 文件红线内且低于 5000 行 |
| `runtime_smoke.py .` | `ok=true`；`go run ./cmd/server` 在 `20527` 启动，`/healthz` HTTP 200 |
| `docker compose config --quiet` | 通过 |

测试真实覆盖：

- 半开时间区间、端点相接不冲突、转向缓冲扩展和多天线容量。
- `station_capacity / satellite_overlap / band_mismatch / duration_shortfall / slew_buffer` 检测与稳定 conflict key。
- 相同输入和权重的候选顺序稳定，锁定窗口从不进入移动列表。
- `detected -> proposed -> pending_review -> accepted | rejected` 合法状态流。
- 接受前窗口版本变化返回 `version_conflict`，失败事务仍可由 reviewer 拒绝。

首次 Docker 构建发现 workspace 未生成模块级 `backend/go.sum`，补充模块锁定文件并恢复 `go 1.22.0` 后解决。最终前端重建遇到一个失效 npm 依赖缓存层；废弃缓存执行完整 `npm ci` 后 Docker production build 通过，新镜像再次通过健康和 Browser 回归。

非阻断供应链说明：固定 Angular 17 的传递开发工具会打印 npm deprecated/audit 汇总；本机配置的 `registry.npmmirror.com` 不实现 npm audit API，单独执行 `npm audit --omit=dev` 返回 404，因此未声称完成依赖漏洞审计。要求中的安装、类型检查、生产构建和运行验收均已真实通过。

## Compose 启动与健康

执行：

```bash
docker compose up -d --build
docker compose ps
curl -fsS http://127.0.0.1:19527/healthz
curl -fsS http://127.0.0.1:19527/readyz
curl -fsS http://127.0.0.1:18527/api/healthz
```

最终镜像状态：

| 服务 | 宿主端口 | 状态 |
| --- | --- | --- |
| `postgres` | `57527` | `Up (healthy)` |
| `backend` | `19527` | `Up (healthy)` |
| `frontend` | `18527` | `Up (healthy)` |

后端 `/healthz` 返回 `status=ok`；`/readyz` 真实 ping PostgreSQL 并返回 `database=available`；Nginx `/api/healthz` 同路径反代返回 HTTP 200，SPA `/windows` 返回 HTTP 200。最终启动日志无迁移、seed、代理或运行阻断错误。

## API 冒烟

使用 Compose PostgreSQL 真实执行 25 项检查。创建了 `QA527` 地面站、`QA-SAT-527` 规划资产、两个重叠窗口和一个锁定窗口；检测生成容量、卫星重叠和时长不足等冲突，完成接受、导出、版本冲突与拒绝两条独立链路。

| # | 方法与路径 | 预期 | 实际 | 结论 |
| ---: | --- | ---: | ---: | --- |
| 1 | `GET /api/v1/stations`（无 token） | 401 | 401 `unauthorized` | 通过 |
| 2 | `POST /api/v1/auth/login`（scheduler） | 200 | 200 | 通过 |
| 3 | `POST /api/v1/auth/login`（reviewer） | 200 | 200 | 通过 |
| 4 | `POST /api/v1/stations`（reviewer） | 403 | 403 `forbidden` | 通过 |
| 5 | `POST /api/v1/stations`（QA527） | 201 | 201，站点 ID 4 | 通过 |
| 6 | `POST /api/v1/satellites`（QA-SAT-527） | 201 | 201，资产 ID 4 | 通过 |
| 7 | `POST /api/v1/windows`（窗口 A） | 201 candidate | 201，窗口 ID 5 | 通过 |
| 8 | `POST /api/v1/windows`（窗口 B） | 201 candidate | 201，窗口 ID 6 | 通过 |
| 9 | `POST /api/v1/windows/5/submit` | 200 submitted | 200，版本 2 | 通过 |
| 10 | `POST /api/v1/windows/6/submit` | 200 submitted | 200，版本 2 | 通过 |
| 11 | `POST /api/v1/windows`（锁定链） | 201 candidate | 201，窗口 ID 7 | 通过 |
| 12 | `POST /api/v1/windows/7/lock` | 200 locked | 200，版本 2 | 通过 |
| 13 | `PUT /api/v1/windows/7`（锁定后移动） | 409 | 409 `locked_window` | 通过 |
| 14 | `POST /api/v1/conflicts/detect` | 200 | 200，首次返回 4 个冲突组 | 通过 |
| 15 | `GET /api/v1/conflicts/2` | 200 | 200，含证据、权重与 3 个建议 | 通过 |
| 16 | `POST /api/v1/conflicts/2/submit` | 200 pending_review | 200，版本 3 | 通过 |
| 17 | `POST /api/v1/conflicts/2/review`（接受） | 200 accepted | 200，版本 4 | 通过 |
| 18 | `GET /api/v1/conflicts/2/export` | 200 | 200，离线 planning record 且明确无控制命令 | 通过 |
| 19 | `POST /api/v1/conflicts/1/submit` | 200 pending_review | 200，版本 3 | 通过 |
| 20 | `GET /api/v1/windows/1` | 200 | 200 | 通过 |
| 21 | `PUT /api/v1/windows/1` | 200 | 200，窗口版本递增 | 通过 |
| 22 | `POST /api/v1/conflicts/1/review`（旧窗口版本接受） | 409 | 409 `version_conflict` | 通过 |
| 23 | `POST /api/v1/conflicts/1/review`（拒绝） | 200 rejected | 200，版本 4 | 通过 |
| 24 | 再次拒绝 conflict 1 | 409 | 409 `invalid_state` | 通过 |
| 25 | `GET /api/v1/audit?page_size=100` | 200 | 200，当时返回 22 条全链事件 | 通过 |

接口链路覆盖全部四个核心实体、JWT、RBAC、严格提交兼容性、锁定保护、检测、稳定建议、人工复核、规划导出、事务窗口版本检查、非法重复状态和审计。访问日志仅包含路由模板、状态、耗时、客户端和 request ID，不记录 JWT、请求体或复核备注全文。

## 内置 Browser 验收

仅使用 Codex 内置 Browser；未使用外部 Chrome、Chrome 扩展、Computer Use 或独立 Playwright。

1. 使用 `scheduler / Scheduler#527` 登录，进入 `/windows`，真实显示站点分组时间线和窗口表。
2. scheduler 创建窗口 #8，选用 maintenance 状态 `DES-03` 与不兼容 X band；创建成功保留为候选，提交真实返回 409，页面显示 `only active station and satellite resources can be submitted or locked`，错误未被吞掉。
3. 对兼容的窗口 #3 提交成功，从 `candidate` 进入 `submitted`；列表和时间线刷新。
4. `/stations` 显示四个站点、五个天线通道、真实占用时间线；`/satellites` 消费同一批资产和窗口 API。
5. `/conflicts` 再次扫描，已存冲突从 4 组增加到 7 组，本次范围返回 5 个冲突；band mismatch #7 展示窗口事实、频段证据和三个稳定排序方案。
6. scheduler 将 conflict #7 提交为 `pending_review`；页面只显示等待 reviewer 操作。
7. 退出 scheduler 后以 `reviewer / Reviewer#527` 登录；站点、卫星和窗口页均无新建、编辑、提交或锁定按钮，直接访问 `/audit` 由 reviewer/admin 路由守卫控制。
8. reviewer 选择 `station-2-window-8 / Evaluate PAC-02`，填写复核备注并接受；conflict #7 进入 `accepted` v4，选择和备注在最终新镜像中保持。
9. `/audit` 最终加载 33 条事件，最新 `conflict.reviewed` 的 before 为 `pending_review` v3，after 为 `accepted` v4 并含 `selected_action_key`；parameters 含 expected version、action key 和备注长度。Resolution decisions 中已接受方案不可编辑。
10. 最终重建后从 `about:blank` 新开 `/conflicts?final=20260822` 和 `/audit` 复验，两个页面 `dev.logs()=[]`，无 console error 或失败网络请求；截图目视无文字遮挡、截断或不连贯重叠。

截图：

- `browser-scheduler-windows.png`：窗口时间线、锁定状态、严格提交 409。
- `browser-scheduler-stations.png`：站点容量表和真实占用时间线。
- `browser-scheduler-conflict-pending.png`：band mismatch 证据、三项建议与 pending review。
- `browser-reviewer-conflict-accepted.png`：reviewer 选择兼容站点建议并接受。
- `browser-reviewer-audit.png`：request-linked ledger、before/after、不可变选择和资产版本。

## 提示词覆盖结论

| 要求 | 实现与真实验证 |
| --- | --- |
| 四个实体逐层分文件 | GroundStation、SatelliteAsset、ContactWindow、ConflictResolution 均独立 model/DTO/repository/service/handler/router；页面消费真实 API |
| 五个页面 | `/stations`、`/satellites`、`/windows`、`/conflicts`、`/audit` 均由内置 Browser 打开并验证核心内容 |
| sweep-line 与缓冲 | 原始容量与扩展 buffer 分开检测；半开区间、端点和多天线有单测 |
| 五类冲突 | station capacity、satellite overlap、band mismatch、duration shortfall、slew buffer 均在共享枚举和算法中实现 |
| 稳定可解释建议 | 保留高优先级、同源备选、兼容站点、人工处理；评分拆解优先级损失、距离、时长和余量，并以 action key 稳定破同分 |
| 锁定与人工边界 | 锁定窗口不进入移动列表；实测锁定后更新 409；算法从不写回窗口，接受只保存人工选择 |
| 事务状态流 | detected 内部落库后 proposed；submit 与 review 条件更新；接受在事务内行锁并核对全部窗口版本，实测 409 |
| JWT/RBAC | scheduler/reviewer/admin 后端权限、前端守卫与按钮显隐一致，实测 401/403 和 reviewer 无维护操作 |
| 审计与错误 | 参数、权重、建议、选择、request ID、前后摘要均真实显示；409 在页面可见；访问日志不含敏感正文 |
| 共享前端能力 | WindowStatusBadge、ResourceTimeline、ResolutionComparePanel、useAuth、useConflictDetection 均被多个页面真实复用 |
| README 与边界 | Docker 启动优先，含账号、栈、变量、API、枚举全位置、算法、安全边界、开发、排错和 License |

项目严格保持离线规划边界，不连接天线、射频设备或任务控制系统，不发送遥控指令，也不包含预约平台、工单、库存、订单、计费或通用日历能力。

## 停服与隔离

验收后执行：

```bash
docker compose down -v --remove-orphans
docker compose ps
```

本项目三个容器、命名卷和默认网络均已删除；`docker compose ps` 为空。仅清理 `satellite-contact-window-deconfliction` 资源，未执行 `docker system prune`，未停止或删除其他项目资源。
