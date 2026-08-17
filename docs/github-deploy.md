# 用 GitHub Actions 部署到自己的云服务器

这份说明只给 **本 fork**（`ojeloqeb100-rgb/nituan-api`）的个人 VPS 用，不是给 QuantumNous 官方仓库的通用发布流程。

## 采用的链路

**GitHub Actions SSH 到服务器 → `git fetch` / `checkout` → `docker compose up -d --build`**

原因：

- 官方发布的是 Docker Hub 镜像 `calciumion/new-api`，**不是 GHCR**，也不包含本 fork 的功能（例如按秒视频计费、成片缓存）。
- 在服务器上按仓库 `Dockerfile` 构建，比再搭一套 GHCR 登录 / 改 `IMAGE` / 服务器 `docker pull` 更少运维。
- 工作流 **不会** 执行 `docker compose down -v`，避免清掉 Postgres 数据卷和 `/data`。

发版触发：

- 向 fork 的 `main` 推送（包括 **合并 PR 到 main**）
- 或在 Actions 里手动 **Run workflow**，可填写要部署的分支 / tag

## GitHub Secrets（只填值，不要写进仓库）

仓库：**Settings → Secrets and variables → Actions → New repository secret**

| Secret | 必填 | 含义 |
| --- | --- | --- |
| `DEPLOY_HOST` | 是 | SSH 主机名或 IP（用域名也可以；不要把真实值提交到 git） |
| `DEPLOY_USER` | 是 | SSH 登录用户（需能无密码使用 Docker） |
| `DEPLOY_SSH_KEY` | 是 | 该用户的 **私钥全文**（含 `BEGIN` / `END` 行）。推荐 ED25519 |
| `DEPLOY_PATH` | 是 | 服务器上仓库的 **绝对路径**，例如 `/opt/nituan-api` |
| `DEPLOY_PORT` | 否 | SSH 端口，缺省 `22` |
| `DEPLOY_SSH_FINGERPRINT` | 否 | 主机公钥 SHA256 指纹，用于校验，防中间人。获取：`ssh-keygen -l -f /etc/ssh/ssh_host_ed25519_key.pub` |

不要把 IP、私钥、数据库密码写进 workflow 或文档示例里的可复制值。

私钥公钥配对示例（在 **你的电脑** 上生成，不要在仓库里生成）：

```bash
ssh-keygen -t ed25519 -a 200 -C "github-actions-deploy" -f ./nituan-deploy -N ""
# 公钥追加到服务器 ~/.ssh/authorized_keys
# 私钥全文粘贴为 DEPLOY_SSH_KEY
```

## 服务器上先做一次

### 1. 安装 Docker

安装 Docker Engine 和 Compose 插件（`docker compose`），把 `DEPLOY_USER` 加入 `docker` 组后重新登录。构建前端 + Go 镜像较吃内存，建议 **4G RAM** 或加 swap。

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
# 按下面「磁盘」和「域名」编辑 override，不要把改过的 override 提交回 git
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

把大盘挂到宿主机某个目录后，在 **服务器上的** `docker-compose.override.yml` 里 bind，例如：

- 整个 `/data` 都在大盘：`/你的挂载点:/data`（同时去掉 `./data:/data`，同一容器路径不能挂两次）
- 只把缓存放到大盘：保留 `./data:/data`，再加 `/你的挂载点/video-cache:/data/video-cache`

**不要**在 workflow 里写盘符；每台机器的挂载路径不同。

### 5. 域名反代到 3000

示例 overlay 把端口绑在 `127.0.0.1:3000`。用 Nginx / Caddy / 面板把域名 HTTPS 反代到 `http://127.0.0.1:3000`。

防火墙放行 `22`、`80`、`443` 即可；不必把 `3000` 暴露到公网。

## 之后怎么发版

1. 在 GitHub 填好上一节 Secrets，服务器已 clone 且存在带 `build:` 的 `docker-compose.override.yml`。
2. 向 fork 提 PR，合并进 **`main`** → Actions 工作流 **Deploy to VPS** 会 SSH 上去更新并 `--build`。
3. 需要发某个功能分支时：Actions → **Deploy to VPS** → **Run workflow**，`ref` 填分支名（该分支必须已经 push 到 fork）。

服务器上被 git 跟踪的文件若有本地修改，工作流会 **拒绝部署**，以免覆盖。机器专属配置只放 `docker-compose.override.yml` 和 `.env`。

## 文件

| 路径 | 作用 |
| --- | --- |
| `.github/workflows/deploy.yml` | SSH 部署 |
| `docker-compose.prod.example.yml` | 从源码构建 + 第二块盘 bind 示例 |
| `docs/github-deploy.md` | 本文 |

官方 CI（`ci.yml`、`docker-build.yml` 等）未改。官方镜像仍走 Docker Hub `calciumion/new-api`，本 fork 不使用那条发布线。
