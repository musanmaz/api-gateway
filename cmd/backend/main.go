package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	name := getenv("INSTANCE", "backend")
	port := getenv("PORT", "8080")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"instance": name, "service": getenv("SERVICE_NAME", name),
			"environment": "test", "hostname": r.Host, "path": r.URL.Path,
			"method": r.Method, "timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprintf(w, "data: instance=%s\n\n", name)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
	})
	mux.Handle("/ws", websocket.Server{
		Handshake: func(*websocket.Config, *http.Request) error { return nil },
		Handler: func(conn *websocket.Conn) {
			fmt.Fprintf(conn, "instance=%s path=/ws\n", name)
		},
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "instance=%s path=%s\n", name, r.URL.Path)
	})
	go serveGRPC(getenv("GRPC_ADDR", ""))
	go serveUDP(getenv("UDP_ADDR", ""))
	go serveTLS(getenv("TLS_ADDR", ""), getenv("TLS_CERT_FILE", ""), getenv("TLS_KEY_FILE", ""), name)
	log.Printf("backend %s listening on %s", name, port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func serveGRPC(addr string) {
	if addr == "" {
		return
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("grpc dinlenemedi: %v", err)
	}
	server := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	log.Printf("grpc listening on %s", addr)
	log.Fatal(server.Serve(lis))
}

func serveUDP(addr string) {
	if addr == "" {
		return
	}
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Fatalf("udp dinlenemedi: %v", err)
	}
	defer conn.Close()
	log.Printf("udp listening on %s", addr)
	buf := make([]byte, 1024)
	for {
		_, peer, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}
		_, _ = conn.WriteTo([]byte("pong"), peer)
	}
}

func serveTLS(addr, certFile, keyFile, name string) {
	if addr == "" || certFile == "" || keyFile == "" {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "passthrough instance=%s path=%s\n", name, r.URL.Path)
	})
	server := &http.Server{Addr: addr, Handler: mux, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	log.Printf("tls backend listening on %s", addr)
	log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
