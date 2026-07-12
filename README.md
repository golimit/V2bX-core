# V2bX

[![](https://img.shields.io/badge/TgChat-UnOfficialV2Board%E4%BA%A4%E6%B5%81%E7%BE%A4-green)](https://t.me/unofficialV2board)
[![](https://img.shields.io/badge/TgChat-YuzukiProjects%E4%BA%A4%E6%B5%81%E7%BE%A4-blue)](https://t.me/YuzukiProjects)

A V2board node server based on multi core, modified from XrayR.  
一个基于多种内核的V2board节点服务端，修改自XrayR，支持V2ay,Trojan,Shadowsocks协议。

**本仓库由 [golimit](https://github.com/golimit) 维护，基于 [wyx2685/V2bX](https://github.com/wyx2685/V2bX) 的二次开发版本。**

**注意： 本项目需要搭配[修改版V2board](https://github.com/wyx2685/v2board)**

## 特点

* 永久开源且免费。
* 支持Vmess/Vless, Trojan， Shadowsocks, Hysteria1/2多种协议。
* 支持Vless和XTLS等新特性。
* 支持单实例对接多节点，无需重复启动。
* 支持限制在线IP。
* 支持限制Tcp连接数。
* 支持节点端口级别、用户级别限速。
* 配置简单明了。
* 修改配置自动重启实例。
* 支持多种内核，易扩展。
* 支持条件编译，可仅编译需要的内核。

## 功能介绍

| 功能        | v2ray | trojan | shadowsocks | hysteria1/2 |
|-----------|-------|--------|-------------|----------|
| 自动申请tls证书 | √     | √      | √           | √        |
| 自动续签tls证书 | √     | √      | √           | √        |
| 在线人数统计    | √     | √      | √           | √        |
| 审计规则      | √     | √      | √           | √         |
| 自定义DNS    | √     | √      | √           | √        |
| 在线IP数限制   | √     | √      | √           | √        |
| 连接数限制     | √     | √      | √           | √         |
| 跨节点IP数限制  |√      |√       |√            |√          |
| 按照用户限速    | √     | √      | √           | √         |
| 动态限速(未测试) | √     | √      | √           | √         |

## TODO

- [ ] 重新实现动态限速
- [ ] 完善使用文档

## Docker 镜像

| Tag | 说明 | 适用场景 |
|-----|------|----------|
| `latest` | 稳定版（默认） | 生产环境推荐 |
| `dev` | 开发版 | 测试新功能 |

```bash
# 拉取稳定版
docker pull ghcr.io/golimit/v2bx:latest

# 拉取开发版
docker pull ghcr.io/golimit/v2bx:dev
```

### docker-compose 部署

```bash
# 下载配置文件
wget https://raw.githubusercontent.com/golimit/V2bx-core/dev/docker-compose.yaml

# 编辑配置
vi v2bx_config/config.json

# 启动服务
docker-compose up -d
```

## 配置说明

`v2bx_config/` 目录下为默认的 sing-box 内核配置示例：

- `config.json` - V2bX 主配置文件
- `sing_origin.json` - sing-box 内核配置
- `hy2config.yaml` - Hysteria2 内核配置
- `geoip.dat` / `geosite.dat` - GeoIP/GeoSite 数据库

**注意：默认配置仅适用于 sing-box 内核，如需使用 xray 或 hysteria2 内核，请参考[官方文档](https://v2bx.v-50.me/)自行配置。**

## 软件安装

### 一键安装

```
wget -N https://raw.githubusercontent.com/wyx2685/V2bX-script/master/install.sh && bash install.sh
```

### 手动安装

[手动安装教程](https://v2bx.v-50.me/v2bx/v2bx-xia-zai-he-an-zhuang/install/manual)

## 构建
``` bash
# 通过-tags选项指定要编译的内核， 可选 xray， sing, hysteria2
GOEXPERIMENT=jsonv2 go build -v -o build_assets/V2bX -tags "sing xray hysteria2 with_quic with_grpc with_utls with_wireguard with_acme with_gvisor" -trimpath -ldflags "-X 'github.com/InazumaV/V2bX/cmd.version=$version' -s -w -buildid="
```

### Go 版本说明

当前推荐使用 Go 1.26.5（Dockerfile 已默认配置）。

**性能优化特性：**
- Go 1.26 引入 Green Tea GC（垃圾回收器）转正并默认启用
- 通过改善内存局部性和 CPU 可扩展性，专门优化"大量小对象"的标记和扫描效率
- GC 开销降低 10%～40%（具体取决于工作负载）
- 对 Intel Ice Lake / AMD Zen 4 及更新的 CPU，可额外多省约 10%（使用向量指令加速小对象扫描）
- 对 V2bX 这类高并发代理程序（大量短生命周期小对象分配）特别有效
- 预期可带来 3%～5% 的端到端延迟改善

**注意事项：**
- `GOEXPERIMENT=jsonv2` 在 Go 1.26 中仍正常工作
- 从 Go 1.27 开始，jsonv2 将成为默认行为，`GOEXPERIMENT=jsonv2` 开关语义会翻转（变成"回退到旧实现"的开关）
- 如遇延迟波动，可用 `GOEXPERIMENT=nogreenteagc` 临时关闭新 GC 做 A/B 对比测试（该开关在 Go 1.27 将移除）
- 建议生产环境锁定具体小版本（如 `golang:1.26.5-alpine`），避免使用浮动标签导致构建不可复现

## 配置文件及详细使用教程

[详细使用教程](https://v2bx.v-50.me/)

## 免责声明

* 此项目用于本人自用，因此本人不能保证向后兼容性。
* 由于本人能力有限，不能保证所有功能的可用性，如果出现问题请在Issues反馈。
* 本人不对任何人使用本项目造成的任何后果承担责任。
* 本人比较多变，因此本项目可能会随想法或思路的变动随性更改项目结构或大规模重构代码，若不能接受请勿使用。

## 赞助

[赞助链接](https://v-50.me/)

## Thanks

* [wyx2685](https://github.com/wyx2685) - 原项目作者，感谢其对 V2bX 项目的开创和贡献
* [Project X](https://github.com/XTLS/)
* [V2Fly](https://github.com/v2fly)
* [VNet-V2ray](https://github.com/ProxyPanel/VNet-V2ray)
* [Air-Universe](https://github.com/crossfw/Air-Universe)
* [XrayR](https://github.com/XrayR/XrayR)
* [sing-box](https://github.com/SagerNet/sing-box)

## Stars 增长记录

[![Stargazers over time](https://starchart.cc/wyx2685/V2bX.svg)](https://starchart.cc/wyx2685/V2bX)
