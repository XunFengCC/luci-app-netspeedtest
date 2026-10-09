# SPDX-License-Identifier: GPL-3.0-only
# Local SDK/feed recipe. An upstream submission must use a pinned release
# download instead of copying its adjacent checkout (see docs/development.md).
NETSPEED_SOURCE_DIR:=$(abspath $(dir $(realpath $(lastword $(MAKEFILE_LIST))))/..)

include $(TOPDIR)/rules.mk

PKG_NAME:=netspeed
PKG_VERSION:=1.0.0
PKG_RELEASE:=2
PKG_LICENSE:=GPL-3.0-only LGPL-3.0-only
PKG_LICENSE_FILES:=LICENSE engine/LGPL-3.0.txt
PKG_MAINTAINER:=FengYinYH <FengYinYH@icloud.com>
PKG_BUILD_PARALLEL:=1
# Release CI precompiles this exact source with pinned Go; source feed builds
# use OpenWrt's own Go host package and architecture mapping.
ifeq ($(NETSPEED_PREBUILT_ENGINE),)
PKG_BUILD_DEPENDS:=golang/host
endif

include $(INCLUDE_DIR)/package.mk
include $(TOPDIR)/feeds/packages/lang/golang/golang-package.mk

define Package/netspeed-engine
  SECTION:=net
  CATEGORY:=Network
  TITLE:=Bounded LibreSpeed measurement engine
  URL:=https://github.com/FengYinYH/luci-app-netspeedtest
  DEPENDS:=$(GO_ARCH_DEPENDS)
endef

define Package/luci-app-netspeed
  SECTION:=luci
  CATEGORY:=LuCI
  SUBMENU:=3. Applications
  TITLE:=One-click speed test
  URL:=https://github.com/FengYinYH/luci-app-netspeedtest
  DEPENDS:=+netspeed-engine +luci-base +luci-compat +luci-lib-jsonc +luci-lib-nixio +lua +firewall4 +nftables-json
  PKGARCH:=all
endef

define Package/netspeed-openclash
  SECTION:=net
  CATEGORY:=Network
  TITLE:=Optional OpenClash integration for NetSpeed
  DEPENDS:=+luci-app-netspeed +curl +ruby +ruby-yaml
  PKGARCH:=all
endef

define Build/Prepare
	$(call Build/Prepare/Default)
	$(CP) $(NETSPEED_SOURCE_DIR)/engine $(PKG_BUILD_DIR)/
	$(CP) $(NETSPEED_SOURCE_DIR)/usr $(NETSPEED_SOURCE_DIR)/www $(PKG_BUILD_DIR)/
	$(CP) $(NETSPEED_SOURCE_DIR)/LICENSE $(PKG_BUILD_DIR)/
endef

define Build/Configure
endef

define Build/Compile
	$(if $(NETSPEED_PREBUILT_ENGINE),$(CP) $(NETSPEED_PREBUILT_ENGINE) $(PKG_BUILD_DIR)/netspeed-engine,cd $(PKG_BUILD_DIR)/engine && $(GO_PKG_VARS) CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags='-s -w' -o ../netspeed-engine .)
endef

define Package/netspeed-engine/install
	$(INSTALL_DIR) $(1)/usr/bin $(1)/usr/share/netspeed/licenses
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/netspeed-engine $(1)/usr/bin/
	$(INSTALL_DATA) $(PKG_BUILD_DIR)/engine/NOTICE $(PKG_BUILD_DIR)/engine/*.txt $(PKG_BUILD_DIR)/LICENSE $(1)/usr/share/netspeed/
	find $(PKG_BUILD_DIR)/engine/vendor -type f \( -iname 'license*' -o -iname 'copying*' -o -iname 'notice*' -o -iname 'patents*' \) | while read p; do rel=$$$${p#$(PKG_BUILD_DIR)/engine/vendor/}; mkdir -p "$(1)/usr/share/netspeed/licenses/$$$${rel%/*}"; cp "$$$$p" "$(1)/usr/share/netspeed/licenses/$$$$rel"; done
endef

define Package/luci-app-netspeed/install
	$(CP) $(PKG_BUILD_DIR)/www $(1)/
	$(INSTALL_DIR) $(1)/usr/libexec $(1)/usr/share/luci/menu.d $(1)/usr/share/rpcd/acl.d $(1)/usr/lib/lua/luci/i18n
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/usr/libexec/netspeed $(PKG_BUILD_DIR)/usr/libexec/netspeed-worker $(1)/usr/libexec/
	$(INSTALL_DATA) $(PKG_BUILD_DIR)/usr/share/luci/menu.d/* $(1)/usr/share/luci/menu.d/
	$(INSTALL_DATA) $(PKG_BUILD_DIR)/usr/share/rpcd/acl.d/* $(1)/usr/share/rpcd/acl.d/
	$(INSTALL_DATA) $(PKG_BUILD_DIR)/usr/lib/lua/luci/i18n/* $(1)/usr/lib/lua/luci/i18n/
endef

define Package/netspeed-openclash/install
	$(INSTALL_DIR) $(1)/usr/libexec
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/usr/libexec/netspeed-openclash $(PKG_BUILD_DIR)/usr/libexec/netspeed-proxy $(1)/usr/libexec/
endef

define Package/luci-app-netspeed/postinst
#!/bin/sh
[ -n "$$IPKG_INSTROOT" ] || { /etc/init.d/rpcd reload; rm -f /tmp/luci-indexcache; }
exit 0
endef

# APK executes preinst as pre-upgrade too; prerm is only for removal. Reject
# replacement during a task's launch window rather than race executable/config
# removal with a worker. Stale locks are reclaimed through the normal helper.
define NetSpeed/preinst
#!/bin/sh
if [ -z "$$IPKG_INSTROOT" ] && [ -d /tmp/netspeed/lock ]; then
  echo 'Stop the NetSpeed task before installing or upgrading.' >&2
  exit 1
fi
exit 0
endef

Package/luci-app-netspeed/preinst=$(NetSpeed/preinst)
Package/netspeed-engine/preinst=$(NetSpeed/preinst)
Package/netspeed-openclash/preinst=$(NetSpeed/preinst)

define NetSpeed/prerm
#!/bin/sh
if [ -z "$$IPKG_INSTROOT" ] && [ -x /usr/libexec/netspeed ]; then
  /usr/libexec/netspeed stop >/dev/null 2>&1 || exit 1
  for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
    [ -d /tmp/netspeed/lock ] || break
    sleep 0.1
  done
  [ ! -d /tmp/netspeed/lock ] || exit 1
fi
exit 0
endef

Package/luci-app-netspeed/prerm=$(NetSpeed/prerm)
Package/netspeed-engine/prerm=$(NetSpeed/prerm)
Package/netspeed-openclash/prerm=$(NetSpeed/prerm)

$(eval $(call BuildPackage,netspeed-engine))
$(eval $(call BuildPackage,luci-app-netspeed))
$(eval $(call BuildPackage,netspeed-openclash))
