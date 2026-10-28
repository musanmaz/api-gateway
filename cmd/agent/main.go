package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	ldsv3 "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtime "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secret "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"

	"gateway-control/internal/agentdisk"
	"gateway-control/internal/compiler"
	"gateway-control/internal/model"
	"gateway-control/internal/sign"
	"gateway-control/internal/secretbox"
 accesslogs "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
 "github.com/corazawaf/coraza/v3"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	nodeName := getenv("NODE_NAME", "gw-local-1")
	public, _, err := sign.LoadPublicKey(getenv("PUBKEY_FILE", ""))
	if err != nil {
		logger.Error("açık anahtar okunamadı", "err", err)
		os.Exit(1)
	}
	tokenRaw, err := os.ReadFile(getenv("AGENT_TOKEN_FILE", ""))
	if err != nil {
		logger.Error("agent token okunamadı", "err", err)
		os.Exit(1)
	}
	disk, err := agentdisk.Open(getenv("STATE_DIR", "/var/lib/gateway-agent"), public)
	if err != nil {
		logger.Error("durum dizini açılamadı", "err", err)
		os.Exit(1)
	}
	state := &nodeState{node: nodeName}
	snapshots := cache.NewSnapshotCache(true, cache.IDHash{}, xdsLogger{})
	server := xds.NewServer(context.Background(), snapshots, xds.CallbackFuncs{
		StreamRequestFunc: func(_ int64, req *discovery.DiscoveryRequest) error {
			if req.GetErrorDetail() != nil {
				detail := model.NackDetail(req.GetErrorDetail().GetMessage())
				logger.Warn("envoy konfigürasyonu reddetti", "detail", detail)
				restore, rejected := state.noteNack(detail)
				if !rejected {
					return nil
				}
				if restore != nil {
					if err := snapshots.SetSnapshot(context.Background(), nodeName, restore); err != nil {
						logger.Error("son paket geri konamadı", "err", err)
					}
					return nil
				}
				snapshots.ClearSnapshot(nodeName)
				return nil
			}
			if err := state.observeAck(req.GetTypeUrl(), req.GetVersionInfo(), disk.Save); err != nil {
				logger.Error("ack paketi diske yazılamadı", "err", err)
			}
			return nil
		},
	})
	if _, pkg, err := disk.Load(); err != nil {
		logger.Warn("güvenilir yerel paket yok", "err", err)
	} else if snapshot, err := stage(snapshots, nodeName, pkg, logger); err != nil {
		logger.Error("yerel paket uygulanamadı", "err", err)
	} else {
		state.rememberGood(pkg, snapshot)
		logger.Info("yerel paket yüklendi", "revision", pkg.Revision)
	}
    traffic:=newTrafficCollector(state)
    waf, err:=createCorazaWAF()
    if err!=nil{logger.Error("Coraza WAF could not initialize","err",err);os.Exit(1)}
	credentialKey, err := secretbox.ReadKey(getenv("GATEWAY_CREDENTIAL_KEY_FILE", ""))
	if err != nil {logger.Error("credential encryption key unavailable", "err", err); os.Exit(1)}
	go serveXDS(logger, getenv("XDS_SOCKET", "/var/run/gateway/xds.sock"), server, state, credentialKey,waf,traffic)

	client := &http.Client{Timeout: 5 * time.Second}
	probeClient := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	control := strings.TrimRight(getenv("CONTROL_URL", "http://api:8080"), "/")
	dataPlane := strings.TrimRight(getenv("DATA_PLANE_URL", "http://envoy:10000"), "/")
	bearer := strings.TrimSpace(string(tokenRaw))
	failures := 0
	run := func() {
		if err := poll(client, control, bearer, public, disk, snapshots, state, logger); err != nil {
			failures++
			if failures == 1 || failures%10 == 0 {
				logger.Warn("kontrol katmanına ulaşılamadı", "err", err, "failures", failures)
			}
		} else {
			failures = 0
		}
		heartbeat(client, control, bearer, dataPlane, probeClient, state, traffic, logger)
	}
	run()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for range tick.C {
		run()
	}
}

func poll(client *http.Client, control, token string, public ed25519.PublicKey, disk *agentdisk.Disk, snapshots cache.SnapshotCache, state *nodeState, logger *slog.Logger) error {
	req, err := http.NewRequest(http.MethodGet, control+"/api/v1/agent/package?node="+state.node, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("paket yanıtı %d", resp.StatusCode)
	}
	var env model.Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return err
	}
	if err := sign.Verify(public, env); err != nil {
		logger.Warn("paket doğrulanamadı", "err", err)
		return nil
	}
	var pkg model.Package
	if err := json.Unmarshal(env.Payload, &pkg); err != nil {
		return err
	}
	if pkg.Revision != env.Revision {
		logger.Warn("paket sürümü uyuşmuyor")
		return nil
	}
	if state.shouldSkip(pkg.Revision) {
		return nil
	}
	snapshot, err := stage(snapshots, state.node, pkg, logger)
	if err != nil {
		logger.Error("paket envoy önbelleğine yazılamadı", "err", err, "revision", pkg.Revision)
		return nil
	}
	state.begin(env, pkg, snapshot)
	logger.Info("paket envoy önbelleğine kondu", "revision", pkg.Revision)
	return nil
}

func stage(snapshots cache.SnapshotCache, nodeName string, pkg model.Package, logger *slog.Logger) (*cache.Snapshot, error) {
	cert, key := localCertificate(pkg, logger)
	snapshot, err := compiler.Build(pkg, cert, key)
	if err != nil {
		return nil, err
	}
	if err := snapshots.SetSnapshot(context.Background(), nodeName, snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func localCertificate(pkg model.Package, logger *slog.Logger) ([]byte, []byte) {
	for _, item := range pkg.Listeners {
		if item.Protocol != "https" {
			continue
		}
		cert, certErr := os.ReadFile(getenv("TLS_CERT_FILE", "/etc/gateway/certs/server.crt"))
		key, keyErr := os.ReadFile(getenv("TLS_KEY_FILE", "/etc/gateway/certs/server.key"))
		if certErr != nil || keyErr != nil {
			logger.Error("yerel tls sertifikası okunamadı", "cert", certErr, "key", keyErr)
			return nil, nil
		}
		return cert, key
	}
	return nil, nil
}

func heartbeat(client *http.Client, control, token, dataPlane string, probeClient *http.Client, state *nodeState, traffic *trafficCollector, logger *slog.Logger) {
	report := state.beat()
	body := map[string]any{
		"node":              state.node,
		"deliveredRevision": report.Delivered,
		"ackedRevision":     report.Acked,
		"nackRevision":      report.Nack,
		"nackDetail":        report.Detail,
		"clearNack":         report.Clear,
		"probes":            probeRoutes(probeClient, dataPlane, report.Pkg),
        "trafficBootId":traffic.bootID,
        "traffic":traffic.snapshot(),
        "telemetry":func() model.NodeTelemetry {v:=readEnvoyStats(getenv("ENVOY_ADMIN_SOCKET","/var/run/gateway/admin.sock"));v.Checks=state.authChecks.Load();v.Denied=state.authDenied.Load();v.WafMatches=state.wafMatches.Load();v.WafBlocked=state.wafBlocked.Load();return v}(),
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, control+"/api/v1/agent/heartbeat", strings.NewReader(string(raw)))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		logger.Debug("heartbeat yanıtı", "status", resp.StatusCode)
		return
	}
	if report.Clear {
		state.nackCleared()
	}
}

func probeRoutes(client *http.Client, base string, pkg *model.Package) []model.Probe {
	if pkg == nil {
		return []model.Probe{}
	}
	out := []model.Probe{}
	for _, host := range pkg.VirtualHosts {
		for _, domain := range host.Domains {
			if host.Protocol != "" && host.Protocol != "http" {
				continue
			}
			for _, item := range host.Routes {
				if len(out) >= 20 {
					return out
				}
				method := http.MethodGet
				if len(item.Methods) > 0 {
					method = item.Methods[0]
				}
				req, err := http.NewRequest(method, base+item.Path, nil)
				if err != nil {
					continue
				}
				req.Host = domain
				checked := time.Now().UTC().Format(time.RFC3339)
				resp, err := client.Do(req)
				if err != nil {
					out = append(out, model.Probe{Host: domain, Path: item.Path, Status: 0, Body: "ulaşılamadı", CheckedAt: checked})
					continue
				}
				payload, _ := io.ReadAll(io.LimitReader(resp.Body, 180))
				resp.Body.Close()
				out = append(out, model.Probe{
					Host: domain, Path: item.Path, Status: resp.StatusCode,
					Body: oneLine(payload), CheckedAt: checked,
				})
			}
		}
	}
	return out
}

func serveXDS(logger *slog.Logger, socketPath string, server xds.Server, state *nodeState, key []byte,waf coraza.WAF,traffic *trafficCollector) {
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		logger.Error("eski xds soketi silinemedi", "err", err)
		os.Exit(1)
	}
	unixListener, err := net.Listen("unix", socketPath)
	if err != nil {
		logger.Error("xds soketi dinlenemedi", "err", err)
		os.Exit(1)
	}
	if err := os.Chmod(socketPath, 0o660); err != nil {
		logger.Error("xds soket izni ayarlanamadı", "err", err)
		os.Exit(1)
	}
	grpcServer := grpc.NewServer()
    accesslogs.RegisterAccessLogServiceServer(grpcServer,traffic)
	auth.RegisterAuthorizationServer(grpcServer, &localAuthorizer{state: state, encryptionKey: key,waf:waf,traffic:traffic})
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpoint.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	cluster.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	route.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	ldsv3.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	secret.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	runtime.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
	logger.Info("xds dinliyor", "socket", socketPath)
	if err := grpcServer.Serve(unixListener); err != nil {
		logger.Error("xds durdu", "err", err)
		os.Exit(1)
	}
}

type xdsLogger struct{}

func (xdsLogger) Debugf(string, ...any) {}
func (xdsLogger) Infof(string, ...any)  {}
func (xdsLogger) Warnf(format string, args ...any) {
	slog.Warn(fmt.Sprintf(format, args...))
}
func (xdsLogger) Errorf(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
}

func oneLine(raw []byte) string {
	text := strings.Join(strings.Fields(string(bytesTrim(raw))), " ")
	if len(text) > 180 {
		return text[:180]
	}
	return text
}

func bytesTrim(raw []byte) []byte {
	return bytesTrimSpace(raw)
}

func bytesTrimSpace(raw []byte) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
