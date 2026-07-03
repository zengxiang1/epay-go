# EPay Go

一个基于 Go + Gin + PostgreSQL + Redis + Vue 3 的支付系统示例项目，提供管理后台、商户中心、统一下单、通道管理，以及订单 / 退款 / 结算流程。

## 技术栈

- 后端：Go、Gin、GORM、PostgreSQL、Redis
- 前端：Vue 3、Vite、Arco Design
- 部署：Docker Compose、Nginx / Caddy

## 目录说明

- `cmd/server`：服务启动入口
- `internal`：后端核心业务
- `web`：前端代码
- `deploy/nginx`：Nginx 配置示例
- `deploy/caddy`：Caddy 配置
- `docker-compose.yml`：默认部署
- `docker-compose.prod.caddy.yml`：Caddy 直接对外的生产部署示例

## 快速开始

### 1. 准备环境变量

```bash
cp .env.example .env
```

然后按需修改数据库、Redis、JWT、默认管理员和支付渠道配置。

### 2. 启动项目

```bash
docker compose up -d --build
```

默认包含以下服务：

- `postgres`
- `redis`
- `backend`
- `frontend`

默认端口：

- `80`：前端
- `8080`：后端
- `5432`：PostgreSQL
- `6379`：Redis

### 常用访问入口

部署完成后，可直接访问以下前端路径：

- **管理员登录**：`/admin/login`
- **商户注册**：`/merchant/register`
- **商户登录**：`/merchant/login`

## 环境变量

参考 `.env.example`。常用变量包括：

- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `REDIS_PASSWORD`
- `JWT_SECRET`
- `DEFAULT_ADMIN_USERNAME`
- `DEFAULT_ADMIN_PASSWORD`
- `SITE_ADDRESS`
- `ACME_EMAIL`

> ⚠️ 支付渠道（支付宝、微信、汇付天下）的 AppID / 商户号 / 密钥等**不在 `.env` 中配置**，而是登录管理后台后在「通道管理」里按通道填写、保存到数据库。`.env.example` 中残留的 `ALIPAY_*` / `WECHAT_*` 变量后端已不再读取，可忽略。

系统首次启动且数据库中没有管理员时，会使用 `DEFAULT_ADMIN_USERNAME` 和 `DEFAULT_ADMIN_PASSWORD` 初始化默认管理员。

## 部署

### Caddy 直接对外

适用于宿主机未占用 `80/443`：

```bash
docker compose -f docker-compose.prod.caddy.yml up -d --build
```

### 宿主机 Nginx 反向代理

适用于宿主机已有 Nginx：

```bash
docker compose up -d --build
```

然后由宿主机 Nginx 反代到容器端口，示例配置见：

- `deploy/nginx/host.prod.conf.example`

## 支付参数说明

已支持的支付通道（均在管理后台「通道管理」中配置密钥并启用）：

- **支付宝官方**（plugin: `alipay`）
- **微信官方**（plugin: `wechat`）
- **汇付天下 / 斗拱聚合支付**：拆为两个独立插件——`hf-wxpay`（汇付-微信，支持扫码 / JSAPI / H5）和 `hf-alipay`（汇付-支付宝，当前仅扫码）。客户端传 `type=wxpay` 会路由到 `wechat` 或 `hf-wxpay` 通道，传 `type=alipay` 会路由到 `alipay` 或 `hf-alipay` 通道；具体走官方还是汇付，由后台通道的启用状态与排序（`sort` 升序优先）决定，对商户透明。

支付通道和支付场景是分开的：

- `type` / `pay_type`：决定渠道，例如 `wxpay`、`alipay`
- `pay_method`：决定场景，例如 `native`、`scan`、`h5`、`jsapi`、`web`

### 关键规则

- `type=native` **不允许单独使用**，因为无法判断是微信还是支付宝
- 后端支持显式别名，并会自动归一化：
  - `WX_NATIVE`
  - `WX_JSAPI`
  - `WX_H5`
  - `ALIPAY_SCAN`
  - `ALIPAY_H5`
  - `ALIPAY_WEB`

### 推荐传法

- **微信 Native**
  - `type=WX_NATIVE`
  - 或 `type=wxpay&pay_method=native`

- **微信 JSAPI**
  - `type=WX_JSAPI`
  - 或 `type=wxpay&pay_method=jsapi`

- **支付宝扫码**
  - `type=ALIPAY_SCAN`
  - 或 `type=alipay&pay_method=scan`

- **支付宝 H5**
  - `type=ALIPAY_H5`
  - 或 `type=alipay&pay_method=h5`

- **支付宝网页支付**
  - `type=ALIPAY_WEB`
  - 或 `type=alipay&pay_method=web`

## 构建说明

如果在中国大陆网络环境构建，可以在 `.env` 中设置：

```env
GOPROXY=https://goproxy.cn,direct
NPM_REGISTRY=https://registry.npmmirror.com
```

- `GOPROXY`：加速后端 Go 依赖下载。
- `NPM_REGISTRY`：加速前端 npm 依赖下载，前端镜像默认已使用该阿里云镜像。

> 注意：`web/package-lock.json` 里的依赖下载地址（`resolved`）会被写死，若该文件在配了内网镜像（如腾讯云内网 `mirrors.tencentyun.com`）的机器上重新生成，会导致其他环境 `npm ci` 因地址不可达而失败。重新生成锁文件时请确保使用公网可达的镜像。


