package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	userv1 "github.com/JustUzair/go-grpc-irctc-backend/gen/go/user/v1"
	env "github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	redirectAddress = "127.0.0.1:8765"
	redirectURI     = "http://127.0.0.1:8765/oauth/callback"
	googleAuthURL   = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL  = "https://oauth2.googleapis.com/token"
)

type callbackResult struct {
	code  string
	state string
	err   string
}

type tokenResponse struct {
	IDToken string `json:"id_token"`
}

func main() {
	config, err := env.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	if config.GoogleClientID == "" || config.GoogleClientSecret == "" {
		log.Fatal("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are required")
	}
	if config.UserServicePort == "" {
		log.Fatal("USER_SERVICE_PORT is required")
	}

	state, err := randomToken()
	if err != nil {
		log.Fatalf("generate OAuth state: %v", err)
	}

	callback := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		result := callbackResult{
			code:  r.URL.Query().Get("code"),
			state: r.URL.Query().Get("state"),
			err:   r.URL.Query().Get("error"),
		}
		callback <- result
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<h2>Google sign-in received.</h2><p>You can return to the terminal.</p>")
		go func() { _ = server.Shutdown(context.Background()) }()
	})

	listener, err := net.Listen("tcp", redirectAddress)
	if err != nil {
		log.Fatalf("listen on %s: %v", redirectAddress, err)
	}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("OAuth callback server: %v", err)
		}
	}()

	authURL := buildAuthorizationURL(config.GoogleClientID, state)
	fmt.Println("Open this URL if your browser did not open automatically:")
	fmt.Println(authURL)
	openBrowser(authURL)

	result := <-callback
	if result.err != "" {
		log.Fatalf("Google authorization failed: %s", result.err)
	}
	if result.state != state {
		log.Fatal("OAuth state mismatch")
	}
	if result.code == "" {
		log.Fatal("Google callback did not include an authorization code")
	}

	idToken, err := exchangeCode(context.Background(), config.GoogleClientID, config.GoogleClientSecret, result.code)
	if err != nil {
		log.Fatalf("exchange authorization code: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.NewClient(
		"127.0.0.1:"+config.UserServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("connect to user service: %v", err)
	}
	defer conn.Close()

	client := userv1.NewUserServiceClient(conn)
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"user-agent", "google-oauth-test-client/1.0",
		"x-forwarded-for", "127.0.0.1",
		"accept", "application/grpc",
	))
	response, err := client.VerifyGoogleIDToken(ctx, &userv1.VerifyGoogleIDTokenRequest{IdToken: idToken})
	if err != nil {
		log.Fatalf("verify Google ID token through gRPC: %v", err)
	}

	fmt.Println("Google OAuth → gRPC verification succeeded")
	fmt.Printf("user: %s %s <%s>\n", response.User.FirstName, response.User.LastName, response.User.Email)
	fmt.Printf("email verified: %t\n", response.User.EmailVerified)
	fmt.Printf("access token expires in: %d seconds\n", response.AccessTokenExpiresIn)
	fmt.Printf("refresh token expires in: %d seconds\n", response.RefreshTokenExpiresIn)
}

func buildAuthorizationURL(clientID, state string) string {
	query := url.Values{
		"client_id":     {clientID},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"redirect_uri":  {redirectURI},
		"state":         {state},
	}
	return googleAuthURL + "?" + query.Encode()
}

func exchangeCode(ctx context.Context, clientID, clientSecret, code string) (string, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send token request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("Google token endpoint returned %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if token.IDToken == "" {
		return "", errors.New("Google token response did not contain an ID token")
	}
	return token.IDToken, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(target string) {
	var command string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
	case "linux":
		command = "xdg-open"
	case "windows":
		command = "rundll32"
		if err := exec.Command(command, "url.dll,FileProtocolHandler", target).Start(); err == nil {
			return
		}
	}
	if command != "" {
		_ = exec.Command(command, target).Start()
	}
}
