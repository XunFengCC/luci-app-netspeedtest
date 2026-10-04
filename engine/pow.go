package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxChallengeBody = 64 << 10

type powChallenge struct {
	Challenge  string `json:"challenge"`
	Token      string `json:"token"`
	Difficulty int    `json:"difficulty"`
}

// authenticateUSTC mirrors the public browser flow: initialize the same-origin
// session, fetch a short challenge, solve SHA-256(challenge || decimal nonce),
// and POST token+nonce. The cookie jar is held only by this run's HTTP client.
func authenticateUSTC(ctx context.Context, client *http.Client, base string) error {
	root, err := absoluteURL(strings.TrimRight(base, "/") + "/")
	if err != nil {
		return err
	}
	if _, err := getForPoW(ctx, client, root.String(), ""); err != nil {
		return fmt.Errorf("initialize USTC session: %w", err)
	}
	challengeURL := strings.TrimRight(base, "/") + "/backend/pow.php"
	challengeBody, err := getForPoW(ctx, client, challengeURL, root.String())
	if err != nil {
		return fmt.Errorf("fetch USTC proof challenge: %w", err)
	}
	var challenge powChallenge
	if err := json.Unmarshal(challengeBody, &challenge); err != nil {
		return fmt.Errorf("parse USTC proof challenge: %w", err)
	}
	if challenge.Challenge == "" || challenge.Token == "" || challenge.Difficulty < 0 || challenge.Difficulty > 64 {
		return fmt.Errorf("invalid USTC proof challenge parameters")
	}
	nonce, err := solvePoW(ctx, challenge.Challenge, challenge.Difficulty)
	if err != nil {
		return fmt.Errorf("solve USTC proof challenge: %w", err)
	}
	encoded, err := json.Marshal(struct {
		Token string `json:"token"`
		Nonce string `json:"nonce"`
	}{Token: challenge.Token, Nonce: nonce})
	if err != nil {
		return err
	}
	verifyURL := strings.TrimRight(base, "/") + "/backend/pow_verify.php"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(string(encoded)))
	if err != nil {
		return err
	}
	applyProviderHeaders(req, root.String())
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("verify USTC proof: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || responseIsHTML(resp) {
		return fmt.Errorf("USTC proof verification returned HTTP %d", resp.StatusCode)
	}
	if _, err := io.Copy(io.Discard, io.LimitReader(resp.Body, maxChallengeBody)); err != nil {
		return fmt.Errorf("read USTC proof verification response: %w", err)
	}
	emptyURL, _ := absoluteURL(strings.TrimRight(base, "/") + "/backend/empty.php")
	if len(client.Jar.Cookies(emptyURL)) == 0 {
		return fmt.Errorf("USTC proof verification did not establish a test cookie")
	}
	return nil
}

func getForPoW(ctx context.Context, client *http.Client, target, referer string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	applyProviderHeaders(req, referer)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxChallengeBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxChallengeBody {
		return nil, fmt.Errorf("response exceeded %d bytes", maxChallengeBody)
	}
	return body, nil
}

func solvePoW(ctx context.Context, challenge string, difficulty int) (string, error) {
	// The published browser code checks ceil(difficulty/4) zero hex digits.
	// Match that nibble rounding exactly rather than silently changing the site
	// protocol to a bit-level comparison.
	digits := (difficulty + 3) / 4
	want := strings.Repeat("0", digits)
	for nonce := uint64(0); nonce < 100_000_000; nonce++ {
		if nonce&1023 == 0 {
			if err := ctx.Err(); err != nil {
				return "", err
			}
		}
		candidate := fmt.Sprint(nonce)
		digest := sha256.Sum256([]byte(challenge + candidate))
		if hex.EncodeToString(digest[:])[:digits] == want {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no valid nonce found within search limit")
}

func applyProviderHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", "netspeed-engine/1.0 (LibreSpeed v1.0.14)")
	req.Header.Set("Accept", "application/json, application/octet-stream, */*")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}
