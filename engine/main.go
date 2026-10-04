// netspeed-engine adapts the LibreSpeed measurement core to the router's
// short-lived NDJSON worker contract. USTC is the only automatic server;
// other vetted LibreSpeed nodes remain available for explicit comparisons.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	serverListTimeout        = 20 * time.Second
	workflowTimeout          = 75 * time.Second
	pingStageTimeout         = 10 * time.Second
	samplePeriod             = 500 * time.Millisecond
	defaultDurationSeconds   = 8
	minDurationSeconds       = 3
	maxDurationSeconds       = 10
	defaultConnections       = 8
	defaultUploadConnections = 4
	maxConnections           = 8
	maxRunBytes              = 3 << 30
	memoryLimit              = 48 << 20
)

type config struct {
	source            string
	connections       int
	uploadConnections int
	durationSeconds   int
	uid               int
	dns               string
	httpProxy         string
	proxyName         string
	customFile        string
}

type serverInfo struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Country         string  `json:"country"`
	CountryCC       string  `json:"country_code"`
	Sponsor         string  `json:"sponsor"`
	Host            string  `json:"host"`
	Distance        float64 `json:"distance_km,omitempty"`
	LatencyMS       float64 `json:"latency_ms,omitempty"`
	PeerIP          string  `json:"peer_ip,omitempty"`
	TransportPeerIP string  `json:"transport_peer_ip,omitempty"`
	Reachable       bool    `json:"reachable"`
}

type event struct {
	Type                    string      `json:"type"`
	Server                  *serverInfo `json:"server,omitempty"`
	Phase                   string      `json:"phase,omitempty"`
	LatencyMS               *float64    `json:"latency_ms,omitempty"`
	MinMS                   *float64    `json:"min_ms,omitempty"`
	MaxMS                   *float64    `json:"max_ms,omitempty"`
	JitterMS                *float64    `json:"jitter_ms,omitempty"`
	BytesPerS               float64     `json:"bytes_per_second,omitempty"`
	Mbps                    float64     `json:"mbps,omitempty"`
	Elapsed                 *float64    `json:"elapsed,omitempty"`
	Download                float64     `json:"download_mbps,omitempty"`
	Upload                  float64     `json:"upload_mbps,omitempty"`
	DownloadBytes           uint64      `json:"download_bytes,omitempty"`
	UploadBytes             uint64      `json:"upload_bytes,omitempty"`
	PhaseDuration           float64     `json:"phase_duration_seconds,omitempty"`
	DownloadDuration        float64     `json:"download_duration_seconds,omitempty"`
	UploadDuration          float64     `json:"upload_duration_seconds,omitempty"`
	ConnectionCount         int         `json:"connection_count,omitempty"`
	DownloadConnectionCount int         `json:"download_connection_count,omitempty"`
	UploadConnectionCount   int         `json:"upload_connection_count,omitempty"`
	Message                 string      `json:"message,omitempty"`
	Country                 string      `json:"country,omitempty"`
	ConnectionMode          string      `json:"connection_mode,omitempty"`
	ProxyName               string      `json:"proxy_name,omitempty"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	debug.SetMemoryLimit(memoryLimit)
	debug.SetGCPercent(50)
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		_ = os.Unsetenv(key)
	}

	cfg := config{source: os.Getenv("NETSPEED_SOURCE"), connections: defaultConnections, uploadConnections: defaultUploadConnections, durationSeconds: defaultDurationSeconds, dns: "223.5.5.5:53"}
	global := flag.NewFlagSet("netspeed-engine", flag.ContinueOnError)
	global.SetOutput(io.Discard)
	global.StringVar(&cfg.source, "source", cfg.source, "source IP address or Linux interface name")
	global.IntVar(&cfg.connections, "connections", cfg.connections, "parallel download connections (1-8)")
	global.IntVar(&cfg.uploadConnections, "upload-connections", cfg.uploadConnections, "parallel upload connections (1-8)")
	global.IntVar(&cfg.durationSeconds, "duration", cfg.durationSeconds, "measurement window in seconds (3-10)")
	global.IntVar(&cfg.uid, "uid", 0, "drop root privileges to this numeric UID before network activity")
	global.StringVar(&cfg.dns, "dns", cfg.dns, "direct DNS resolver address")
	global.StringVar(&cfg.httpProxy, "http-proxy", "", "HTTP proxy listener at 127.0.0.1:PORT (optional)")
	global.StringVar(&cfg.proxyName, "proxy-name", "", "non-secret label for the configured local proxy")
	global.StringVar(&cfg.customFile, "custom-file", "/etc/netspeed/custom.json", "saved custom LibreSpeed servers")
	if err := global.Parse(args); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	args = global.Args()
	if err := loadCustomProviders(cfg.customFile); err != nil {
		return err
	}
	if err := validateConnectionCounts(cfg.connections, cfg.uploadConnections); err != nil {
		return err
	}
	if err := validateDurationSeconds(cfg.durationSeconds); err != nil {
		return err
	}
	if cfg.uid < 0 {
		return errors.New("--uid must be zero (disabled) or a positive numeric UID")
	}
	if err := validateProxyOptions(cfg); err != nil {
		return err
	}
	if cfg.uid > 0 {
		if err := syscall.Setgroups([]int{}); err != nil {
			return fmt.Errorf("clear supplementary groups before dropping to uid %d: %w", cfg.uid, err)
		}
		if err := syscall.Setgid(cfg.uid); err != nil {
			return fmt.Errorf("set gid %d before network activity: %w", cfg.uid, err)
		}
		if err := syscall.Setuid(cfg.uid); err != nil {
			return fmt.Errorf("set uid %d before network activity: %w", cfg.uid, err)
		}
	}
	if len(args) == 0 {
		return errors.New("usage: netspeed-engine [--source IP_OR_INTERFACE] [--connections N] [--upload-connections N] [--duration SECONDS] [--uid UID] [--dns IP:PORT] [--http-proxy 127.0.0.1:PORT] [--proxy-name LABEL] servers | run [auto|server-id]")
	}

	switch args[0] {
	case "validate-custom":
		return nil // loadCustomProviders already validated the entire candidate file.
	case "servers":
		if len(args) != 1 {
			return errors.New("usage: netspeed-engine [flags] servers")
		}
		ctx, cancel := context.WithTimeout(context.Background(), serverListTimeout)
		defer cancel()
		servers := make([]serverInfo, 0, len(providers))
		// Metadata probes run concurrently; discovery never transfers bulk data.
		type checked struct {
			index int
			info  serverInfo
		}
		results := make(chan checked, len(providers))
		for i, p := range providers {
			go func(i int, p provider) {
				info, err := checkProvider(ctx, p, cfg)
				if err != nil {
					info = p.info("", false)
				}
				info.Reachable = err == nil
				results <- checked{i, info}
			}(i, p)
		}
		for range providers {
			result := <-results
			servers = append(servers, result.info)
		}
		sort.SliceStable(servers, func(i, j int) bool {
			if servers[i].CountryCC != servers[j].CountryCC {
				return servers[i].CountryCC == "CN"
			}
			if servers[i].LatencyMS != servers[j].LatencyMS {
				return servers[i].LatencyMS < servers[j].LatencyMS
			}
			return servers[i].ID < servers[j].ID
		})
		return json.NewEncoder(stdout).Encode(servers)
	case "run":
		if len(args) > 2 {
			return errors.New("usage: netspeed-engine [flags] run auto|server-id")
		}
		id := "auto"
		if len(args) == 2 {
			id = args[1]
		}
		ctx, cancel := context.WithTimeout(context.Background(), workflowTimeout)
		defer cancel()
		return runTest(ctx, stdout, id, cfg)
	default:
		return errors.New("expected command: servers or run")
	}
}

func runTest(ctx context.Context, stdout io.Writer, requestedID string, cfg config) error {
	auto := requestedID == "" || requestedID == "auto"
	var p provider
	if auto {
		candidates := domesticProviders()
		if cfg.httpProxy != "" {
			candidates = providers
		}
		selected, err := chooseFastestProvider(ctx, cfg, candidates)
		if err != nil {
			return emitFailure(stdout, err)
		}
		p = selected
	} else {
		selected, ok := selectProvider(requestedID)
		if !ok {
			return emitFailure(stdout, fmt.Errorf("server id %q was not found", requestedID))
		}
		p = selected
	}
	client, peer, err := newProviderClient(cfg, p.UserAgent)
	if err != nil {
		return emitFailure(stdout, err)
	}
	if p.proofOfWork {
		powCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = authenticateUSTC(powCtx, client, p.BaseURL)
		cancel()
		if err != nil {
			return emitFailure(stdout, fmt.Errorf("prepare USTC verification: %w", err))
		}
	}
	if err := emit(stdout, event{Type: "network", Country: p.Country, ConnectionMode: connectionMode(cfg), ProxyName: cfg.proxyName}); err != nil {
		return err
	}
	if err := emit(stdout, event{Type: "phase", Phase: "ping"}); err != nil {
		return err
	}
	server := p.server(client)
	pingCtx, cancelPing := context.WithTimeout(ctx, pingStageTimeout)
	latency, jitter, minLatency, maxLatency, err := server.PingAndJitterContext(pingCtx, 10)
	cancelPing()
	if err != nil {
		return emitFailure(stdout, fmt.Errorf("ping server %s: %w", p.ID, err))
	}
	if err := emit(stdout, event{Type: "server", Server: ptr(p.info(peer.IP(), cfg.httpProxy != ""))}); err != nil {
		return err
	}
	if err := emit(stdout, event{Type: "ping", LatencyMS: floatPtr(latency), JitterMS: floatPtr(jitter), MinMS: floatPtr(minLatency), MaxMS: floatPtr(maxLatency)}); err != nil {
		return err
	}

	budget := newByteBudget(maxRunBytes)
	phaseWindow := time.Duration(cfg.durationSeconds) * time.Second
	phaseContextTimeout := phaseWindow + 5*time.Second
	if err := emit(stdout, event{Type: "phase", Phase: "download"}); err != nil {
		return err
	}
	downloadCtx, cancelDownload := context.WithTimeout(ctx, phaseContextTimeout)
	var downloadDuration float64
	downMbps, downloadBytes, err := server.DownloadContext(downloadCtx, cfg.connections, 100, phaseWindow, samplePeriod, budget, func(bytesPerSecond float64, elapsed time.Duration) {
		downloadDuration = elapsed.Seconds()
		_ = emit(stdout, sampleEvent("download", bytesPerSecond, elapsed))
	})
	cancelDownload()
	if err != nil {
		return emitFailure(stdout, fmt.Errorf("download test: %w", err))
	}

	if err := emit(stdout, event{Type: "phase", Phase: "upload"}); err != nil {
		return err
	}
	upCtx, cancelUpload := context.WithTimeout(ctx, phaseContextTimeout)
	var uploadDuration float64
	upMbps, uploadBytes, err := server.UploadContext(upCtx, cfg.uploadConnections, 256, phaseWindow, samplePeriod, budget, func(bytesPerSecond float64, elapsed time.Duration) {
		uploadDuration = elapsed.Seconds()
		_ = emit(stdout, sampleEvent("upload", bytesPerSecond, elapsed))
	})
	cancelUpload()
	if err != nil {
		return emitFailure(stdout, fmt.Errorf("upload test: %w", err))
	}
	if !validRate(downMbps) || !validRate(upMbps) {
		return emitFailure(stdout, fmt.Errorf("LibreSpeed returned an invalid result (download=%g Mbps, upload=%g Mbps)", downMbps, upMbps))
	}
	info := p.info(peer.IP(), cfg.httpProxy != "")
	return emit(stdout, event{
		Type: "result", Download: downMbps, Upload: upMbps,
		DownloadBytes: downloadBytes, UploadBytes: uploadBytes,
		PhaseDuration: phaseWindow.Seconds(), DownloadDuration: downloadDuration,
		UploadDuration: uploadDuration, LatencyMS: floatPtr(latency),
		JitterMS: floatPtr(jitter), Server: &info, ConnectionCount: cfg.connections,
		DownloadConnectionCount: cfg.connections,
		UploadConnectionCount:   cfg.uploadConnections,
		ConnectionMode:          connectionMode(cfg), ProxyName: cfg.proxyName,
	})
}

func validateConnectionCounts(download, upload int) error {
	if download < 1 || download > maxConnections {
		return fmt.Errorf("--connections must be between 1 and %d", maxConnections)
	}
	if upload < 1 || upload > maxConnections {
		return fmt.Errorf("--upload-connections must be between 1 and %d", maxConnections)
	}
	return nil
}

func validateDurationSeconds(seconds int) error {
	if seconds < minDurationSeconds || seconds > maxDurationSeconds {
		return fmt.Errorf("--duration must be between %d and %d seconds", minDurationSeconds, maxDurationSeconds)
	}
	return nil
}

func sampleEvent(phase string, bytesPerSecond float64, elapsed time.Duration) event {
	seconds := elapsed.Seconds()
	return event{
		Type: "sample", Phase: phase, BytesPerS: bytesPerSecond,
		Mbps: bytesPerSecond * 8 / 1_000_000, Elapsed: &seconds,
	}
}

func validRate(v float64) bool { return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }

func ptr(v serverInfo) *serverInfo { return &v }

func floatPtr(v float64) *float64 { return &v }

func emit(w io.Writer, value event) error {
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return nil
}

func emitFailure(w io.Writer, err error) error {
	message := userMessage(err)
	_ = emit(w, event{Type: "error", Message: message})
	return err
}

func userMessage(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "测速超时，请重试或更换节点"
	}
	if errors.Is(err, context.Canceled) {
		return "测速已取消"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "byte budget"):
		return "已达到本次测速流量上限"
	case strings.Contains(message, "server id") || strings.Contains(message, "was not found"):
		return "测速节点无效，请重新选择"
	case strings.Contains(message, "domestic server"):
		return "当前没有可用的国内测速节点，请稍后重试"
	case strings.Contains(message, "reachable server"):
		return "当前没有可用的测速节点，请检查代理或稍后重试"
	case strings.Contains(message, "pow") || strings.Contains(message, "proof") || strings.Contains(message, "verification"):
		return "节点验证未完成，请更换节点或稍后重试"
	case strings.Contains(message, "status ") || strings.Contains(message, "response") || strings.Contains(message, "no successful") || strings.Contains(message, "no confirmed") || strings.Contains(message, "all transfer workers"):
		return "测速节点响应异常，请更换节点或稍后重试"
	default:
		return "网络测试失败，请检查网络后重试"
	}
}
