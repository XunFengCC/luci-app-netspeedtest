# 安装、升级与卸载

先查看 `/etc/openwrt_release` 和 `/etc/apk/arch`（APK）或 `opkg print-architecture`（IPK），不仅看 CPU 名称。同一 ARM64 内核可以使用不同包架构标签；`apk --print-arch` 是工具编译架构，不等同于设备配置。

OpenWrt 25.12 / APK：

```sh
apk update
apk add --allow-untrusted /tmp/netspeed-engine-*.apk /tmp/luci-app-netspeed-*.apk
# 需要 OpenClash 集成时：
apk add --allow-untrusted /tmp/netspeed-openclash-*.apk
```

OpenWrt 24.10 / IPK（仅使用明确针对该发行版构建的包）：

```sh
opkg update
opkg install /tmp/netspeed-engine_*.ipk /tmp/luci-app-netspeed_*.ipk
opkg install /tmp/netspeed-openclash_*.ipk
```

GitHub 首次发布的包没有官方 OpenWrt 软件源签名。APK 本地安装的 `--allow-untrusted` 只用于你已经从本项目 release 下载并核对 SHA256 的包，不要设置为全局配置。依赖从设备当前软件源安装。

安装后打开「网络 → 一键测速」。正在测速时不要升级；停止后升级同架构软件包。卸载用 `apk del netspeed-openclash luci-app-netspeed netspeed-engine` 或相应 `opkg remove`。卸载前脚本会停止本插件的任务并清理临时规则；历史和自定义节点留在 `/etc/netspeed`，卸载不删除用户数据。

安装不重启网络或 OpenClash；首次权限安装需要 reload rpcd，使新的执行权限可用。已打开的 LuCI 页面可能需要刷新。

代理连接选项从路由器本机 OpenClash 的控制器和当前 provider 配置读取，与访问页面的电脑无关。缺少 OpenClash、Ruby/YAML或可用物理节点时只显示直连。节点有依赖链、dialer-proxy、仅远程尚未下载的 provider 配置时，当前版本可能不支持。
