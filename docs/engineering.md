# 测量设计、证据与边界

## 背景

目标是在路由器 LuCI 中一键取得真实上下行、延迟、抖动和历史，而不是把镜像文件下载速度当完整测速。2026-10 首次实机为 Cudy TR3000 / ImmortalWrt 25.12 / ARM64，OpenClash 0.47.156、Mihomo 1.19.30。外部设备需要独立验证；本项目不宣称所有 OpenWrt 设备都有相同容量。

## 已证实的实现

测量使用 LibreSpeed CLI 1.0.14 的重复并行 HTTP 请求与字节计数，8 条下载、4 条上传连接、各 8 秒、500 ms 采样。曲线是阶段累计有效字节除以墙钟时间，两个方向各自从阶段开始计时；纵轴采用随范围扩展的整齐刻度。上传只有完整发送并取得成功空响应后才计数。失败、停止、负值及不完整结果不进历史。

延迟、抖动公式和时限在 `engine/README.md`。引擎有取消 context、75 秒总时限、3 GiB 有效载荷预算，worker 有 150 秒 watchdog。没有常驻引擎或数据库。实机静态引擎约 6.1 MiB，直连 RSS 约 11–12 MiB；Go 的 48 MiB 软内存限制不等于 OS 硬限制。

直连进程降为数字 UID/GID 65533，用独立 DNS，fw4 `nat_output`、`mangle_output` 临时加入只匹配该 UID 的 return。结束按自有 comment/handle 清理，不修改日常代理选择。节点发现只做轻量空响应；全目录并行健康探测不包含并行吞吐测试。

代理模式从本机 OpenClash 发现物理节点，复用设备已有 Mihomo 创建一次性单节点进程，仅监听 loopback，不复制订阅、全规则或 geodata。附加 RSS 实测约 35–38 MiB。HTTPS CONNECT 由代理端解析目标域名，代理失败不回退直连。孤儿清理核对 PID、专用 UID 和配置路径，端口冲突失败而不抢占。

实机的 USTC 完整结果约 195/26 Mbps；另一个兼容大学节点约 191/3.7 Mbps。这说明节点容量会影响结果，不说明所有宽带上限是这些数。同一引擎在电脑与路由器的对照曾暴露 socket 与短阶段影响；单 socket 接收缓冲调至 212992，并增加流数/时长后消除大部分差距。此调整会限制 socket 自动接收窗口，其他 RTT、内核及高带宽设备需重评。

代理到海外服务器的两次完整结果约 151/24、146/22 Mbps；这些是现场兼容和路径验证，不是稳定性 SLA。选中代理出口独立核对，主 OpenClash selector 未改变。

## 失败路径与取舍

- 早期 Ookla 兼容开源引擎出现上传负值且未返回 error，原生 TCP 路径也没有完整成功；未保存，不解释为宽带速度。官方闭源 CLI 不加入公开发行物，最终采用可重建的 LibreSpeed 最小 fork。
- 站点 empty 接口健康不保证大上传可用；一次 HTTP 403 正确失败。严格检查状态、正文及上传确认，避免拒绝页或 PHP 源码成为测速数据。
- 当前 Mihomo 动态 listener API 不可用，主配置 reload 会影响家庭网络；采用独立最小进程，代价是短暂增加内存。依赖链仍未支持。
- 路由器 Ruby 不包含桌面常见的 FileUtils/Socket/JSON；helper 只依赖 YAML 和核心 API，端口检查用 BusyBox netstat。
- BusyBox start-stop-daemon 的 numeric UID 仍要求 passwd 条目；改为引擎 Setuid，不写入系统账号。
- 硬杀 worker 后 trap 不执行：过期锁回收清理所属 engine/core/watchdog 和 nft 规则。停止、超时、重复开始和硬杀恢复已实机验证。历史 32 条隔离样本验证留下最新 30 条。
- 某次 LuCI 连静态文件也不响应，仅重启 uhttpd 恢复。具体根因未知；不能归因于 rpcd。部署只在 ACL 变化时 reload rpcd，减少无关干扰。

## 验证层次与未覆盖项

Go fake-server、race、vet，以及代理 CONNECT/DNS/不回退测试通过。实机验证真实上下行、history 对应、直连清理、停止与异常恢复；浏览器验证语言、菜单、历史清空及窄屏布局。

尚未验证旧 fw3、所有 CPU/固件、所有 provider 类型、任意第三方代理链和持续全天线路稳定性。UID 65533 是插件隔离资源，安装前须确认未被其他服务使用。临时规则失效会让测试失败而不保存；它不能阻止所有其他插件自定义的 output 接管方式。

私人凭据、家庭网络拓扑、原始截图、历史记录和现场恢复包不在公开仓库或发布包中。

## 首次公开软件包验证

2026-10-04，官方 OpenWrt 25.12.0 filogic SDK 生成三个 APK，包架构 `aarch64_cortex-a53`，原创工程版本 0.1.0-r2。同一来源的静态引擎由 Go 1.27.1 预编译，SDK 做最终 strip 和原生打包；没有在此打包阶段重建设备源提供的 Ruby/Rust 等运行依赖。

实机执行测试包 r0 → r1 → 最终 r2 的安装升级，再完整卸载和重装 r2；包管理器的 pre-install/pre-upgrade/pre-deinstall 脚本实际执行。持久数据 15 个文件的 SHA256 前后一致，卸载后 owned helper/engine、锁、临时规则无残留，重装后控制器节点发现和 LuCI HTTP 正常。没有重复饱和测速：测量核心未变，本阶段验证的是分发与生命周期。

打包检查曾发现两个问题并在发行前纠正：APK 的 `--print-arch` 是工具编译架构，必须读 `/etc/apk/arch`；Makefile 的脚本与安装 recipe 有不同变量展开层次，生成脚本中双 `$` 会成为 shell PID，而不是变量引用。最终用 SDK `apk adbdump` 逐项核对许可路径、架构与真实生命周期脚本。APK 升级不会执行 prerm，因此另外用 preinst/pre-upgrade 在测速锁存在时拒绝替换，防止启动窗口竞态。
