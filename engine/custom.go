package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type customProvider struct {
	CountryCC string `json:"country_code"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Base      string `json:"base_url"`
	Download  string `json:"download_url"`
	Upload    string `json:"upload_url"`
	Ping      string `json:"ping_url"`
}

func validateCustomProvider(p customProvider) error {
	if p.CountryCC != "" && (len(p.CountryCC) != 2 || p.CountryCC[0] < 'A' || p.CountryCC[0] > 'Z' || p.CountryCC[1] < 'A' || p.CountryCC[1] > 'Z') {
		return fmt.Errorf("国家代码需填写两个大写英文字母")
	}
	if len(p.Name) == 0 || len(p.Name) > 80 || len(p.Base) > 2048 {
		return fmt.Errorf("自定义节点名称或地址无效")
	}
	u, err := url.Parse(p.Base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("请输入不含账号密码的 HTTP/HTTPS 服务地址")
	}
	for _, path := range []string{p.Download, p.Upload, p.Ping} {
		if len(path) == 0 || len(path) > 512 || strings.ContainsAny(path, "\r\n\x00") {
			return fmt.Errorf("测速接口路径无效")
		}
		target, err := url.Parse(path)
		if err != nil || target.IsAbs() || target.Host != "" {
			return fmt.Errorf("接口需填写相对服务地址的路径")
		}
	}
	return nil
}

func loadCustomProviders(path string) error {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(raw) > 65536 {
		return fmt.Errorf("自定义节点文件过大")
	}
	var rows []customProvider
	if err = json.Unmarshal(raw, &rows); err != nil {
		return err
	}
	if len(rows) > 20 {
		return fmt.Errorf("最多保存20个自定义节点")
	}
	for _, p := range rows {
		if err = validateCustomProvider(p); err != nil {
			return err
		}
		cc := p.CountryCC
		if cc == "" {
			cc = "CUSTOM"
		}
		providers = append(providers, provider{ID: p.ID, Name: p.Name, Country: "Custom", CountryCC: cc, Sponsor: "自定义", BaseURL: p.Base, DownloadURL: p.Download, UploadURL: p.Upload, PingURL: p.Ping, UserAgent: "Mozilla/5.0"})
	}
	return nil
}
