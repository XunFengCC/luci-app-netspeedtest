# NetSpeedTest

[简体中文](README.md) | **English**

A LuCI speed-test app for OpenWrt and ImmortalWrt: download and upload throughput, live charts, latency, jitter, automatic or manual test-server selection, and the last 30 successful results.

The default connection is direct. With the optional OpenClash integration, you can discover and test a proxy node without changing your everyday proxy selection. The interface follows LuCI's language and supports Simplified Chinese, Traditional Chinese, and English.

## Installation and compatibility

Download packages from [GitHub Releases](https://github.com/FengYinYH/luci-app-netspeedtest/releases). The current stable release is **1.0.0**, with package version **1.0.0-r2**. The app is not yet in the official OpenWrt feeds.

This release provides APKs for **OpenWrt 25.12-family firmware, ARM64, `aarch64_cortex-a53`, modern LuCI and fw4**. It has been tested on a Cudy TR3000 running ImmortalWrt 25.12. Stock OpenWrt has not yet been validated on hardware. Other architectures and OpenWrt 24.10 IPKs are not released; legacy fw3/iptables is unsupported.

Check the router's firmware and configured package architecture over SSH:

```sh
cat /etc/openwrt_release
cat /etc/apk/arch
```

Use `/etc/apk/arch`, rather than `apk --print-arch`, which reports the package manager's own build architecture. OpenWrt 24.10 uses IPK and cannot install these APKs.

### Files to download

| File | Purpose |
| --- | --- |
| `netspeed-engine-1.0.0-r2.apk` | Required: static LibreSpeed measurement engine |
| `luci-app-netspeed-1.0.0-r2.apk` | Required: interface, direct-test controller and history |
| `netspeed-openclash-1.0.0-r2.apk` | Optional: discover and test OpenClash proxy nodes |
| `SHA256SUMS` | Package checksums |

Runtime dependencies come from your router's configured feeds. Direct tests require LuCI's Lua compatibility modules and nftables. Proxy integration additionally requires curl, Ruby/YAML and an existing running OpenClash/Mihomo installation. Mihomo is not bundled.

### Install through LuCI

1. Download the two required APKs and verify their checksums against `SHA256SUMS`.
2. Open **System → Software** in LuCI and update the package lists.
3. Upload and install `netspeed-engine`, followed by `luci-app-netspeed`.
4. For proxy tests, also upload and install `netspeed-openclash`.
5. Refresh LuCI and open **Network → Speed Test**. If your firmware's upload interface cannot install an unsigned local APK, use the SSH commands below.

### Install through SSH

Upload the downloaded APKs to `/tmp/` on the router, verify their checksums, and run:

```sh
apk update
apk add --allow-untrusted /tmp/netspeed-engine-1.0.0-r2.apk /tmp/luci-app-netspeed-1.0.0-r2.apk
```

For proxy tests, also run:

```sh
apk add --allow-untrusted /tmp/netspeed-openclash-1.0.0-r2.apk
```

GitHub release packages are not signed by the official OpenWrt feeds. Use `--allow-untrusted` only for the local packages you have downloaded and verified; there is no need to enable it globally. Installation refreshes LuCI execution permissions without restarting networking or OpenClash.

## Usage

1. Open **Network → Speed test**. The defaults are **Direct** and **Automatic server**.
2. Click **Start test**. Latency, download and upload are measured in sequence. You can stop an active test.
3. Read throughput in Mbps, latency and jitter in ms, and both speed curves. Complete successful tests are saved automatically; failed or stopped tests do not save partial results.
4. To test a particular region, open the test-server menu and choose a server identified by its flag, country, city and provider. The two groups reflect current direct reachability; all catalog servers remain available for manual selection.
5. For proxy tests, install the optional integration and choose a discovered OpenClash physical node under the connection selector, then select a test server and start. Nodes come from **the router's own OpenClash installation**; the computer viewing LuCI does not need Clash. Your normal proxy selection stays unchanged.

History keeps the last 30 successful tests. Click a record for details, or use **Clear history** and confirm to remove records. The interface follows LuCI's selected language.

### Add a custom test server

Choose **Add custom server** in the test-server menu. Enter its name, a two-letter country code such as `CN` or `US`, an HTTP/HTTPS base URL, and three relative endpoint paths. The service must be LibreSpeed-compatible: download returns a binary payload, upload consumes the request body and returns a successful empty response, and ping returns a successful empty response.

| Field | Example |
| --- | --- |
| Name | My speed-test server |
| Country code | `US` |
| Base URL | `https://speed.example.net/` |
| Download endpoint | `backend/garbage.php` |
| Upload endpoint | `backend/empty.php` |
| Ping endpoint | `backend/empty.php` |

The example domain only illustrates the format; use your actual service URL. For endpoints at the root, use `garbage.php` and `empty.php`. URLs must not contain credentials or fragments, and endpoint paths must be relative. Up to 20 custom servers can be saved.

### Upgrade and uninstall

Stop the test first, then install matching new APKs using the same procedure. Package replacement is refused while a task is active.

To uninstall:

```sh
apk del netspeed-openclash luci-app-netspeed netspeed-engine
```

Omit `netspeed-openclash` if only the direct packages are installed. Removal stops tasks and cleans the app's temporary firewall rules. History and custom servers remain under `/etc/netspeed` and are available after reinstalling.

Proxy nodes with dependency chains such as `dialer-proxy`, or providers whose local configuration has not yet been downloaded, may not work in this version. Without the optional integration or usable physical nodes, only Direct is offered.

## Servers and measurement

The catalog contains 21 LibreSpeed-compatible servers across 17 countries. Servers are grouped by lightweight direct reachability probes. Direct automatic selection compares healthy mainland-China servers; proxy automatic selection compares catalog servers reachable through the selected proxy. Selection uses warm HTTP latency, which does not guarantee the highest upload or download capacity. Manual selection supports direct overseas and overseas-to-China tests.

LibreSpeed telemetry and getIP are disabled, and proxy credentials are not recorded. Service operators can still see the source address of ordinary connections. Results are stored locally on the router.

Latency is an HTTP round-trip measurement. Jitter follows LibreSpeed's smoothed adjacent-RTT differences. Curves show cumulative average payload throughput from the start of each phase. Exact definitions, failure paths and limits are documented in [engineering notes](docs/engineering.md) and the [engine README](engine/README.md).

## Author and contact

Author: FengYinYH (风吟). Contact: [FengYinYH@icloud.com](mailto:FengYinYH@icloud.com).

## Development and licensing

The engine requires Go 1.26 or newer. Offline dependencies and the minimal LibreSpeed fork are included in `engine/vendor`; regenerating vendor would overwrite the local changes. See [development notes](docs/development.md) for build and validation instructions.

Original project code is **GPL-3.0-only**, see [LICENSE](LICENSE). LibreSpeed retains LGPL v3, and other dependencies retain their own licenses. See [third-party notices](THIRD_PARTY.md) and the [engine NOTICE](engine/NOTICE). Binary distributions must include corresponding source, build instructions and notices, allowing recipients to modify the library and rebuild the application. The project is not affiliated with Ookla or Speedtest.
