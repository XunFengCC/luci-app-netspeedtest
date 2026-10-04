package defs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ProgressCallback receives the LibreSpeed cumulative average in bytes/sec
// and elapsed wall time. The adapter samples this value at a fixed cadence.
type ProgressCallback func(bytesPerSecond float64, elapsed time.Duration)

// ByteBudget reserves payload bytes across the download and upload phases.
// Implementations should return a value <= n and refund reservations for
// requests the provider does not confirm.
type ByteBudget interface {
	Reserve(n uint64) uint64
	Release(n uint64)
}

var ErrTransferBudgetReached = errors.New("run transfer byte budget reached")

// DownloadContext performs the upstream LibreSpeed multistream download
// algorithm with caller cancellation, per-response validation and callbacks.
func (s *Server) DownloadContext(ctx context.Context, requests, chunks int, duration, sampleEvery time.Duration, budget ByteBudget, progress ProgressCallback) (float64, uint64, error) {
	if requests < 1 || duration <= 0 || sampleEvery <= 0 {
		return 0, 0, errors.New("invalid LibreSpeed download parameters")
	}
	u, err := s.GetURL()
	if err != nil {
		return 0, 0, err
	}
	u.Path = path.Join(u.Path, s.DownloadURL)
	query := u.Query()
	query.Set("ckSize", strconv.Itoa(chunks))
	query.Set("r", strconv.FormatInt(time.Now().UnixNano(), 10))
	u.RawQuery = query.Encode()

	counter := NewCounter()
	counter.Start()
	var firstErr error
	var firstErrOnce sync.Once
	budgetReached := make(chan struct{}, 1)
	request := func(testCtx context.Context) (uint64, error) {
		req, err := http.NewRequestWithContext(testCtx, http.MethodGet, u.String(), nil)
		if err != nil {
			return 0, err
		}
		req.Header.Set("User-Agent", clientUserAgent())
		if s.Referer != "" {
			req.Header.Set("Referer", s.Referer)
		}
		req.Header.Set("Accept-Encoding", "identity")
		resp, err := s.httpClient().Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || isHTMLResponse(resp) {
			return 0, fmt.Errorf("download endpoint returned invalid HTTP response (status %d, content-type %q)", resp.StatusCode, resp.Header.Get("Content-Type"))
		}
		writer := &budgetCountingWriter{counter: counter, budget: budget}
		_, copyErr := io.Copy(writer, resp.Body)
		if errors.Is(copyErr, ErrTransferBudgetReached) {
			select {
			case budgetReached <- struct{}{}:
			default:
			}
			return 0, nil
		}
		if copyErr != nil {
			return 0, copyErr
		}
		if writer.written == 0 {
			return 0, errors.New("download endpoint returned no payload bytes")
		}
		return 0, nil
	}
	err = runTransfer(ctx, requests, duration, sampleEvery, counter, request, progress, budgetReached, &firstErr, &firstErrOnce)
	if err != nil {
		return 0, counter.Total(), err
	}
	if counter.Total() == 0 {
		if firstErr != nil {
			return 0, 0, fmt.Errorf("no successful download response: %w", firstErr)
		}
		return 0, 0, errors.New("download test received no bytes")
	}
	return counter.AvgMbps(), counter.Total(), nil
}

// UploadContext performs LibreSpeed's parallel repeated-blob upload. Bytes are
// committed only after the endpoint returns 2xx and consumes the full request,
// so rejected requests cannot inflate the reported upload rate.
func (s *Server) UploadContext(ctx context.Context, requests, uploadSizeKiB int, duration, sampleEvery time.Duration, budget ByteBudget, progress ProgressCallback) (float64, uint64, error) {
	if requests < 1 || uploadSizeKiB < 1 || duration <= 0 || sampleEvery <= 0 {
		return 0, 0, errors.New("invalid LibreSpeed upload parameters")
	}
	u, err := s.GetURL()
	if err != nil {
		return 0, 0, err
	}
	u.Path = path.Join(u.Path, s.UploadURL)

	counter := NewCounter()
	counter.SetUploadSize(uploadSizeKiB)
	counter.GenerateBlob()
	payload := counter.Payload()
	counter.Start()
	var firstErr error
	var firstErrOnce sync.Once
	budgetReached := make(chan struct{}, 1)
	request := func(testCtx context.Context) (uint64, error) {
		reserved := uint64(len(payload))
		if budget != nil {
			reserved = budget.Reserve(reserved)
			if reserved != uint64(len(payload)) {
				budget.Release(reserved)
				select {
				case budgetReached <- struct{}{}:
				default:
				}
				return 0, ErrTransferBudgetReached
			}
		}
		var sent atomic.Uint64
		body := io.TeeReader(bytes.NewReader(payload), countingAtomicWriter{count: &sent})
		req, err := http.NewRequestWithContext(testCtx, http.MethodPost, u.String(), body)
		if err != nil {
			if budget != nil {
				budget.Release(reserved)
			}
			return 0, err
		}
		req.ContentLength = int64(len(payload))
		req.Header.Set("User-Agent", clientUserAgent())
		if s.Referer != "" {
			req.Header.Set("Referer", s.Referer)
		}
		req.Header.Set("Accept-Encoding", "identity")
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, err := s.httpClient().Do(req)
		if err != nil {
			if budget != nil {
				budget.Release(reserved)
			}
			return 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			if budget != nil {
				budget.Release(reserved)
			}
			return 0, fmt.Errorf("upload endpoint returned invalid HTTP response (status %d, content-type %q)", resp.StatusCode, resp.Header.Get("Content-Type"))
		}
		confirmation, err := io.ReadAll(io.LimitReader(resp.Body, 2))
		if err != nil {
			if budget != nil {
				budget.Release(reserved)
			}
			return 0, fmt.Errorf("read upload confirmation: %w", err)
		}
		if len(confirmation) != 0 {
			if budget != nil {
				budget.Release(reserved)
			}
			return 0, fmt.Errorf("upload endpoint returned a nonempty confirmation body")
		}
		if sent.Load() != uint64(len(payload)) {
			if budget != nil {
				budget.Release(reserved)
			}
			return 0, fmt.Errorf("upload endpoint acknowledged an incomplete request (%d of %d bytes sent)", sent.Load(), len(payload))
		}
		return sent.Load(), nil
	}
	err = runTransfer(ctx, requests, duration, sampleEvery, counter, request, progress, budgetReached, &firstErr, &firstErrOnce)
	if err != nil {
		return 0, counter.Total(), err
	}
	if counter.Total() == 0 {
		if firstErr != nil {
			return 0, 0, fmt.Errorf("no confirmed upload response: %w", firstErr)
		}
		return 0, 0, errors.New("upload test completed without an accepted request")
	}
	return counter.AvgMbps(), counter.Total(), nil
}

type transferRequest func(context.Context) (uint64, error)

func runTransfer(ctx context.Context, requests int, duration, sampleEvery time.Duration, counter *BytesCounter, transfer transferRequest, progress ProgressCallback, budgetReached <-chan struct{}, firstErr *error, firstErrOnce *sync.Once) error {
	testCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{}, requests)
	failed := make(chan struct{}, requests)
	var wg sync.WaitGroup
	active := 0
	spawn := func() {
		active++
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := transfer(testCtx)
			if n > 0 {
				counter.AddBytes(n)
			}
			if err != nil {
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					firstErrOnce.Do(func() { *firstErr = err })
				}
				select {
				case failed <- struct{}{}:
				case <-testCtx.Done():
				}
				return
			}
			select {
			case done <- struct{}{}:
			case <-testCtx.Done():
			}
		}()
	}
	start := time.Now()
	for i := 0; i < requests; i++ {
		if ctx.Err() != nil {
			cancel()
			wg.Wait()
			return ctx.Err()
		}
		spawn()
		if i+1 < requests {
			select {
			case <-time.After(200 * time.Millisecond):
			case <-ctx.Done():
				cancel()
				wg.Wait()
				return ctx.Err()
			}
		}
	}
	phaseTimer := time.NewTimer(duration)
	ticker := time.NewTicker(sampleEvery)
	defer phaseTimer.Stop()
	defer ticker.Stop()
running:
	for {
		select {
		case <-done:
			active--
			spawn()
		case <-failed:
			active--
			if active == 0 {
				if *firstErr != nil {
					cancel()
					wg.Wait()
					return fmt.Errorf("all transfer workers failed: %w", *firstErr)
				}
				cancel()
				wg.Wait()
				return errors.New("all transfer workers stopped before the measurement window ended")
			}
		case <-ticker.C:
			if progress != nil {
				progress(counter.CurrentSpeed(), time.Since(start))
			}
		case <-phaseTimer.C:
			break running
		case <-budgetReached:
			cancel()
			wg.Wait()
			return ErrTransferBudgetReached
		case <-ctx.Done():
			cancel()
			wg.Wait()
			return ctx.Err()
		}
	}
	cancel()
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}
	if progress != nil {
		progress(counter.CurrentSpeed(), time.Since(start))
	}
	return nil
}

type budgetCountingWriter struct {
	counter *BytesCounter
	budget  ByteBudget
	written uint64
}

func (w *budgetCountingWriter) Write(p []byte) (int, error) {
	n := uint64(len(p))
	if w.budget != nil {
		n = w.budget.Reserve(n)
	}
	if n > 0 {
		w.counter.AddBytes(n)
		w.written += n
	}
	if n < uint64(len(p)) {
		return int(n), ErrTransferBudgetReached
	}
	return len(p), nil
}

func isHTMLResponse(resp *http.Response) bool {
	return strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/html")
}

type countingAtomicWriter struct{ count *atomic.Uint64 }

func (w countingAtomicWriter) Write(p []byte) (int, error) {
	w.count.Add(uint64(len(p)))
	return len(p), nil
}

func clientUserAgent() string {
	if UserAgent == "" || strings.HasPrefix(UserAgent, "/ ") {
		return "netspeed-engine/1.0 (LibreSpeed v1.0.14)"
	}
	return UserAgent
}
