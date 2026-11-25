# GeoData

按需裁剪的轻量级 `geosite.dat` 与 `geoip.dat` 自动构建仓库，兼容 **Mihomo (Clash.Meta)**（OpenWrt Nikki / Clash Verge Rev / CMFA）。

每天 **北京时间早上 07:00（UTC 23:00）** 由 GitHub Actions 自动从 [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat) 提取所需分类，并合并 `rules/` 下的自定义规则打包发布至 `release` 分支与 GitHub Releases（`latest`）。

## 特点

- 只保留 `geosite.txt` 与 `geoip.txt` 中列出的分类，大幅减小文件体积与设备内存占用。
- 直接从上游编译好的 `.dat` 提取重打包，完整保留 `@cn`、`!cn` 等属性，无需自行维护 `include:` 依赖。
- 同时提供独立的 `moegirl.yaml`（萌娘百科非直连子域名）、`qiuwen.yaml`（求闻百科直连）与 `wikimedia.yaml`（Wikimedia / Wikipedia）规则文件，方便他人直接在 `rule-providers` 中单独引用。

## 1. 在 Mihomo 中引用裁剪后的 GeoData

```yaml
geox-url:
  mmdb: https://fastly.jsdelivr.net/gh/alecthw/mmdb_china_ip_list@release/lite/Country.mmdb
  geoip: https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/geoip.dat
  geosite: https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/geosite.dat
```

如使用本仓库的 `geosite.dat`，内置的自定义分类可直接通过 `GEOSITE` 调用：
- `GEOSITE,moegirl`
- `GEOSITE,qiuwen`
- `GEOSITE,grok`
- `GEOSITE,museai`
- `GEOSITE,cursor`
- `GEOSITE,wikimedia`
- `GEOSITE,hijack-reject`
- `GEOSITE,pcdn-reject`
- `GEOSITE,custom-direct`
- `GEOSITE,custom-proxy`
- `GEOIP,game-cdn`

> **首次切换提示**：若客户端本地已缓存了旧版 `geosite.dat`，首次启用上述自定义 `GEOSITE` 分类前，请先在客户端手动更新一次 Geo 数据库（或下载本仓库的 `geosite.dat` 覆盖到配置目录），否则预校验可能会因读取旧缓存而报错。

## 2. 单独调用萌娘百科、求闻百科与维基百科规则（`rule-providers`）

如果你不想更换自己的 `geosite.dat`，只想单独使用**萌娘百科**、**求闻百科**或 **Wikimedia（含 Wikipedia）** 规则，可按如下方式添加到配置中（推荐优先使用 `fastly.jsdelivr.net` CDN，也可替换为 Releases 直链）：

```yaml
rule-providers:
  moegirl:
    type: http
    behavior: classical
    interval: 86400
    path: ./rule_provider/moegirl.yaml
    url: "https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/rules/moegirl.yaml"
    # GitHub Releases 备用直链：
    # url: "https://github.com/lovelyCARDINAL/GeoData/releases/latest/download/moegirl.yaml"

  qiuwen:
    type: http
    behavior: classical
    interval: 86400
    path: ./rule_provider/qiuwen.yaml
    url: "https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/rules/qiuwen.yaml"
    # GitHub Releases 备用直链：
    # url: "https://github.com/lovelyCARDINAL/GeoData/releases/latest/download/qiuwen.yaml"

  wikimedia:
    type: http
    behavior: classical
    interval: 86400
    path: ./rule_provider/wikimedia.yaml
    url: "https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/rules/wikimedia.yaml"
    # GitHub Releases 备用直链：
    # url: "https://github.com/lovelyCARDINAL/GeoData/releases/latest/download/wikimedia.yaml"

  pcdn-reject:
    type: http
    behavior: classical
    interval: 86400
    path: ./rule_provider/pcdn-reject.yaml
    url: "https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/rules/pcdn-reject.yaml"
    # GitHub Releases 备用直链：
    # url: "https://github.com/lovelyCARDINAL/GeoData/releases/latest/download/pcdn-reject.yaml"

  # 仅支持单独挂载（含 PROCESS-NAME 规则，不打包进 geosite.dat）
  remote-desktop:
    type: http
    behavior: classical
    interval: 86400
    path: ./rule_provider/remote-desktop.yaml
    url: "https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/rules/remote-desktop.yaml"
    # GitHub Releases 备用直链：
    # url: "https://github.com/lovelyCARDINAL/GeoData/releases/latest/download/remote-desktop.yaml"

  # 仅支持单独挂载（含 Win + macOS PROCESS-NAME 规则，保留 adobe.com 官网访问）
  adobe-reject:
    type: http
    behavior: classical
    interval: 86400
    path: ./rule_provider/adobe-reject.yaml
    url: "https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/rules/adobe-reject.yaml"
    # GitHub Releases 备用直链：
    # url: "https://github.com/lovelyCARDINAL/GeoData/releases/latest/download/adobe-reject.yaml"

rules:
  # 请注意：moegirl务必放在 GEOSITE,category-wiki-cn,DIRECT 与 GEOSITE,cn,DIRECT 之前
  - "RULE-SET,moegirl,Proxy" #Proxy为你的代理策略组名称，自行替换
  - "RULE-SET,qiuwen,DIRECT"
  - "RULE-SET,wikimedia,Proxy"
  - "RULE-SET,pcdn-reject,REJECT"
  - "RULE-SET,remote-desktop,DIRECT"
  - "RULE-SET,adobe-reject,REJECT"
```

## 文件说明

- `geosite.txt`：需要保留的 `GEOSITE` 分类列表（大小写不敏感，支持 `#` 注释）
- `geoip.txt`：需要保留的 `GEOIP` 分类列表（大小写不敏感，支持 `#` 注释）
- `rules/`：自定义分流规则集（`behavior: classical`），包含：
  - `rules/moegirl.yaml`：萌娘百科非直连子域名规则
  - `rules/qiuwen.yaml`：求闻百科等直连站点规则
  - `rules/grok.yaml`：Grok / x.ai / grokbot 规则
  - `rules/museai.yaml`：MuseAI 规则
  - `rules/cursor.yaml`：Cursor / Anysphere 规则
  - `rules/wikimedia.yaml`：Wikimedia / Wikipedia 扩展域名规则
  - `rules/hijack-reject.yaml`：运营商劫持与国内恶意/流氓软件拦截规则
  - `rules/pcdn-reject.yaml`：国内视频/直播/服务商 PCDN 拦截规则
- `main.go`：按清单过滤并重打包 `.dat` 的构建程序
- `.github/workflows/build.yml`：每日定时自动构建并推送到 `release` 分支与 GitHub Releases 的工作流

## 上游署名与协议

本仓库采用 **GPL-3.0** 开源协议（与上游 `CC-BY-SA-4.0` 单向兼容），规则数据衍生并感谢以下上游项目：
- [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat)（GPL-3.0）
- [v2fly/domain-list-community](https://github.com/v2fly/domain-list-community)（MIT）
- [Aethersailor/Custom_OpenClash_Rules](https://github.com/Aethersailor/Custom_OpenClash_Rules)（CC-BY-SA-4.0）
