# 网速测试（NetSpeedTest）

**简体中文** | [English](README.en.md)

OpenWrt / ImmortalWrt 的 LuCI 测速插件：下载、上传、实时曲线、延迟、抖动、自动或手动选择测速服务器，以及最近 30 次历史记录。

连接方式默认直连。安装 OpenClash 的路由器可以自动发现代理节点，选择一个节点进行测试；不改变日常代理选择。界面跟随 LuCI 语言，支持简体中文、繁体中文和英文。

## 安装与适用版本

安装包发布在 [GitHub Releases](https://github.com/FengYinYH/luci-app-netspeedtest/releases)。当前正式版为 **1.0.0**，软件包版本为 **1.0.0-r2**。尚未进入 OpenWrt 官方软件源。

首次公开版本提供 ARM64 / `aarch64_cortex-a53` / OpenWrt 25.12 系列的 APK，以现代 LuCI / fw4 为目标。已实测 Cudy TR3000、ImmortalWrt 25.12；纯 OpenWrt 尚无实机验收，其他架构和 24.10 IPK 尚未发行。旧 fw3/iptables 不支持。

先在路由器 SSH 中查看版本与架构：

```sh
cat /etc/openwrt_release
cat /etc/apk/arch
```

当前安装包要求 25.12 系列、`aarch64_cortex-a53`、现代 LuCI / fw4。`apk --print-arch` 显示工具自身的编译架构，不等同于 `/etc/apk/arch`。24.10 使用 IPK，不能安装本次 APK。

### 下载哪些文件

| 文件 | 用途 |
| --- | --- |
| `netspeed-engine-1.0.0-r2.apk` | 必需，静态 LibreSpeed 测量引擎 |
| `luci-app-netspeed-1.0.0-r2.apk` | 必需，界面、直连控制和历史 |
| `netspeed-openclash-1.0.0-r2.apk` | 可选，检测并测试 OpenClash 代理节点 |
| `SHA256SUMS` | 安装包校验值 |

依赖由路由器当前软件源提供。直连包需要 Lua 的 LuCI 兼容模块与 nftables；代理集成另需 curl、Ruby/YAML，以及已经运行的 OpenClash/Mihomo。本项目不分发 Mihomo。

### 从 LuCI 安装

1. 在 Release 下载前两个 APK，核对 `SHA256SUMS` 中对应文件的校验值。
2. 打开路由器 LuCI 的「系统 → 软件包」，更新软件源列表。
3. 上传并安装 `netspeed-engine`，再安装 `luci-app-netspeed`。
4. 需要代理测速时，再上传并安装 `netspeed-openclash`。
5. 刷新 LuCI，打开「网络 → 一键测速」。如果固件的上传界面不能安装本地未签名 APK，使用下面的 SSH 方法。

### 从 SSH 安装

将下载的 APK 上传到路由器 `/tmp/`，核对文件校验值后执行：

```sh
apk update
apk add --allow-untrusted /tmp/netspeed-engine-1.0.0-r2.apk /tmp/luci-app-netspeed-1.0.0-r2.apk
```

需要代理测速时再执行：

```sh
apk add --allow-untrusted /tmp/netspeed-openclash-1.0.0-r2.apk
```

GitHub 发行包没有官方 OpenWrt 软件源签名；`--allow-untrusted` 仅用于已经下载并校验的本地包，无需设置为全局选项。安装不会重启网络或 OpenClash，首次安装会刷新 LuCI 执行权限。

## 使用

1. 打开「网络 → 一键测速」。默认是「直连」和「自动选择节点」。
2. 点击「开始测速」，等待延迟、下载和上传依次完成；测速中可以停止。
3. 查看下载/上传 Mbps、延迟与抖动 ms，以及两条速度曲线。成功结果自动保存到历史，失败和停止不保存部分结果。
4. 要测试指定地区，打开「测速节点」菜单，选择一个带国旗、国家、城市和提供者名称的服务器。两个分组按当前直连探测结果划分，所有目录节点都可手动选择。
5. 要测试代理，先安装可选集成，然后在「连接方式」选择自动检测到的 OpenClash 物理节点，再选择测速服务器并开始。节点来自**路由器本机**，访问页面的电脑不需要安装 Clash；测速不改变日常代理选择。

历史保留最近 30 次成功结果，点击记录可查看详情；「清除历史记录」会在确认后删除记录。界面跟随 LuCI 语言，支持简中、繁中和英文。

### 添加自定义测速节点

在测速节点菜单选择「添加自定义节点」，填写名称、两位国家代码（如 `CN`、`US`）、HTTP/HTTPS 服务根地址和三个相对接口路径。服务必须兼容 LibreSpeed：下载返回二进制载荷，上传接收请求正文并返回成功空响应，延迟接口返回成功空响应。

| 字段 | 示例 |
| --- | --- |
| 名称 | 我的测速服务器 |
| 国家代码 | `US` |
| 服务地址 | `https://speed.example.net/` |
| 下载接口 | `backend/garbage.php` |
| 上传接口 | `backend/empty.php` |
| 延迟接口 | `backend/empty.php` |

示例域名仅展示格式，请填你的实际服务地址。接口位于根目录时可改成 `garbage.php`、`empty.php`；地址不得包含账号密码或 fragment，接口填写相对路径。最多保存 20 个自定义节点。

### 升级和卸载

先停止测速，再用相同方式安装匹配发行版/架构的新 APK。测速期间软件包会拒绝替换，避免正在运行的任务被打断。

卸载命令：

```sh
apk del netspeed-openclash luci-app-netspeed netspeed-engine
```

只安装直连包时省略 `netspeed-openclash`。卸载会停止任务并清理本插件的临时规则；历史和自定义节点保留在 `/etc/netspeed`，重装后仍可使用。

部分带 `dialer-proxy` 等依赖链的代理，以及尚未下载本地 provider 配置的节点，当前版本可能无法测试。缺少可选集成或没有可用物理节点时，连接方式只显示直连。

## 节点与测量

内置 21 个 LibreSpeed 兼容服务器，覆盖 17 个国家，按当前直连探测结果分组。节点是提供下载载荷、接收上传和空响应延迟接口的测速服务，普通网址不能替代。支持添加最多 20 个自定义兼容服务。

直连自动选择当前健康的中国大陆节点；代理自动选择通过所选代理可达的目录节点。选择依据是暖连接延迟，不能保证服务器的上传或下载容量最高。手动选择可用于直连海外或海外回国测试。

不发送 LibreSpeed telemetry，不调用 getIP，不记录代理凭据。服务提供者仍可看到正常连接源地址。历史保存在路由器 `/etc/netspeed`，可在界面清空。

延迟是 HTTP 往返时间，抖动沿用 LibreSpeed 的相邻 RTT 平滑差值。曲线展示从每个阶段开始累计的平均有效载荷速度。完整算法、限制及失败路径见 [工程说明](docs/engineering.md) 和 [引擎说明](engine/README.md)。

## 作者与联系

作者：风吟（FengYinYH）。联系邮箱：[FengYinYH@icloud.com](mailto:FengYinYH@icloud.com)。

## 开发与许可

引擎需要 Go 1.26 或更新版本，离线依赖与最小 fork 已保存在 `engine/vendor`。不要直接重新生成 vendor，否则会覆盖本地适配。构建和验证见 [开发说明](docs/development.md)。

本项目原创代码使用 **GPL-3.0-only**，见 [LICENSE](LICENSE)。LibreSpeed 库保留 LGPL v3，其他依赖保留自己的许可证；见 [第三方许可](THIRD_PARTY.md) 和 [引擎 NOTICE](engine/NOTICE)。分发二进制时须同时提供对应源码、构建方式和许可，允许用户修改库并重建整个程序。项目与 Ookla / Speedtest 没有隶属关系。
