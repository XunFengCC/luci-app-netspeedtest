# 开发与 OpenWrt 收录

```sh
cd engine
go test -mod=vendor ./...
go test -mod=vendor -race ./...
go vet -mod=vendor ./...
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags='-s -w' -o ../build/netspeed-engine .
```

Go 1.26+。改变 `vendor/defs` 适配必须更新 NOTICE 和测量测试；禁止在 CI 对公共节点做饱和测试。

`openwrt/Makefile` 是本仓库 SDK/feed 配方，用同一源码生成引擎、LuCI 和可选代理包。可作为本地 package 目录连接到 SDK。标准 Go host 编译依赖通过 OpenWrt feeds 提供；发布构建也可给定 `NETSPEED_PREBUILT_ENGINE`，先由固定 Go 工具链从本次源码编译，然后由 SDK 打包，不能替换为未知来源二进制。

Linux 构建机须满足 SDK 的主机依赖，尤其是 GNU awk（mawk 缺少 asort 会让软件包扫描失败）、编译器与 ncurses 开发头文件。SDK 和 feeds 使用同一发行版的固定版本；公开安装包附 `BUILDINFO` 和 SHA256。

发行打包使用 `NETSPEED_PACKAGE_ONLY=1 sh scripts/sdk-build.sh SDK_DIR /absolute/path/to/build/netspeed-engine`，运行依赖由设备源提供，不在此阶段从头编译 Ruby/Rust。完整固件集成使用普通构建目标。

仅包维护者、项目 URL 等元数据变化时递增 `PKG_RELEASE`，使用独立的 `v<版本>-r<修订>` 标签发布同源的三个包，不移动既有标签或替换旧附件。例如 1.0.0-r2 同步公开作者身份与联系邮箱，不改变测量代码。已配置的 SDK 可直接使用上述脚本的 package-only make 目标，复用固定 feeds，避免在修订发行时刷新依赖。每次仍需核对 APK 版本、架构、维护者、URL、依赖和生命周期脚本，并附对应源码及校验值；已有实机验证不等于本次执行了设备升级验证。

APK 架构必须核对设备 `/etc/apk/arch`；`apk --print-arch` 显示工具自身的编译架构，不能替代这个配置。实机两者分别为 `aarch64_cortex-a53`、`aarch64`，后者标签的候选包被模拟安装正确拒绝；最终使用官方 SDK 的原始 `aarch64_cortex-a53` 标签，不改架构配置。

## 官方软件源流程

OpenWrt 用户在 LuCI 的软件包管理页面看到的是其配置软件源，不是统一审核制应用商店。独立 GitHub Releases 不会自动进入官方源。

官方收录通常需要向 [openwrt/packages](https://github.com/openwrt/packages) 提交后端配方，向 [openwrt/luci](https://github.com/openwrt/luci) 提交 LuCI 应用；通过维护者审核后进入相应 feed。遵循 [packages CONTRIBUTING](https://github.com/openwrt/packages/blob/master/CONTRIBUTING.md) 与 [OpenWrt 创建软件包文档](https://openwrt.org/docs/guide-developer/packages)。

当前已发行独立正式版，尚未官方收录。上游提交前还需要：固定发行源码 URL/校验值、用官方 Go host toolchain 完整构建、核对 LuCI 国际化惯例和后端接口、验证纯 OpenWrt 实机安装升级卸载、确认 maintainer 和签署者信息。OpenClash 不是官方源依赖，因此可选集成不得成为直连包的强制依赖。

开源仓库是工程正本；私人的部署工具、网络诊断和恢复包在发布仓库之外维护。
