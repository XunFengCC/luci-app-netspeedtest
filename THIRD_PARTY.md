# 第三方许可与发布规则

原创前端、控制器、引擎适配和打包脚本使用 GPL-3.0-only。已有第三方文件保留其原有许可，不因仓库根许可证改变。

静态链接的 LibreSpeed CLI v1.0.14 使用 LGPL v3；上游 commit 为 `42e5f76c90e64e04ab7079fd76d833dfe41805da`。修改文件、原因与重建要求记录在 `engine/NOTICE`。每个二进制发布附完整对应源码（包括修改后的 vendor）及可复现构建脚本，以满足修改和重新组合所需的源码可用性；没有禁止逆向调试库修改的额外条款。

其他 vendored 模块的原始许可证保存在 `engine/vendor`：

| 模块 | 许可 |
| --- | --- |
| github.com/briandowns/spinner | Apache-2.0 |
| github.com/fatih/color | MIT |
| github.com/google/uuid | BSD-3-Clause |
| github.com/mattn/go-colorable | MIT |
| github.com/mattn/go-isatty | MIT |
| github.com/prometheus-community/pro-bing | MIT |
| golang.org/x/net、x/sync、x/sys、x/term | BSD-3-Clause；子目录可能附加来源许可，保留全文 |

LuCI 和 OpenClash/Mihomo 是设备上的运行依赖，不打进本项目安装包。翻译 LMO 由 LuCI po2lmo 格式生成，只包含本项目的翻译数据，不分发转换器。

静态引擎还包含 Go 标准库/运行时，采用 BSD-3-Clause。构建工具链 Go 1.27.1 的原文许可和专利授权保存在 `engine/Go-LICENSE.txt`、`engine/Go-PATENTS.txt`，同样安装至 `/usr/share/netspeed`；工具链更新时核对这些文本。

测速服务器目录只保存名称、位置和公开接口；不复制第三方站点代码。公网服务的可用性和使用规则可能变化。USTC 使用其公开正常 PoW 流程，不绕过验证；禁止 telemetry。不得在持续集成中对公共节点做饱和测速。

首次许可核对：2026-10-04。依据为 [LGPL v3 原文](https://www.gnu.org/licenses/lgpl-3.0.html)、项目内保留的上游许可证，以及各 vendored 依赖的 LICENSE 文件。官方收录和以后的依赖更新仍须重新核对。
