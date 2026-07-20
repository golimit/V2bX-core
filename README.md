# V2bX

基于多内核的 V2board 节点服务端，修改自 XrayR。  
维护者：[golimit](https://github.com/golimit) · 上游：[wyx2685/V2bX](https://github.com/wyx2685/V2bX) ← [InazumaV/V2bX](https://github.com/InazumaV/V2bX)

支持协议：VMess / VLESS / Trojan / Shadowsocks / Hysteria1/2；多节点单实例、在线 IP / TCP 连接限制、端口与用户级限速、条件编译多内核。

## 内核版本（dev）

| 内核 | 本仓库锁定 | 上游最新稳定 | 状态 |
|------|------------|--------------|------|
| [Xray-core](https://github.com/XTLS/Xray-core) | `v1.260711.0` → [golimit/xray-core](https://github.com/golimit/xray-core) `@b17a88f9` | **v26.7.11** | 已对齐；相对 upstream/main 仅差 1 个 CI commit |
| [sing-box](https://github.com/SagerNet/sing-box) | `v1.13.14` → [golimit/sing-box_mod](https://github.com/golimit/sing-box_mod) `v1.13.14-mod.2` | **1.13.14** | 已对齐稳定版（未跟 1.14 alpha） |

以 `go.mod` 的 `require` / `replace` 为准。

## 功能矩阵

| 功能 | v2ray | trojan | shadowsocks | hysteria1/2 |
|------|-------|--------|-------------|-------------|
| 自动申请 / 续签 TLS | √ | √ | √ | √ |
| 在线人数 / IP / 连接数限制 | √ | √ | √ | √ |
| 跨节点 IP 限制 / 用户限速 | √ | √ | √ | √（hy2 仅设备限制） |
| 审计规则 / 自定义 DNS | √ | √ | √ | √ |
| 动态限速（TCP 新连接） | √ | √ | √ | × |

> 动态限速：节点配置 `LimitConfig.EnableDynamicSpeedLimit`；累计上报流量超阈值后，对该用户施加临时更严限速（**新 TCP 连接**生效）。sing UDP / hysteria2 无用户级令牌桶，不支持动态限速。

## Docker

| Tag | 说明 |
|-----|------|
| `latest` | 稳定版 |
| `dev` | 开发版 |

```bash
docker pull ghcr.io/golimit/v2bx:latest   # 或 :dev

wget https://raw.githubusercontent.com/golimit/V2bX/dev/docker-compose.yaml
# 编辑 v2bx_config/config.json 后：
docker-compose up -d
```

## 配置

`v2bx_config/` 为 sing-box 内核示例：`config.json`、`sing_origin.json`、`hy2config.yaml`、`geoip.dat` / `geosite.dat`。  
默认仅适用于 sing-box；xray / hysteria2 请参考 [文档](https://v2bx.v-50.me/)。

节点 `LimitConfig` 动态限速字段（启用时须完整配置，否则启动失败）：

```json
"LimitConfig": {
  "EnableDynamicSpeedLimit": true,
  "DynamicSpeedLimitConfig": {
    "Periodic": 60,
    "Traffic": 1073741824,
    "SpeedLimit": 10,
    "ExpireTime": 30
  }
}
```

| 字段 | 单位 | 说明 |
|------|------|------|
| `Periodic` | 秒 | 检查累计流量的周期 |
| `Traffic` | 字节 | 触发阈值（本节点累计上报流量） |
| `SpeedLimit` | Mbps | 惩罚限速（与节点/用户静态限速取更严） |
| `ExpireTime` | 分钟 | 惩罚持续时间；过期后恢复，可再次累计触发 |

## 构建

```bash
# tags 可选：xray、sing、hysteria2
GOEXPERIMENT=jsonv2 go build -v -o build_assets/V2bX \
  -tags "sing xray hysteria2 with_quic with_grpc with_utls with_wireguard with_acme with_gvisor" \
  -trimpath -ldflags "-X 'github.com/InazumaV/V2bX/cmd.version=$version' -s -w -buildid="
```

推荐 Go **1.26.5**（Dockerfile 已锁定）。构建需 `GOEXPERIMENT=jsonv2`。

### 本地定制内核

`go.mod` 的 `replace` 指向 golimit fork，**不作为 submodule**（`.gitignore` 忽略），本地 clone 联调：

| 目录 | 仓库 | replace |
|------|------|---------|
| `sing-box_mod/` | [golimit/sing-box_mod](https://github.com/golimit/sing-box_mod) | `sagernet/sing-box` |
| `xray-core/` | [golimit/xray-core](https://github.com/golimit/xray-core) | `xtls/xray-core` |
| `sing-vmess/` | [golimit/sing-vmess](https://github.com/golimit/sing-vmess) | `sagernet/sing-vmess`（module path 仍声明为官方名） |

```bash
git clone -b v1.13.14-mod git@github.com:golimit/sing-box_mod.git sing-box_mod
git clone -b main git@github.com:golimit/xray-core.git xray-core
# cd xray-core && git checkout b17a88f9b46d
git clone -b main git@github.com:golimit/sing-vmess.git sing-vmess
```

本地联调时临时改 `replace` 为 `./...`，**勿提交进 CI**。生产仍用远程 tag / commit。改完子仓库：push 子仓 → 更新本仓 `go.mod`/`go.sum` → 提交。

## 免责声明

本项目为 vibe coding 产物，不保证兼容与可用性；使用后果自负。文档：[v2bx.v-50.me](https://v2bx.v-50.me/)

## Thanks

[InazumaV](https://github.com/InazumaV) · [wyx2685](https://github.com/wyx2685) · [XTLS/Xray-core](https://github.com/XTLS/Xray-core) · [SagerNet/sing-box](https://github.com/SagerNet/sing-box) · [XrayR](https://github.com/XrayR/XrayR) · [golimit/sing-box_mod](https://github.com/golimit/sing-box_mod) · [golimit/xray-core](https://github.com/golimit/xray-core) · [golimit/sing-vmess](https://github.com/golimit/sing-vmess)
