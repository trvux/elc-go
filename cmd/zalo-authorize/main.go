// Command zalo-authorize performs the one-time OAuth authorization needed
// before the Zalo OA lead-notification feature can send anything. This
// can't be reduced to "paste one API token" — Zalo issues the first
// access/refresh token pair only after a human approves the app in a
// browser (PKCE authorization-code flow), confirmed against current
// developers.zalo.me docs. After this runs once, internal/inquiry's
// background token refresher (see infrastructure/zalo_token_refresher.go)
// keeps the token fresh on its own — this tool is not needed again unless
// the refresh token itself expires (30-day lifetime) from prolonged inactivity.
//
// Usage:
//
//	ZALO_OA_APP_ID=... ZALO_OA_APP_SECRET=... ZALO_OA_REDIRECT_URI=https://yourdomain.com/callback \
//	  go run ./cmd/zalo-authorize
//
// ZALO_OA_REDIRECT_URI must exactly match a Callback URL already configured
// in the Zalo App console (developers.zalo.me) — Zalo rejects any other
// value. It doesn't need to be a real working page; a placeholder URL you
// control is enough, since you'll copy the `code` param off the resulting
// (likely 404) redirect and paste it back into this tool.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/trvux/elc-go/internal/inquiry/domain"
	inquiryinfra "github.com/trvux/elc-go/internal/inquiry/infrastructure"
	"github.com/trvux/elc-go/internal/platform/db"
)

func main() {
	_ = godotenv.Load()

	appID := requireEnv("ZALO_OA_APP_ID")
	appSecret := requireEnv("ZALO_OA_APP_SECRET")
	redirectURI := requireEnv("ZALO_OA_REDIRECT_URI")

	verifier, challenge, err := generatePKCE()
	if err != nil {
		fatalf("generate PKCE pair: %v", err)
	}
	state := randomState()

	authURL := fmt.Sprintf(
		"https://oauth.zaloapp.com/v4/permission?app_id=%s&redirect_uri=%s&code_challenge=%s&state=%s",
		url.QueryEscape(appID), url.QueryEscape(redirectURI), url.QueryEscape(challenge), url.QueryEscape(state),
	)

	fmt.Println("1. Open this URL in a browser where you can approve the app on behalf of the OA:")
	fmt.Println()
	fmt.Println("   " + authURL)
	fmt.Println()
	fmt.Println("2. After approving, you'll be redirected to your ZALO_OA_REDIRECT_URI with a ?code=... param.")
	fmt.Println("   The page itself may 404 (that's fine) — just copy the code value from the address bar.")
	fmt.Println("   The code is only valid for 10 minutes, so do this promptly.")
	fmt.Println()
	fmt.Print("Paste the code here: ")

	reader := bufio.NewReader(os.Stdin)
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)
	if code == "" {
		fatalf("no code entered")
	}

	accessToken, refreshToken, expiresIn, err := exchangeCode(appID, appSecret, code, verifier)
	if err != nil {
		fatalf("exchange code for token: %v", err)
	}

	ctx := context.Background()
	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	tokenRepo := inquiryinfra.NewPostgresZaloTokenRepository(pool)
	token := domain.NewZaloOAToken(accessToken, refreshToken, time.Now().Add(time.Duration(expiresIn)*time.Second))
	if err := tokenRepo.Save(ctx, token); err != nil {
		fatalf("save token: %v", err)
	}

	fmt.Println()
	fmt.Println("Saved. The background refresher will keep this token fresh from now on.")
	fmt.Println("Next: have each staff member follow the OA in Zalo and send it one message —")
	fmt.Println("that's what lets the OA push a message back to them (see internal/inquiry/domain/zalo_follower.go).")
}

func generatePKCE() (verifier, challenge string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomState() string {
	raw := make([]byte, 16)
	_, _ = rand.Read(raw)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func exchangeCode(appID, appSecret, code, verifier string) (accessToken, refreshToken string, expiresIn int, err error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("app_id", appID)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", verifier)

	req, err := http.NewRequest(http.MethodPost, "https://oauth.zaloapp.com/v4/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", appSecret)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer res.Body.Close()

	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    string `json:"expires_in"`
		Error        int    `json:"error"`
		ErrorReason  string `json:"error_reason"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", "", 0, err
	}
	if parsed.AccessToken == "" {
		return "", "", 0, fmt.Errorf("zalo returned no access_token (error=%d reason=%s) — code may have expired (10 min lifetime), try again", parsed.Error, parsed.ErrorReason)
	}

	expiresInSeconds, err := strconv.Atoi(parsed.ExpiresIn)
	if err != nil {
		expiresInSeconds = 3600 // documented default if the field is ever missing/malformed
	}

	return parsed.AccessToken, parsed.RefreshToken, expiresInSeconds, nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("%s is required", key)
	}
	return v
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
