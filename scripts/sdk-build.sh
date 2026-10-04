#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
# Run in an extracted official SDK on Linux. The optional prebuilt engine must
# have been compiled from this checkout for the SDK architecture, never fetched
# from a separate release. SDK machinery creates native APK/IPK and scripts.
set -eu
source_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
sdk_dir=${1:?usage: sdk-build.sh SDK_DIR [PREBUILT_ENGINE]}
sdk_dir=$(CDPATH= cd -- "$sdk_dir" && pwd)
engine=${2:-}
cd "$sdk_dir"
./scripts/feeds update base packages luci
./scripts/feeds install luci-compat luci-lib-jsonc luci-lib-nixio ruby-yaml curl
if [ -z "$engine" ]; then
  ./scripts/feeds install golang
fi
if [ -e package/netspeed ] || [ -L package/netspeed ]; then
  [ "$(readlink -f package/netspeed)" = "$(readlink -f "$source_dir/openwrt")" ] || {
    echo 'package/netspeed is owned by another source checkout' >&2
    exit 1
  }
else
  ln -s "$source_dir/openwrt" package/netspeed
fi
cat >> .config <<'EOF'
CONFIG_PACKAGE_netspeed-engine=m
CONFIG_PACKAGE_luci-app-netspeed=m
CONFIG_PACKAGE_netspeed-openclash=m
CONFIG_SIGNED_PACKAGES=n
EOF
make defconfig
if [ "${NETSPEED_PACKAGE_ONLY:-0}" = 1 ]; then
  # For release packaging with an externally compiled static engine, runtime
  # dependencies remain in package metadata and come from the user's feed.
  # Rebuilding Ruby (including Rust/YJIT) is unnecessary for these script-only
  # packages. Full source/buildroot integration uses the ordinary target below.
  [ -n "$engine" ] || { echo 'package-only mode requires a same-source engine' >&2; exit 1; }
  make -C package/netspeed TOPDIR="$sdk_dir" compile V=s -j2 NETSPEED_PREBUILT_ENGINE="$engine"
else
  make package/netspeed/compile V=s -j2 NETSPEED_PREBUILT_ENGINE="$engine"
fi
