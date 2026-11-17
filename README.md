# GeoData

按需裁剪的轻量级 `geosite.dat` 与 `geoip.dat` 自动构建仓库，兼容 **Mihomo (Clash.Meta)**（OpenWrt Nikki / Clash Verge Rev / CMFA）。

每天 **北京时间凌晨 02:00（UTC 18:00）** 由 GitHub Actions 自动从 [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat) 提取所需分类并重新打包发布至 `release` 分支。

## 特点

- 只保留 `geosite.txt` 与 `geoip.txt` 中列出的分类，大幅减小文件体积与设备内存占用。
- 直接从上游编译好的 `.dat` 提取重打包，完整保留 `@cn`、`!cn` 等属性，无需自行维护 `include:` 依赖。
- 增删分类只需编辑 `geosite.txt` / `geoip.txt` 并手动触发一次 Workflow 即可。

## 在 Mihomo 中引用

```yaml
geox-url:
  mmdb: https://fastly.jsdelivr.net/gh/alecthw/mmdb_china_ip_list@release/lite/Country.mmdb
  geoip: https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/geoip.dat
  geosite: https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/geosite.dat
```

## 自定义规则调用方式

`rules/` 目录下的每个规则文件（`moegirl.yaml`、`grok.yaml`、`museai.yaml`、`cursor.yaml`）同时支持**两种调用方式**：

1. **通过本仓库的 `geosite.dat` 直接调用**：构建时会自动将这 4 个分类打包进 `geosite.dat`，可直接写 `GEOSITE,moegirl,Proxy`、`GEOSITE,grok,AI`、`GEOSITE,museai,AI`、`GEOSITE,cursor,AI`。
2. **作为独立 `rule-provider` 单独调用**（方便他人按需引用，无需更换整个 `geosite.dat`）：
   ```yaml
   rule-providers:
     moegirl:
       type: http
       behavior: classical
       interval: 86400
       path: ./rule_provider/moegirl.yaml
       url: "https://fastly.jsdelivr.net/gh/lovelyCARDINAL/GeoData@release/rules/moegirl.yaml"
   ```

## 文件说明

- `geosite.txt`：需要保留的 `GEOSITE` 分类列表（大小写不敏感，支持 `#` 注释）
- `geoip.txt`：需要保留的 `GEOIP` 分类列表（大小写不敏感，支持 `#` 注释）
- `rules/`：自定义分流规则集（`behavior: classical`），包含：
  - `rules/moegirl.yaml`：萌娘百科非直连子域名规则
  - `rules/grok.yaml`：Grok / x.ai 规则
  - `rules/museai.yaml`：MuseAI 规则
  - `rules/cursor.yaml`：Cursor / Anysphere 规则
  - `rules/wikimedia.yaml`：Wikimedia / Wikipedia 扩展域名规则
  - `rules/hijack-reject.yaml`：运营商劫持与国内恶意/流氓软件拦截规则
  - `rules/qiuwen.yaml`：求闻百科等直连站点规则
- `main.go`：按清单过滤并重打包 `.dat` 的构建程序
- `.github/workflows/build.yml`：每日定时自动构建并推送到 `release` 分支的工作流
