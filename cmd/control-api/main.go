package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"gateway-control/internal/httpapi"
	"gateway-control/internal/sign"
	"gateway-control/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dsn := getenv("DATABASE_URL", "")
	if dsn == "" {
		logger.Error("DATABASE_URL zorunlu")
		os.Exit(1)
	}
	if err := store.Migrate(dsn); err != nil {
		logger.Error("migration başarısız", "err", err)
		os.Exit(1)
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		logger.Error("veritabanı bağlantısı kurulamadı", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	signer, err := sign.LoadPrivateKey(getenv("GATEWAY_SIGNING_KEY_FILE", ""))
	if err != nil {
		logger.Error("imza anahtarı okunamadı", "err", err)
		os.Exit(1)
	}
	tokens, err := httpapi.LoadTokens(
		getenv("GATEWAY_OPERATOR_TOKEN_FILE", ""),
		getenv("GATEWAY_VIEWER_TOKEN_FILE", ""),
		getenv("GATEWAY_AGENT_TOKEN_FILE", ""),
	)
	if err != nil {
		logger.Error("kimlik dosyası okunamadı", "err", err)
		os.Exit(1)
	}
	server := &http.Server{
		Addr:              getenv("LISTEN_ADDR", ":8080"),
		Handler:           httpapi.New(st, signer, tokens).WithTryURL(getenv("GATEWAY_TRY_URL", "")).WithLabTargets(getenv("GATEWAY_LAB_TARGETS", "")).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	logger.Info("yönetim API dinliyor", "addr", server.Addr, "keyId", signer.KeyID())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("API durdu", "err", err)
		os.Exit(1)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
