package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/librespeed/speedtest-cli/defs"
)

type provider struct {
	ID          string
	Name        string
	Country     string
	CountryCC   string
	Sponsor     string
	BaseURL     string
	DownloadURL string
	UploadURL   string
	PingURL     string
	UserAgent   string
	proofOfWork bool
}

// These nodes are a small, explicit set. Automatic selection is restricted to
// healthy domestic nodes in direct mode; proxy mode can use any healthy
// catalog node because proxy egress geography may differ. Discovery only performs USTC PoW where required and empty ping
// requests, never data transfers or IP/telemetry lookups.
var providers = []provider{
	{ID: "10006", Name: "Moscow", Country: "Russia", CountryCC: "RU", Sponsor: "HOSTKEY", BaseURL: "http://spd-rudp.hostkey.ru/", DownloadURL: "garbage", UploadURL: "empty", PingURL: "empty"},
	{ID: "10004", Name: "City not specified", Country: "Australia", CountryCC: "AU", Sponsor: "Datto Workplace", BaseURL: "https://speedtest-au.workplace.datto.com/", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"},
	{ID: "10005", Name: "Paris", Country: "France", CountryCC: "FR", Sponsor: "Abadcer", BaseURL: "https://abadcer.com/speedtest/", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"},
	{ID: "52", Name: "New York", Country: "United States", CountryCC: "US", Sponsor: "Clouvider", BaseURL: "https://nyc.speedtest.clouvider.net/backend/", DownloadURL: "garbage.php", UploadURL: "empty.php", PingURL: "empty.php"},
	{ID: "93", Name: "Chicago", Country: "United States", CountryCC: "US", Sponsor: "Sharktech", BaseURL: "https://chispeed.sharktech.net/", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"},
	{ID: "101", Name: "Helsinki", Country: "Finland", CountryCC: "FI", Sponsor: "Pekka Jalonen / Hetzner", BaseURL: "https://www.librespeed.fi/", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"},
	{ID: "35", Name: "Rome", Country: "Italy", CountryCC: "IT", Sponsor: "GARR", BaseURL: "https://st-be-rm2.infra.garr.it/", DownloadURL: "garbage.php", UploadURL: "empty.php", PingURL: "empty.php"},
	{ID: "79", Name: "Prague", Country: "Czech Republic", CountryCC: "CZ", Sponsor: "CESNET", BaseURL: "https://speedtest.cesnet.cz/", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"},
	{ID: "74", Name: "Poznan", Country: "Poland", CountryCC: "PL", Sponsor: "Kamil Szczepanski / INEA", BaseURL: "https://speedtest.kamilszczepanski.com/", DownloadURL: "garbage.php", UploadURL: "empty.php", PingURL: "empty.php"},
	{ID: "106", Name: "Belgrade", Country: "Serbia", CountryCC: "RS", Sponsor: "Serbian Open eXchange", BaseURL: "https://speedtest1.sox.rs/librespeed/", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"},
	{ID: "104", Name: "Argalasti", Country: "Greece", CountryCC: "GR", Sponsor: "skoultsos.eu / Cosmote", BaseURL: "https://argalasti.skoultsos.eu/", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"},
	{ID: "10003", Name: "Brazil", Country: "Brazil", CountryCC: "BR", Sponsor: "Grupo IP", BaseURL: "https://st.grupoip.net.br/", DownloadURL: "backend/garbage", UploadURL: "backend/empty", PingURL: "backend/empty"},
	{ID: "54", Name: "美国洛杉矶", Country: "United States", CountryCC: "US", Sponsor: "Clouvider", BaseURL: "https://la.speedtest.clouvider.net/backend/", DownloadURL: "garbage.php", UploadURL: "empty.php", PingURL: "empty.php"},
	{ID: "91", Name: "美国洛杉矶", Country: "United States", CountryCC: "US", Sponsor: "Sharktech", BaseURL: "https://laxspeed.sharktech.net/", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"},
	{ID: "49", Name: "英国伦敦", Country: "United Kingdom", CountryCC: "GB", Sponsor: "Clouvider", BaseURL: "https://lon.speedtest.clouvider.net/backend/", DownloadURL: "garbage.php", UploadURL: "empty.php", PingURL: "empty.php"},
	{ID: "50", Name: "德国法兰克福", Country: "Germany", CountryCC: "DE", Sponsor: "Clouvider", BaseURL: "https://fra.speedtest.clouvider.net/backend/", DownloadURL: "garbage.php", UploadURL: "empty.php", PingURL: "empty.php"},
	{ID: "51", Name: "荷兰阿姆斯特丹", Country: "Netherlands", CountryCC: "NL", Sponsor: "Clouvider", BaseURL: "https://ams.speedtest.clouvider.net/backend/", DownloadURL: "garbage.php", UploadURL: "empty.php", PingURL: "empty.php"},
	{ID: "10001", Name: "USTC Hefei", Country: "China", CountryCC: "CN", Sponsor: "University of Science and Technology of China", BaseURL: "https://test.ustc.edu.cn", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php", proofOfWork: true},
	{ID: "10002", Name: "Southeast University", Country: "China", CountryCC: "CN", Sponsor: "Southeast University", BaseURL: "https://xnfz.seu.edu.cn/speed/", DownloadURL: "garbage.php", UploadURL: "empty.php", PingURL: "empty.php", UserAgent: "Mozilla/5.0"},
	{ID: "68", Name: "Singapore", Country: "Singapore", CountryCC: "SG", Sponsor: "Salvatore Cahyo", BaseURL: "https://speedtest.dsgroupmedia.com", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"},
	{ID: "82", Name: "Tokyo, Japan", Country: "Japan", CountryCC: "JP", Sponsor: "A573", BaseURL: "https://librespeed.a573.net", DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"},
}

func domesticProviders() []provider {
	out := make([]provider, 0, len(providers))
	for _, p := range providers {
		if p.CountryCC == "CN" {
			out = append(out, p)
		}
	}
	return out
}

func selectProvider(id string) (provider, bool) {
	if id == "" || id == "auto" {
		for _, p := range providers {
			if p.CountryCC == "CN" {
				return p, true
			}
		}
		return provider{}, false
	}
	for _, p := range providers {
		if p.ID == id {
			return p, true
		}
	}
	return provider{}, false
}

func (p provider) server(client *http.Client) *defs.Server {
	id, _ := strconv.Atoi(p.ID)
	return &defs.Server{
		ID: id, Name: p.Name, Server: p.BaseURL,
		DownloadURL: p.DownloadURL, UploadURL: p.UploadURL, PingURL: p.PingURL,
		SponsorName: p.Sponsor, Client: client, Referer: strings.TrimRight(p.BaseURL, "/") + "/",
	}
}

// checkProvider applies the same idle HTTP ping path used by the measurement
// core, accepting only successful empty responses (to reject HTML fallbacks).
func checkProvider(ctx context.Context, p provider, cfg config) (serverInfo, error) {
	client, peer, err := newProviderClient(cfg, p.UserAgent)
	if err != nil {
		return serverInfo{}, err
	}
	if p.proofOfWork {
		powCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = authenticateUSTC(powCtx, client, p.BaseURL)
		cancel()
		if err != nil {
			return serverInfo{}, fmt.Errorf("server %s PoW: %w", p.ID, err)
		}
	}
	probeCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	server := p.server(client)
	latency, jitter, _, _, err := server.PingAndJitterContext(probeCtx, 3)
	if err != nil {
		return serverInfo{}, err
	}
	info := p.info(peer.IP(), cfg.httpProxy != "")
	info.LatencyMS = latency
	_ = jitter
	return info, nil
}

// chooseFastestProvider validates candidates using three empty HTTP pings (the
// core discards the first setup-heavy probe), then picks the lowest warm
// latency. Callers constrain candidates to CN in direct mode and the full
// catalog in proxy mode. The selected node is measured again with a fresh client.
func chooseFastestProvider(ctx context.Context, cfg config, candidates []provider) (provider, error) {
	checkCtx, cancel := context.WithTimeout(ctx, serverListTimeout)
	defer cancel()
	var selected provider
	var bestLatency float64
	found := false
	type probe struct {
		provider provider
		info     serverInfo
		err      error
	}
	results := make(chan probe, len(candidates))
	count := 0
	for _, p := range candidates {
		if cfg.httpProxy == "" && p.CountryCC != "CN" {
			continue
		}
		count++
		go func(p provider) { info, err := checkProvider(checkCtx, p, cfg); results <- probe{p, info, err} }(p)
	}
	// Probe in parallel so a slow or unreachable country does not consume the
	// selection deadline before geographically closer nodes can be considered.
	for i := 0; i < count; i++ {
		r := <-results
		if r.err != nil || r.info.LatencyMS <= 0 {
			continue
		}
		if !found || r.info.LatencyMS < bestLatency {
			selected, bestLatency, found = r.provider, r.info.LatencyMS, true
		}
	}
	if !found {
		if cfg.httpProxy == "" {
			return provider{}, fmt.Errorf("no reachable domestic server")
		}
		return provider{}, fmt.Errorf("no reachable server through configured proxy")
	}
	return selected, nil
}

func (p provider) info(peerIP string, proxied bool) serverInfo {
	u, _ := url.Parse(p.BaseURL)
	host := ""
	if u != nil {
		host = u.Host
	}
	info := serverInfo{ID: p.ID, Name: p.Name, Country: p.Country, CountryCC: p.CountryCC, Sponsor: p.Sponsor, Host: host, Reachable: true}
	if proxied {
		info.TransportPeerIP = peerIP
	} else {
		info.PeerIP = peerIP
	}
	return info
}
