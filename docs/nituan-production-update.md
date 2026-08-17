# 泥团中转站生产更新手册

本手册只适用于正在跑的生产机：**站点 `api.nituan.cc`，目录 `/www/new-api`，compose 项目名 `new-api`**。

通用原理（空机器、别的 VPS）见 [`github-deploy.md`](./github-deploy.md)。**不要**把那边的 `/opt/nituan-api` 当成这台机的路径，也不要再 clone 一份新目录来发版。

仓库：fork [`ojeloqeb100-rgb/nituan-api`](https://github.com/ojeloqeb100-rgb/nituan-api)。功能分支 `feat/video-per-second-billing`，PR：<https://github.com/ojeloqeb100-rgb/nituan-api/pull/1>。

---

## 1. 原则：只更新代码，不碰客户数据

发版只做两件事：把 fork 的代码拉到 **同一个目录**，再用 **同一套数据卷** 重建应用容器。

客户、余额、令牌、渠道、模型定价全部在 Postgres 里，不在空的 `./data` 目录。只要同时满足下面几条，这些数据还在：

- 继续在 **`/www/new-api`** 操作，不换目录、不新建 clone
- compose 项目名保持 **`new-api`**（Postgres 实际卷名是 `new-api_pg_data`）
- **`/data` 左边仍是 `/www/new-api/data`**（`./data:/data`）
- Postgres 仍用命名卷 **`pg_data` → `new-api_pg_data`**
- **不改 `SQL_DSN`**（继续指向容器 `postgres` 里的 `new-api` 库）
- 不执行 `docker compose down -v`、`docker volume rm`、不删 `data/`、不清空库

`docker compose up -d --build` 只换镜像和容器，bind mount / named volume 原样保留。GitHub 工作流也 **不会** `git clean`、`git reset --hard`、`compose down` / `down -v`。

本次功能把成片路径写在任务表已有 JSON 字段里，**不新增、不删除表/列**。GORM `AutoMigrate` 只补缺，不会 drop。

---

## 2. 生产现状（以现场为准）

| 项 | 值 |
| --- | --- |
| 站点 | `api.nituan.cc`（Nginx 反代 `http://127.0.0.1:3000`） |
| SSH | 用户 `root` |
| 目录 / `DEPLOY_PATH` | **`/www/new-api`** |
| compose 项目名 | `new-api` |
| 应用容器 | `new-api`，当前镜像 `calciumion/new-api:latest`（Docker Hub 官方镜像，**不含**本 fork 的按秒计费） |
| 应用端口 | 容器 `3000` 映射到宿主机 `0.0.0.0:3000`（反代走本机 3000，不要随便改成新端口） |
| `/data` | 宿主机 **`/www/new-api/data` → `/data`**（bind）。目录是空的，没有 SQLite `one-api.db` |
| Postgres | 容器 `postgres`，库名 `new-api`，用户 `root` |
| 数据卷 | 命名卷 **`new-api_pg_data`** → `/var/lib/postgresql/data`（客户/余额/模型都在这里） |
| `.env` | **没有**。`SQL_DSN` 来自仓库里的 `docker-compose.yml`（PostgreSQL） |
| `docker-compose.override.yml` | 发版前应已放在服务器上（git 忽略，不进仓库）。只加 `build:`，**不改 volumes** |
| git `origin` | 必须是本 fork `ojeloqeb100-rgb/nituan-api`（官方仓可留作 `upstream`） |

客户数据位置：

- **用户 / 余额**：Postgres 表 `users`（`quota` / `used_quota`）
- **令牌 / 渠道 / 模型定价**：同一库的 `tokens`、`channels`、`options`（如 `ModelRatio`、`ModelPrice`、`billing_setting.*`）
- **`/www/new-api/data` 为空**，删它不会直接删库；但发版仍必须继续挂这一条左边路径，改了会挂到空目录，看起来像丢数据

---

## 3. 合并 PR 前检查清单

按顺序勾。缺任何一项都不要点 Merge。

1. **已做 Postgres `pg_dump`**，文件不在 `/www/new-api` 里，且大小非 0。目录：`/root/backups/`。
2. 已记下（备份前后应一致）：
   - 目录 `/www/new-api`
   - 项目名 `new-api`
   - `/data` 左边 `/www/new-api/data`
   - Postgres 卷名 `new-api_pg_data`
3. GitHub Secrets 已填写，且 **`DEPLOY_PATH=/www/new-api`**（见第 5 节）。
4. 服务器 `origin` 已指向 fork；存在带 `build:` 的 `docker-compose.override.yml`；**tracked 文件没有本地改动**。
5. PR 在 GitHub 上显示 **可以 Merge**（无 conflict）。若提示冲突，先更新分支，**不要硬点**。
6. 确认不会执行第 8 节里的禁止命令。

合并 PR = 向 fork 的 `main` 推送 = 触发工作流 **Deploy to VPS** = 第一次用 GitHub 往这台机发版。工作流文件 `.github/workflows/deploy.yml` 目前只在功能分支上，**合进 `main` 的那一次 push 会带上它并触发**。在此之前 Actions 列表里可能看不到这个 workflow，这是正常的。

---

## 4. 如何备份（合并前必须）

在服务器上执行。备份放在 **`/root/backups/`**，不要只拷在 `/www/new-api` 里。

```bash
mkdir -p /root/backups
chmod 700 /root/backups

docker exec postgres pg_dump -U root new-api > /root/backups/new-api-$(date +%Y%m%d-%H%M%S).sql

chmod 600 /root/backups/new-api-*.sql
ls -lh /root/backups
```

确认：文件大小不是 0；用 `head -n 5` 应能看到 PostgreSQL dump 头（不要把 dump 全文拷进聊天或 git）。

可选核对（只读，不要 UPDATE）：

```bash
docker exec postgres psql -U root -d new-api -c 'SELECT count(*) FROM users;'
docker inspect new-api --format '{{range .Mounts}}{{.Type}} {{.Source}} -> {{.Destination}}{{println}}{{end}}'
docker inspect postgres --format '{{range .Mounts}}{{.Type}} {{.Name}} {{.Source}} -> {{.Destination}}{{println}}{{end}}'
```

没有这份备份，不要点 Merge。

---

## 5. 如何配 GitHub Secrets

仓库：**Settings → Secrets and variables → Actions → New repository secret**

也可以在已登录 `gh` 的电脑上：`gh secret set NAME -R ojeloqeb100-rgb/nituan-api`。

| Secret | 必填 | 这台机应填的值 |
| --- | --- | --- |
| `DEPLOY_HOST` | 是 | SSH 能连上的主机名或 IP（可用 `api.nituan.cc` 对应的那台）。**不要写进 git** |
| `DEPLOY_USER` | 是 | `root` |
| `DEPLOY_SSH_KEY` | 是 | 该用户的 **私钥全文**（含 `BEGIN` / `END` 行）。推荐 OpenSSH ED25519 |
| `DEPLOY_PATH` | 是 | **`/www/new-api`**。填别的路径（例如 `/opt/nituan-api`）会在新目录起一套空的 `*_pg_data`，看起来像丢库 |
| `DEPLOY_PORT` | 否 | 缺省 `22` |
| `DEPLOY_SSH_FINGERPRINT` | 否 | 主机公钥 SHA256 指纹，防中间人。服务器上：`ssh-keygen -l -E sha256 -f /etc/ssh/ssh_host_ed25519_key.pub`，Secret 里填 `SHA256:……` 那一段 |

不要把 IP、私钥、数据库密码写进仓库或本手册的可复制示例值。

私钥必须是 **GitHub Actions 用来 SSH 上这台机** 的那把，公钥已在服务器 `root` 的 `authorized_keys` 里。网页粘贴时注意：

- 要包含首尾 `-----BEGIN …-----` / `-----END …-----`
- 不要少行、不要把公钥当成私钥
- 建议 UTF-8、Unix 换行（LF）

---

## 6. 服务器一次性准备（origin + override）

在 **`/www/new-api`** 里做。不要 `compose down`，不要改正在跑的容器（准备阶段只改 git remote 和未跟踪的 override）。

### 6.1 把 origin 改成本 fork

工作流会拒绝从官方仓拉代码。官方仓可保留为 `upstream`：

```bash
cd /www/new-api
git remote -v

# 若 origin 仍是 QuantumNous/new-api：
git remote rename origin upstream
git remote add origin https://github.com/ojeloqeb100-rgb/nituan-api.git
git fetch origin

# 不要在这里 checkout 功能分支、不要 merge、不要 compose up
git status --porcelain --untracked-files=no
# 上面必须没有输出（tracked 文件不能有本地改动，否则工作流会拒绝）
```

fork 是公开仓库，服务器用 HTTPS fetch **不需要**再配一把 GitHub Deploy key。

### 6.2 添加 `docker-compose.override.yml`（只加构建，不换卷）

该文件已被 `.gitignore` 忽略，**不要 commit**。对照仓库 `docker-compose.prod.example.yml`，但 **data / postgres 路径必须与现在完全一致**。

不要从 example 里照抄 `127.0.0.1:3000:3000` 来「顺便改端口」——当前已是 `3000:3000`，Nginx 反代本机 3000；第一次发版不要额外改端口。

服务器上应类似：

```yaml
# Server-only overlay for /www/new-api. Do not commit.
# Only switches the app to a local image build. Do not change volume paths.
services:
  new-api:
    build:
      context: .
      dockerfile: Dockerfile
    image: nituan-api:local
    environment:
      - VIDEO_CACHE_DIR=/data/video-cache
    volumes:
      - ./data:/data
      - ./logs:/app/logs
```

**不要**在 override 里写 postgres 的新 volume、不要改 `SQL_DSN`、不要改 `/data` 或 `pg_data` 的左边路径。未出现在 override 里的 `postgres` 服务继续用官方 compose 的 `pg_data`（实际名 `new-api_pg_data`）。

写好后只做配置核对，**不要** `up`（第一次 `up --build` 交给 Merge 后的 Actions）：

```bash
cd /www/new-api
docker compose config | grep -E "source:|target:|image:|nituan-api|/data|pg_data"
```

应仍看到 `/www/new-api/data` 和项目卷 `new-api_pg_data`。目录名已经是 `new-api`，**不必**再写 `COMPOSE_PROJECT_NAME`。

### 6.3 大盘只加缓存（可选，现在不必做）

成片缓存在容器内 `/data/video-cache`。以后若要第二块盘，**只追加** bind，不要换掉 `./data:/data`：

```yaml
volumes:
  - ./data:/data
  - ./logs:/app/logs
  - /你的大盘挂载点/video-cache:/data/video-cache
```

---

## 7. 合并 PR 后如何验证用户 / 余额还在

1. GitHub → Actions → **Deploy to VPS**：日志里应有 `OK /data stays at /www/new-api/data`、`OK postgres volume new-api_pg_data`，然后是 `compose up -d --build`。**不能**出现 `down -v` / `git clean` / `volume rm`。
2. 第一次构建前端 + Go 镜像可能要十几分钟，机器内存建议够用（或已有 swap）。
3. 容器起来后立刻核对：

```bash
docker inspect new-api --format '{{range .Mounts}}{{.Source}} -> {{.Destination}}{{println}}{{end}}'
# /data 左边必须仍是 /www/new-api/data

docker inspect postgres --format '{{range .Mounts}}{{.Name}} {{.Source}} -> {{.Destination}}{{println}}{{end}}'
# 卷名必须仍是 new-api_pg_data

docker exec postgres psql -U root -d new-api -c 'SELECT count(*) FROM users;'
docker exec postgres psql -U root -d new-api -c 'SELECT id, username, quota, used_quota FROM users ORDER BY id;'
```

后台登录：用户、令牌、余额、渠道、历史任务还在。若用户表是空的或只剩默认 `root`，**立刻停发版、不要再 `up`**，用 `/root/backups/` 里的 SQL dump 恢复，并检查是不是挂到了新路径/新 volume。

站点：`https://api.nituan.cc` 应仍由 Nginx 反代到本机 3000。

---

## 8. 禁止事项

不要在这台机上执行：

- `docker compose down -v`
- `docker volume rm`（尤其是 `new-api_pg_data`）
- 删除 `/www/new-api/data` 或 Postgres 数据目录
- 修改 `SQL_DSN` 指到空库或别的主机
- 把 `./data:/data` 或 `pg_data` **左边路径**改成新目录
- 把仓库 clone 到 `/opt/nituan-api` 或其它目录再 `compose up`（会得到空的 `nituan-api_pg_data`）
- 把 `DEPLOY_PATH` 填成非 `/www/new-api` 的路径
- `git reset --hard`、`git clean -fdx`（会清掉 override）
- force push
- 把 SSH 私钥、数据库密码、完整 SQL dump 提交进 git 或发到聊天里

---

## 9. 必须由你在网页上点的卡点

下面这些我这边无法代替你点（或点了有风险），需要你操作 / 确认：

1. **Merge PR**：打开 <https://github.com/ojeloqeb100-rgb/nituan-api/pull/1>。仅当第 3 节清单都勾完、且 GitHub 显示 **能 Merge（无冲突）** 时再点。点 Merge 就会触发第一次部署。
2. **PR 若提示冲突**：不要 Merge。需要先把 fork 的 `main` 合进 `feat/video-per-second-billing` 并解决冲突后再推送。冲突文件包括计费相关的 `model/user.go`、`service/task_billing.go` 等，不要用网页「随便选一边」带过。
3. **Actions 权限**：仓库 Settings → Actions → General 应允许运行 workflows。若 Merge 后没有出现 **Deploy to VPS**，到这里确认 Actions 已启用，且没有拦住 `appleboy/ssh-action`。
4. **第一次 SSH 失败**：看 Deploy 日志。常见原因：
   - `DEPLOY_SSH_KEY` 少了 BEGIN/END、粘成了公钥、或换行损坏 → 在 Secrets 里重填私钥全文
   - host key / fingerprint 不匹配 → 核对本机 `ssh-keygen -l -E sha256 -f /etc/ssh/ssh_host_ed25519_key.pub` 后更新 `DEPLOY_SSH_FINGERPRINT`，或先清空该 Secret 重试（仅当你确认连的就是这台机）
5. **`DEPLOY_PATH` 填错不要靠重跑碰运气**：必须是 `/www/new-api`。填错时工作流应拒绝；若你改过 Secret，改回来后再 Run workflow。

---

## 10. 建议操作顺序（第一次发版）

1. 服务器 `pg_dump` → `/root/backups/`（第 4 节）
2. 服务器改 `origin` + 写 override（第 6 节）
3. GitHub 填写 Secrets，`DEPLOY_PATH=/www/new-api`（第 5 节）
4. 确认 PR 无冲突
5. **你在 GitHub 点 Merge PR**（不要在服务器上手动 `compose up --build`，除非 Actions 失败需要排障）
6. 看 Actions 日志，再按第 7 节验证用户数和余额

之后日常发版：向 fork 的 `main` 推送（或再开 PR 合进 `main`）即可。要发某个功能分支：Actions → Deploy to VPS → Run workflow，`ref` 填已经 push 到 fork 的分支名（该 workflow 出现在 `main` 上之后才可用手动触发）。
