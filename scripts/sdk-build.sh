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
./scripts/feeds update packages luci
./scripts/feeds install luci-compat luci-lib-jsonc luci-lib-nixio ruby-yaml curl
if [ -z "$engine" ]; then
  ./scripts/feeds install golang
fi
ln -s "$source_dir/openwrt" package/netspeed
cat >> .config <<'EOF'
CONFIG_PACKAGE_netspeed-engine=m
CONFIG_PACKAGE_luci-app-netspeed=m
CONFIG_PACKAGE_netspeed-openclash=m
CONFIG_SIGNED_PACKAGES=n
EOF
make defconfig
make package/netspeed/compile V=s -j2 NETSPEED_PREBUILT_ENGINE="$engine"
