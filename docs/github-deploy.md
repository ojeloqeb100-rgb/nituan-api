# 用 GitHub Actions 部署到自己的云服务器

这份说明只给 **本 fork**（`ojeloqeb100-rgb/nituan-api`）的个人 VPS 用，不是给 QuantumNous 官方仓库的通用发布流程。

## 结论（已有客户数据时先看这里）

**这样更新不会丢客户数据**，前提同时成立：

- 不执行 `docker compose down -v`、`docker volume rm`、不删除 `./data`、不清空/重建空库
- `docker compose up -d --build` **只换镜像和容器**，bind mount / named volume **原样保留**
- **不要改 volumes 左边的宿主机路径**（例如不要把 `./data:/data` 改成另一块盘或新目录；改了容器会挂到空目录，看起来像丢库）
- **不要换一个新目录再起一套 compose**（Postgres 的 `pg_data` 实际名字带目录前缀，换目录等于挂到一个新的空卷）
- 从 Docker Hub `calciumion/new-api` 改成本地 `Dockerfile` 构建：只要 **同一个 `./data:/data`（或同一外部数据库）** 且不 `down -v`，数据还在

本次功能改动把成片路径写在任务表已有的 JSON 字段里，**不新增、不删除数据库表/列**。启动时 GORM `AutoMigrate` 只补缺，**不会 drop**。

GitHub 工作流 **不会** `git clean`、`git reset --hard`、`compose down` / `down -v`，也 **不会** 删除 `.env`、`docker-compose.override.yml`、`data/`。

---

## 已有生产实例：第一次用 GitHub 发版前

「泥团中转站」如果已经在跑、里面有客户，按这个顺序做。**先备份，再改仓库/发版。**

### 1. 在服务器上备份（必须）

在**当前正在跑 compose 的目录**里做，备份拷到**另一块盘或本机**，不要只放在即将改动的目录里。

```bash
# 先确认你现在就在「正在跑的」那个目录
pwd
docker inspect new-api --format '{{index .Config.Labels "com.docker.compose.project.working_dir"}}'
docker inspect new-api --format '{{range .Mounts}}{{.Type}} {{.Source}} -> {{.Destination}}{{println}}{{end}}'
```

记下 `/data` 左边的宿主机路径，以及（若有）Postgres/MySQL 的 volume 名。后面 override **必须继续用同一条左边路径**。

```bash
ts=$(date +%Y%m%d%H%M)
dest="$HOME/nituan-backup-$ts"
mkdir -p "$dest"

# 应用目录里的 data（SQLite 时这里就是库；官方镜像 WORKDIR 是 /data，库文件一般是 data/one-api.db）
if [ -d data ]; then
  cp -a data "$dest/data"
fi

# 机器专属配置（git 不跟踪，丢了会连不上库）
cp -a .env "$dest/" 2>/dev/null || true
cp -a docker-compose.override.yml "$dest/" 2>/dev/null || true

# 官方 compose 默认走容器内 Postgres 时，再 dump 一份
if docker inspect postgres >/dev/null 2>&1; then
  docker exec postgres pg_dump -U root new-api > "$dest/new-api.sql"
fi

# 若实际是 MySQL：
# docker exec mysql mysqldump -u root -p --databases new-api > "$dest/new-api.sql"

# 若 SQL_DSN 指向外部数据库：在那台库上做 dump，不要只拷 data 目录
```

没有这份备份，不要点 Actions 发版。

### 2. 确认数据现在在哪

官方 `docker-compose.yml` 里有两处，**都可能是客户数据**：

| 位置 | 什么时候有客户数据 |
| --- | --- |
| 宿主机 `./data` → 容器 `/data` | 未设 `SQL_DSN` 时用 SQLite（`/data/one-api.db`）。即使走 Postgres，这里也可能有文件/缓存 |
| named volume `pg_data`（实际名 `{目录名}_pg_data`） | 官方 compose **默认** `SQL_DSN` 指向容器 `postgres` |
| 外部 MySQL / Postgres | `SQL_DSN` 指向你自己的库 |

用现在的环境变量确认，不要猜：

```bash
docker exec new-api sh -c 'printf "SQL_DSN=%s\n" "$SQL_DSN"'
```

- 空或 `local`：客户在 `./data`（SQLite）
- `postgresql://...@postgres...`：客户在 `pg_data`
- 指向别的主机/端口：客户在那台外部库

### 3. 从官方镜像换成本地构建（数据怎么还在）

现在生产若跑的是 Docker Hub `calciumion/new-api`，本 fork 的按秒计费/成片缓存 **不在那个镜像里**，所以要在 **同一目录** 加 `docker-compose.override.yml` 从 `Dockerfile` 构建。

`docker compose up -d --build` 会：

- 用新镜像 **重建 `new-api` 容器**
- **继续挂** 原来的 `./data:/data`
- **继续用** 原来的 `pg_data`（只要还在同一个 compose 项目/目录）
- **不会** 删 volume、不会清空库

会丢数据的操作（工作流已禁止，人也不要手敲）：

- `docker compose down -v`
- `docker volume rm …`
- `rm -rf data` / 重建空库
- 把 `./data:/data` 左边改成新路径
- 把仓库 clone 到新目录再 `compose up`（新的空 `xxx_pg_data`）
- 改 `SQL_DSN` 指到一个空库

### 4. 两块盘：只加缓存，不换数据库目录

成片缓存在容器内 `/data/video-cache`。第二块大盘 **只在 override 里追加** bind，**不要换掉** 原来的 `./data:/data`：

```yaml
volumes:
  - ./data:/data
  - ./logs:/app/logs
  - /你的大盘挂载点/video-cache:/data/video-cache
```

不要写成「整盘替换 `/data`」或删掉 `./data:/data`。同一容器路径挂两次也不行；正确做法是 **保留原数据库目录，再多挂一层缓存目录**。

若你现在的 `/data` 左边已经不是 `./data`（自定义绝对路径），override 里 **继续写现在正在用的那条左边路径**，不要改回 `./data`。

### 5. 服务器 git 指到本 fork（仍用正在跑的目录）

`DEPLOY_PATH` 必须是 **当前 compose 已经在用的绝对路径**，不要新建一份 clone。

```bash
cd /当前正在跑的目录
git remote -v
# 若 origin 还是 QuantumNous/new-api，改成本 fork（工作流会拒绝拉错仓库）
git remote set-url origin https://github.com/ojeloqeb100-rgb/nituan-api.git
git fetch origin

# 还没有 overlay 时再拷；已有 override 不要覆盖掉里面的 volumes
test -f docker-compose.override.yml || cp docker-compose.prod.example.yml docker-compose.override.yml
```

编辑 override：加上 `build:`，**保留** 原来的 `./data:/data`（或你正在用的左边路径）。需要大盘时只加 video-cache 那一行。

若工作流提示 compose 项目名会变，在 `.env` 里写上当前项目名（不要编一个新的）：

```bash
docker inspect new-api --format '{{index .Config.Labels "com.docker.compose.project"}}'
# 把打印出来的值写入 .env：
# COMPOSE_PROJECT_NAME=上面打印的值
```

### 6. 合并 PR 发版

1. 备份已完成，volume 左边路径已核对。
2. GitHub 填好 Secrets（见下文）。`DEPLOY_PATH` = 正在跑的那个目录。
3. 把功能分支合进 fork 的 **`main`**，或在 Actions 里 **Run workflow**（可先发功能分支做一次）。
4. 看 **Deploy to VPS** 日志：应出现 `OK /data stays at …`，然后是 `compose up -d --build`，**不能**出现 `down -v` / `git clean`。

### 7. 如何验证客户还在

发版后立刻核对，和备份前的 inspect 对比：

```bash
docker inspect new-api --format '{{range .Mounts}}{{.Source}} -> {{.Destination}}{{println}}{{end}}'
# /data 左边必须和备份前相同

docker exec new-api sh -c 'ls -l /data; ls -l /data/one-api.db 2>/dev/null || true'
# 若走 Postgres：
docker exec postgres psql -U root -d new-api -c 'SELECT count(*) FROM users;'
```

后台登录：用户、令牌、余额、历史任务还在。若用户表是空的或只剩默认 `root`，**立刻停发版、不要再 `up`**，用备份的 `data/` 或 SQL dump 恢复，并检查是不是挂到了新路径/新 volume。

---

## 采用的链路

**GitHub Actions SSH 到服务器 → `git fetch` / `checkout` / 快进 `merge` → `docker compose up -d --build`**

原因：

- 官方发布的是 Docker Hub 镜像 `calciumion/new-api`，**不是 GHCR**，也不包含本 fork 的功能（例如按秒视频计费、成片缓存）。
- 在服务器上按仓库 `Dockerfile` 构建，比再搭一套 GHCR 登录 / 改 `IMAGE` / 服务器 `docker pull` 更少运维。
- 工作流 **不会** 执行 `docker compose down` 或 `down -v`，避免清掉 Postgres 数据卷和 `/data`。

发版触发：

- 向 fork 的 `main` 推送（包括 **合并 PR 到 main**）
- 或在 Actions 里手动 **Run workflow**，可填写要部署的分支 / tag

### GitHub 发版实际会做什么

- 检查 `origin` 是不是本 fork；检查被 git **跟踪** 的文件有没有本地改动（有则 **拒绝**，不 stash、不 clean）
- `git fetch`，再 `checkout` + **快进** `merge`（或检出 tag）。只更新已跟踪文件
- 若更新前存在 `.env` / `docker-compose.override.yml` / `data/`，更新后少了任何一个就 **拒绝**
- 已有 `new-api` 容器时：核对 compose 目录、项目名、`/data` 宿主机路径、Postgres/MySQL volume 名，有变化就 **拒绝**
- 最后只跑：`docker compose up -d --build`

### 不会做什么

- 不会 `git clean`（包括 `git clean -fdx`），因此未跟踪的 `.env`、`docker-compose.override.yml`、`data/` 不会被清掉
- 不会 `git reset --hard`
- 不会 `docker compose down`、`down -v`、`volume rm`
- 不会 `rm -rf data`、不会重建空库
- 不会改你服务器上已有 volume 的名字

---

## GitHub Secrets（只填值，不要写进仓库）

仓库：**Settings → Secrets and variables → Actions → New repository secret**

| Secret | 必填 | 含义 |
| --- | --- | --- |
| `DEPLOY_HOST` | 是 | SSH 主机名或 IP（用域名也可以；不要把真实值提交到 git） |
| `DEPLOY_USER` | 是 | SSH 登录用户（需能无密码使用 Docker） |
| `DEPLOY_SSH_KEY` | 是 | 该用户的 **私钥全文**（含 `BEGIN` / `END` 行）。推荐 ED25519 |
| `DEPLOY_PATH` | 是 | 服务器上**已经在跑**的仓库 **绝对路径**。已有生产必须填这个目录，不要填新 clone |
| `DEPLOY_PORT` | 否 | SSH 端口，缺省 `22` |
| `DEPLOY_SSH_FINGERPRINT` | 否 | 主机公钥 SHA256 指纹，用于校验，防中间人。获取：`ssh-keygen -l -f /etc/ssh/ssh_host_ed25519_key.pub` |

不要把 IP、私钥、数据库密码写进 workflow 或文档示例里的可复制值。

私钥公钥配对示例（在 **你的电脑** 上生成，不要在仓库里生成）：

```bash
ssh-keygen -t ed25519 -a 200 -C "github-actions-deploy" -f ./nituan-deploy -N ""
# 公钥追加到服务器 ~/.ssh/authorized_keys
# 私钥全文粘贴为 DEPLOY_SSH_KEY
```

## 服务器上先做一次（空机器 / 还没跑过）

已有生产请走上面「已有生产实例」节，不要按空机器再 clone 一份。

### 1. 安装 Docker

安装 Docker Engine 和 Compose 插件（`docker compose`），把 `DEPLOY_USER` 加入 `docker` 组后重新登录。构建前端 + Go 镜像较吃内存，建议 **4G RAM** 或加 swap。已有实例做 volume 校验需要服务器上有 `python3`。

### 2. Clone 本 fork

必须 clone **fork**，不要 clone `QuantumNous/new-api`，否则拉下来的不是你的代码：

```bash
git clone https://github.com/ojeloqeb100-rgb/nituan-api.git /opt/nituan-api
cd /opt/nituan-api
git checkout main   # 若 main 还没有合并，可先 checkout 功能分支
```

若仓库是私有的：在服务器上另配一把 **只读 Deploy key**（和 GitHub Actions 用的那把「GitHub → 服务器」密钥不是同一把），再用 SSH URL clone。

`DEPLOY_PATH` 填这个目录的绝对路径。

### 3. 生产 overlay 与 `.env`

不要直接改仓库里的 `docker-compose.yml`（`git pull` 会冲突或覆盖密码）。在服务器上：

```bash
cp docker-compose.prod.example.yml docker-compose.override.yml
# 按「磁盘」和「域名」编辑 override，不要把改过的 override 提交回 git
# 已有生产：volumes 左边必须与当前 docker inspect 一致
```

同目录创建 `.env`（已在 `.gitignore`）。上线前至少改掉 compose 默认密码，并设置 `SESSION_SECRET`。HTTPS 域名还需要：

```env
SESSION_SECRET=请换成足够长的随机串
SESSION_COOKIE_SECURE=true
SESSION_COOKIE_TRUSTED_URL=https://你的域名
TRUSTED_PROXIES=反代所在网卡的 CIDR
```

然后在 `docker-compose.override.yml` 里取消对应 `environment` 行的注释。说明见 `docs/authentication.md`。

首次启动（在填好 Secrets 并合并到 `main` 之前，可先手动跑通）：

```bash
cd /opt/nituan-api
docker compose up -d --build
```

### 4. 第二块盘与 `/data/video-cache`

官方 compose 是 `./data:/data`。应用默认把成片缓存在容器内 **`/data/video-cache`**（可用环境变量 `VIDEO_CACHE_DIR`）。

只在 **服务器上的** `docker-compose.override.yml` **追加** 缓存盘，例如：

```yaml
- ./data:/data
- /你的挂载点/video-cache:/data/video-cache
```

**不要**把 `./data:/data` 换成大盘路径。**不要**在 workflow 里写盘符；每台机器的挂载路径不同。

### 5. 域名反代到 3000

示例 overlay 把端口绑在 `127.0.0.1:3000`。用 Nginx / Caddy / 面板把域名 HTTPS 反代到 `http://127.0.0.1:3000`。

若你现在是用公网 `:3000` 直接访问、还没有反代，先搭好反代再改端口绑定，否则发版后会连不上（这不是丢库）。

防火墙放行 `22`、`80`、`443` 即可；不必把 `3000` 暴露到公网。

## 之后怎么发版

1. **已有生产**：先完成上面的备份和 volume 确认。
2. 在 GitHub 填好 Secrets，`DEPLOY_PATH` 指向正在跑的目录，该目录有带 `build:` 的 `docker-compose.override.yml`。
3. 向 fork 提 PR，合并进 **`main`** → Actions 工作流 **Deploy to VPS** 会 SSH 上去更新并 `--build`。
4. 需要发某个功能分支时：Actions → **Deploy to VPS** → **Run workflow**，`ref` 填分支名（该分支必须已经 push 到 fork）。

服务器上被 git 跟踪的文件若有本地修改，工作流会 **拒绝部署**，以免覆盖。机器专属配置只放 `docker-compose.override.yml` 和 `.env`。

## 文件

| 路径 | 作用 |
| --- | --- |
| `.github/workflows/deploy.yml` | SSH 部署（只更新代码/镜像，不删数据） |
| `docker-compose.prod.example.yml` | 从源码构建；保留 `./data:/data`，可选追加 video-cache |
| `docs/github-deploy.md` | 本文 |

官方 CI（`ci.yml`、`docker-build.yml` 等）未改。官方镜像仍走 Docker Hub `calciumion/new-api`，本 fork 不使用那条发布线。
