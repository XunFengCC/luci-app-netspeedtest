package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/librespeed/speedtest-cli/defs"
)

func TestUSTCPoWFlowKeepsCookieAcrossRequests(t *testing.T) {
	var powVerified atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "ready", Path: "/"})
			w.WriteHeader(http.StatusOK)
		case "/backend/pow.php":
			if _, err := r.Cookie("session"); err != nil {
				http.Error(w, "missing session", http.StatusForbidden)
				return
			}
			_ = json.NewEncoder(w).Encode(powChallenge{Challenge: "test-challenge", Token: "short-lived-token", Difficulty: 8})
		case "/backend/pow_verify.php":
			var got struct{ Token, Nonce string }
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil || got.Token != "short-lived-token" {
				http.Error(w, "invalid proof", http.StatusForbidden)
				return
			}
			proof := sha256.Sum256([]byte("test-challenge" + got.Nonce))
			if strings.HasPrefix(strings.ToLower(hexString(proof[:])), "00") == false {
				http.Error(w, "wrong nonce", http.StatusForbidden)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "pow", Value: "accepted", Path: "/backend"})
			powVerified.Store(true)
			w.WriteHeader(http.StatusOK)
		case "/backend/empty.php":
			if _, err := r.Cookie("pow"); err != nil {
				http.Error(w, "missing proof cookie", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, _, err := newProviderClient(config{dns: "127.0.0.1:53"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := authenticateUSTC(ctx, client, server.URL); err != nil {
		t.Fatalf("authenticateUSTC: %v", err)
	}
	if !powVerified.Load() {
		t.Fatal("verify endpoint did not receive a valid proof")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/backend/empty.php", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("empty endpoint status = %d; proof cookie was not retained", resp.StatusCode)
	}
}

func TestProviderDiscoveryRejectsHTMLPingFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html>not a ping backend</html>")
	}))
	defer server.Close()
	p := provider{ID: "test", Name: "fake", Country: "Test", CountryCC: "ZZ", BaseURL: server.URL, PingURL: "empty.php"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := checkProvider(ctx, p, config{dns: "127.0.0.1:53"}); err == nil {
		t.Fatal("expected HTML fallback to be rejected")
	}
}

func TestEmptyPHPResponseWithDefaultHTMLContentTypeIsValidPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	p := provider{ID: "test", Name: "fake", Country: "Test", CountryCC: "ZZ", BaseURL: server.URL, PingURL: "empty.php"}
	info, err := checkProvider(context.Background(), p, config{dns: "127.0.0.1:53"})
	if err != nil {
		t.Fatalf("empty 2xx response with PHP's default MIME type was rejected: %v", err)
	}
	if info.LatencyMS <= 0 {
		t.Fatalf("probe did not record latency: %#v", info)
	}
}

func TestSelectProviderAutoIsDomesticAndManualNodesRemainAvailable(t *testing.T) {
	auto, ok := selectProvider("auto")
	if !ok || auto.ID != "10001" || auto.CountryCC != "CN" {
		t.Fatalf("auto provider = %#v, %v; want USTC mainland", auto, ok)
	}
	for _, id := range []string{"10001", "10002", "68", "82"} {
		if _, ok := selectProvider(id); !ok {
			t.Errorf("manual server %s is unavailable", id)
		}
	}
	if _, ok := selectProvider("999"); ok {
		t.Fatal("unknown provider unexpectedly selected")
	}
}

func TestProviderUserAgentIsScopedAndDiscoveryUsesWarmPings(t *testing.T) {
	var seen atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.UserAgent(); got != "Mozilla/5.0" {
			t.Errorf("provider User-Agent = %q, want provider-specific value", got)
		}
		seen.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	p := provider{ID: "test", Name: "fake", Country: "China", CountryCC: "CN", BaseURL: server.URL, PingURL: "empty.php", UserAgent: "Mozilla/5.0"}
	info, err := checkProvider(context.Background(), p, config{dns: "127.0.0.1:53"})
	if err != nil {
		t.Fatalf("checkProvider: %v", err)
	}
	if seen.Load() != 3 {
		t.Fatalf("discovery made %d pings, want three with the first discarded", seen.Load())
	}
	if info.LatencyMS <= 0 {
		t.Fatalf("warm ping latency missing: %#v", info)
	}
}

func TestAutoSelectsFastestHealthyDomesticNodeOnly(t *testing.T) {
	newNode := func(id, cc string, delay time.Duration) (*httptest.Server, provider) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(delay)
			w.WriteHeader(http.StatusOK)
		}))
		return srv, provider{ID: id, Name: id, Country: "test", CountryCC: cc, BaseURL: srv.URL, PingURL: "empty.php"}
	}
	slowServer, slow := newNode("slow-cn", "CN", 25*time.Millisecond)
	defer slowServer.Close()
	fastServer, fast := newNode("fast-cn", "CN", time.Millisecond)
	defer fastServer.Close()
	foreignServer, foreign := newNode("foreign", "SG", time.Microsecond)
	defer foreignServer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	selected, err := chooseFastestProvider(ctx, config{dns: "127.0.0.1:53"}, []provider{slow, fast, foreign})
	if err != nil {
		t.Fatalf("choose fastest domestic: %v", err)
	}
	if selected.ID != fast.ID {
		t.Fatalf("selected %s; want lower-latency domestic %s", selected.ID, fast.ID)
	}
	if _, err := chooseFastestProvider(ctx, config{dns: "127.0.0.1:53"}, []provider{foreign}); err == nil {
		t.Fatal("auto selection fell back to a foreign server")
	}
}

func TestAutoSelectionCanUseForeignProviderOnlyThroughProxy(t *testing.T) {
	foreign := provider{ID: "foreign", Country: "SG", CountryCC: "SG"}
	ctx := context.Background()
	if _, err := chooseFastestProvider(ctx, config{}, []provider{foreign}); err == nil {
		t.Fatal("direct auto selection accepted a foreign server")
	}
	// Candidate reachability itself is tested through the HTTP CONNECT tests;
	// this assertion locks the proxy-mode geographic policy.
	if got := connectionMode(config{httpProxy: "127.0.0.1:1234"}); got != "proxy" {
		t.Fatalf("connection mode = %q", got)
	}
}

func TestSampleEventUsesObservedBytesPerSecond(t *testing.T) {
	e := sampleEvent("upload", 125000, 1500*time.Millisecond)
	if e.Type != "sample" || e.Phase != "upload" || e.BytesPerS != 125000 || e.Mbps != 1 || e.Elapsed == nil || *e.Elapsed != 1.5 {
		t.Fatalf("unexpected sample event: %#v", e)
	}
}

func TestZeroJitterIsPresentInPingAndResultEvents(t *testing.T) {
	for _, kind := range []string{"ping", "result"} {
		var out strings.Builder
		if err := emit(&out, event{Type: kind, LatencyMS: floatPtr(0), JitterMS: floatPtr(0)}); err != nil {
			t.Fatal(err)
		}
		var got map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
			t.Fatal(err)
		}
		if string(got["jitter_ms"]) != "0" || string(got["latency_ms"]) != "0" {
			t.Fatalf("%s omitted zero-valued metrics: %s", kind, out.String())
		}
	}
}

func TestEngineErrorsAreShortChineseMessages(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{context.DeadlineExceeded, "测速超时，请重试或更换节点"},
		{errors.New("download endpoint returned invalid HTTP response (status 403)"), "测速节点响应异常，请更换节点或稍后重试"},
		{errors.New("prepare USTC verification: invalid proof"), "节点验证未完成，请更换节点或稍后重试"},
	}
	for _, tc := range cases {
		if got := userMessage(tc.err); got != tc.want {
			t.Errorf("userMessage(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestDownloadAndUploadConnectionCountsValidateIndependently(t *testing.T) {
	if err := validateConnectionCounts(8, 4); err != nil {
		t.Fatalf("valid asymmetric count was rejected: %v", err)
	}
	for _, counts := range [][2]int{{0, 4}, {4, 0}, {9, 4}, {4, 9}} {
		if err := validateConnectionCounts(counts[0], counts[1]); err == nil {
			t.Errorf("connection counts %v were accepted", counts)
		}
	}
}

func TestDurationIsBoundedToThreeThroughTenSeconds(t *testing.T) {
	for _, seconds := range []int{3, 5, 10} {
		if err := validateDurationSeconds(seconds); err != nil {
			t.Errorf("duration %d rejected: %v", seconds, err)
		}
	}
	for _, seconds := range []int{0, 2, 11} {
		if err := validateDurationSeconds(seconds); err == nil {
			t.Errorf("duration %d accepted outside the supported range", seconds)
		}
	}
}

func TestProductDefaultsUseEightDownloadFourUploadAndEightSeconds(t *testing.T) {
	if defaultConnections != 8 || defaultUploadConnections != 4 || defaultDurationSeconds != 8 {
		t.Fatalf("defaults = download %d, upload %d, duration %d; want 8, 4, 8", defaultConnections, defaultUploadConnections, defaultDurationSeconds)
	}
}

func TestResultEventRetainsPayloadBytesAndPhaseDurations(t *testing.T) {
	var out strings.Builder
	if err := emit(&out, event{
		Type: "result", DownloadBytes: 1234, UploadBytes: 5678,
		PhaseDuration: 3, DownloadDuration: 3.61, UploadDuration: 3.62, ConnectionCount: 8, DownloadConnectionCount: 8, UploadConnectionCount: 4,
	}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatal(err)
	}
	if got["download_bytes"] != float64(1234) || got["upload_bytes"] != float64(5678) || got["phase_duration_seconds"] != float64(3) || got["download_duration_seconds"] != 3.61 || got["upload_duration_seconds"] != 3.62 || got["connection_count"] != float64(8) || got["download_connection_count"] != float64(8) || got["upload_connection_count"] != float64(4) {
		t.Fatalf("result event lost transfer totals or duration: %s", out.String())
	}
}

func hexString(src []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(src)*2)
	for i, b := range src {
		out[2*i] = digits[b>>4]
		out[2*i+1] = digits[b&15]
	}
	return string(out)
}

func testDefinition(serverURL string) *defs.Server {
	return &defs.Server{Server: serverURL, DownloadURL: "backend/garbage.php", UploadURL: "backend/empty.php", PingURL: "backend/empty.php"}
}

func TestDownloadUsesLiveByteSamplesAndNormalDurationIsNotFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		flusher, _ := w.(http.Flusher)
		chunk := strings.Repeat("x", 8192)
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		deadline := time.NewTimer(600 * time.Millisecond)
		defer deadline.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-deadline.C:
				return
			case <-ticker.C:
				if _, err := io.WriteString(w, chunk); err != nil {
					return
				}
				if flusher != nil {
					flusher.Flush()
				}
			}
		}
	}))
	defer server.Close()

	var samples []float64
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	mbps, total, err := testDefinition(server.URL).DownloadContext(ctx, 1, 100, 180*time.Millisecond, 35*time.Millisecond, nil, func(rate float64, _ time.Duration) {
		samples = append(samples, rate)
	})
	if err != nil {
		t.Fatalf("normal phase duration returned error: %v", err)
	}
	if total == 0 || mbps <= 0 {
		t.Fatalf("no measured download: total=%d bytes, rate=%g Mbps", total, mbps)
	}
	if len(samples) < 2 {
		t.Fatalf("got %d progress samples, want several real timed samples", len(samples))
	}
	for i, sample := range samples {
		if sample <= 0 {
			t.Fatalf("sample %d did not reflect transferred bytes: %g", i, sample)
		}
	}
}

func TestDownloadRejectsNonSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "denied", http.StatusForbidden)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _, err := testDefinition(server.URL).DownloadContext(ctx, 1, 100, 100*time.Millisecond, 25*time.Millisecond, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("download error = %v; want rejected HTTP 403", err)
	}
}

func TestDownloadRejectsEarlyFailureAfterPartialBytes(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = io.WriteString(w, "partial payload")
			return
		}
		http.Error(w, "backend unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, total, err := testDefinition(server.URL).DownloadContext(ctx, 1, 100, 500*time.Millisecond, 25*time.Millisecond, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("download returned a partial result after its worker failed: bytes=%d err=%v", total, err)
	}
	if total == 0 {
		t.Fatal("test setup did not transfer the intended partial bytes")
	}
}

func TestUploadRequiresSuccessfulResponseBeforeCountingBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		http.Error(w, "rejected", http.StatusForbidden)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, total, err := testDefinition(server.URL).UploadContext(ctx, 1, 1, 100*time.Millisecond, 25*time.Millisecond, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("upload error = %v; want rejected HTTP 403", err)
	}
	if total != 0 {
		t.Fatalf("rejected upload counted %d bytes as accepted", total)
	}
}

func TestUploadCountsOnlyCompletedAcceptedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength != 1024 {
			http.Error(w, "wrong content length", http.StatusBadRequest)
			return
		}
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			return
		}
		w.WriteHeader(http.StatusNoContent)
		time.Sleep(20 * time.Millisecond)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	mbps, total, err := testDefinition(server.URL).UploadContext(ctx, 1, 1, 120*time.Millisecond, 30*time.Millisecond, nil, nil)
	if err != nil {
		t.Fatalf("accepted upload returned error: %v", err)
	}
	if total < 1024 || mbps <= 0 || total%1024 != 0 {
		t.Fatalf("accepted upload count/rate invalid: total=%d, Mbps=%g", total, mbps)
	}
}

func TestUploadAcceptsEmptyPHPConfirmationRegardlessOfMIMEType(t *testing.T) {
	var complete atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := io.Copy(io.Discard, r.Body)
		if err == nil && n == r.ContentLength {
			complete.Add(1)
		}
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, total, err := testDefinition(server.URL).UploadContext(ctx, 1, 1, 100*time.Millisecond, 25*time.Millisecond, nil, nil)
	if err != nil || total == 0 {
		t.Fatalf("empty 2xx PHP confirmation was not accepted: bytes=%d err=%v", total, err)
	}
	if complete.Load() == 0 {
		t.Fatal("test server did not receive a complete accepted upload")
	}
}

func TestExternalStageDeadlineIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, _, err := testDefinition(server.URL).DownloadContext(ctx, 1, 100, time.Second, 20*time.Millisecond, nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("download error = %v; want external deadline error", err)
	}
}

func TestSharedRunByteBudgetRejectsIncompleteWindow(t *testing.T) {
	budget := newByteBudget(1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, "more than budget")
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, total, err := testDefinition(server.URL).DownloadContext(ctx, 1, 100, 200*time.Millisecond, 20*time.Millisecond, budget, nil)
	if !errors.Is(err, defs.ErrTransferBudgetReached) {
		t.Fatalf("budget exhaustion error = %v; want bounded-window rejection", err)
	}
	if total != 1 {
		t.Fatalf("download bytes = %d, want budgeted 1 byte", total)
	}
	if used := budget.Reserve(1); used != 0 {
		t.Fatalf("byte budget allowed another %d bytes after exhaustion", used)
	}
}
