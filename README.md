# NetSpeed · 一键测速

OpenWrt / ImmortalWrt 的 LuCI 测速插件：下载、上传、实时曲线、延迟、抖动、自动或手动选择测速服务器，以及最近 30 次历史记录。

连接方式默认直连。安装 OpenClash 的路由器可以自动发现代理节点，选择一个节点进行测试；不改变日常代理选择。界面跟随 LuCI 语言，支持简体中文、繁体中文和英文。

## 安装与适用版本

安装包发布在 [GitHub Releases](https://github.com/XunFengCC/luci-app-netspeed/releases)。选择与你的 **OpenWrt 版本、CPU 软件包架构**一致的包。OpenWrt 25.12 使用 APK，24.10 使用 IPK；不能混用。尚未进入 OpenWrt 官方软件源。

首次公开版本以现代 LuCI / fw4 为目标。已实测 Cudy TR3000、ARM64、ImmortalWrt 25.12；其他平台的构建通过不等于已经实机验证。旧 fw3/iptables 不支持。

三个软件包分别是：

- `netspeed-engine`：静态 LibreSpeed 测量核心。
- `luci-app-netspeed`：界面、直连任务控制及历史。需要 Lua 的 LuCI 兼容模块和 nftables。
- `netspeed-openclash`：可选的 OpenClash 发现与测速集成，需要 curl、Ruby/YAML，以及已经安装并运行的 OpenClash/Mihomo；本项目不分发 Mihomo。

下载同一发行版、同一架构的前两个包，在 LuCI 软件包管理页面上传安装，或按 [安装说明](docs/installation.md) 使用 SSH。需要代理功能时再安装第三个包。

## 节点与测量

内置 21 个 LibreSpeed 兼容服务器，覆盖 17 个国家，按当前直连探测结果分组。节点是提供下载载荷、接收上传和空响应延迟接口的测速服务，普通网址不能替代。支持添加最多 20 个自定义兼容服务。

直连自动选择当前健康的中国大陆节点；代理自动选择通过所选代理可达的目录节点。选择依据是暖连接延迟，不能保证服务器的上传或下载容量最高。手动选择可用于直连海外或海外回国测试。

不发送 LibreSpeed telemetry，不调用 getIP，不记录代理凭据。服务提供者仍可看到正常连接源地址。历史保存在路由器 `/etc/netspeed`，可在界面清空。

延迟是 HTTP 往返时间，抖动沿用 LibreSpeed 的相邻 RTT 平滑差值。曲线展示从每个阶段开始累计的平均有效载荷速度。完整算法、限制及失败路径见 [工程说明](docs/engineering.md) 和 [引擎说明](engine/README.md)。

## 开发与许可

引擎需要 Go 1.26 或更新版本，离线依赖与最小 fork 已保存在 `engine/vendor`。不要直接重新生成 vendor，否则会覆盖本地适配。构建和验证见 [开发说明](docs/development.md)。

本项目原创代码使用 **GPL-3.0-only**，见 [LICENSE](LICENSE)。LibreSpeed 库保留 LGPL v3，其他依赖保留自己的许可证；见 [第三方许可](THIRD_PARTY.md) 和 [引擎 NOTICE](engine/NOTICE)。分发二进制时须同时提供对应源码、构建方式和许可，允许用户修改库并重建整个程序。项目与 Ookla / Speedtest 没有隶属关系。
