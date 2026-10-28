package compiler

import (
	"fmt"
 accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
 grpcaccesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/grpc/v3"
	"time"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	extauthz "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_authz/v3"
	localratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	tlsinspector "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	udp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/udp/udp_proxy/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"gateway-control/internal/model"
)

func httpListener(item model.PkgListener) (*listener.Listener, error) {
	codec := hcm.HttpConnectionManager_HTTP1
	var http2 *core.Http2ProtocolOptions
	var upgrades []*hcm.HttpConnectionManager_UpgradeConfig
	switch item.Protocol {
	case "http2", "grpc":
		codec = hcm.HttpConnectionManager_HTTP2
		http2 = &core.Http2ProtocolOptions{}
	case "https":
		codec = hcm.HttpConnectionManager_AUTO
		http2 = &core.Http2ProtocolOptions{}
		upgrades = websocketUpgrade()
	case "websocket", "http":
		upgrades = websocketUpgrade()
	}
	manager := &hcm.HttpConnectionManager{
		StatPrefix:                   "ingress",
        AccessLog:[]*accesslog.AccessLog{{
          Name:"envoy.access_loggers.http_grpc",
          ConfigType:&accesslog.AccessLog_TypedConfig{
            TypedConfig:mustAny(&grpcaccesslog.HttpGrpcAccessLogConfig{
              CommonConfig:&grpcaccesslog.CommonGrpcAccessLogConfig{
                LogName:"gateway-service-traffic",
                GrpcService:&core.GrpcService{
                  Timeout:durationpb.New(2*time.Second),
                  TargetSpecifier:&core.GrpcService_EnvoyGrpc_{
                    EnvoyGrpc:&core.GrpcService_EnvoyGrpc{ClusterName:"xds_cluster"},
                  },
                },
                TransportApiVersion:core.ApiVersion_V3,
                BufferFlushInterval:durationpb.New(time.Second),
              },
            }),
          },
        }},
		CodecType:                    codec,
		NormalizePath:                wrapperspb.Bool(true),
        UseRemoteAddress:wrapperspb.Bool(true),
        XffNumTrustedHops:2,
		MergeSlashes:                 true,
		PathWithEscapedSlashesAction: hcm.HttpConnectionManager_REJECT_REQUEST,
		CommonHttpProtocolOptions:    &core.HttpProtocolOptions{IdleTimeout: durationpb.New(300 * time.Second)},
		Http2ProtocolOptions:         http2,
		UpgradeConfigs:               upgrades,
		RequestHeadersTimeout:        durationpb.New(10 * time.Second),
		MaxRequestHeadersKb:          wrapperspb.UInt32(32),
		StreamIdleTimeout:            durationpb.New(300 * time.Second),
		StripPortMode:                &hcm.HttpConnectionManager_StripAnyHostPort{StripAnyHostPort: true},
		HttpFilters: []*hcm.HttpFilter{{
            Name: "envoy.filters.http.ext_authz",
            ConfigType: &hcm.HttpFilter_TypedConfig{
                TypedConfig: mustAny(&extauthz.ExtAuthz{
                    Services: &extauthz.ExtAuthz_GrpcService{
                        GrpcService: &core.GrpcService{
                            Timeout: durationpb.New(time.Second),
                            TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
                                EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: "xds_cluster"},
                            },
                        },
                    },
                    TransportApiVersion: core.ApiVersion_V3,
                    FailureModeAllow: false,
                }),
            },
        },{
            Name:"envoy.filters.http.local_ratelimit",
            ConfigType:&hcm.HttpFilter_TypedConfig{
              TypedConfig:mustAny(&localratelimit.LocalRateLimit{StatPrefix:"gateway_local_ratelimit"}),
            },
        },{
			Name: "envoy.filters.http.router",
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: mustAny(&router.Router{}),
			},
		}},
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{RouteConfigName: item.RouteConfigName, ConfigSource: adsSource()},
		},
	}
	chain := &listener.FilterChain{
		Filters: []*listener.Filter{{
			Name:       "envoy.filters.network.http_connection_manager",
			ConfigType: &listener.Filter_TypedConfig{TypedConfig: mustAny(manager)},
		}},
	}
	if item.Protocol == "https" {
		chain.TransportSocket = downstreamTLS(item.TlsSecret)
	}
	return tcpSocketListener(item, []*listener.FilterChain{chain}, nil), nil
}

func l4Listener(item model.PkgListener) (*listener.Listener, error) {
	if item.Cluster == "" {
		return nil, fmt.Errorf("%s listener kümesiz", item.Protocol)
	}
	switch item.Protocol {
	case "tcp":
		return tcpSocketListener(item, []*listener.FilterChain{tcpChain(item.Cluster, nil)}, nil), nil
	case "tls_passthrough":
		if len(item.ServerNames) == 0 {
			return nil, fmt.Errorf("tls passthrough için hostname yok")
		}
		return tcpSocketListener(item, []*listener.FilterChain{tcpChain(item.Cluster, item.ServerNames)}, []*listener.ListenerFilter{{
			Name: "envoy.filters.listener.tls_inspector",
			ConfigType: &listener.ListenerFilter_TypedConfig{
				TypedConfig: mustAny(&tlsinspector.TlsInspector{}),
			},
		}}), nil
	case "udp":
		return &listener.Listener{
			Name: item.Name,
			Address: &core.Address{Address: &core.Address_SocketAddress{SocketAddress: &core.SocketAddress{
				Protocol: core.SocketAddress_UDP, Address: "0.0.0.0",
				PortSpecifier: &core.SocketAddress_PortValue{PortValue: item.Port},
			}}},
			ListenerFilters: []*listener.ListenerFilter{{
				Name: "envoy.filters.udp_listener.udp_proxy",
				ConfigType: &listener.ListenerFilter_TypedConfig{TypedConfig: mustAny(&udp.UdpProxyConfig{
					StatPrefix:     "ingress",
					RouteSpecifier: &udp.UdpProxyConfig_Cluster{Cluster: item.Cluster},
					IdleTimeout:    durationpb.New(60 * time.Second),
				})},
			}},
		}, nil
	default:
		return nil, fmt.Errorf("desteklenmeyen l4 protokolü: %s", item.Protocol)
	}
}

func tcpChain(cluster string, serverNames []string) *listener.FilterChain {
	chain := &listener.FilterChain{
		Filters: []*listener.Filter{{
			Name: "envoy.filters.network.tcp_proxy",
			ConfigType: &listener.Filter_TypedConfig{TypedConfig: mustAny(&tcp.TcpProxy{
				StatPrefix:       "ingress",
				ClusterSpecifier: &tcp.TcpProxy_Cluster{Cluster: cluster},
			})},
		}},
	}
	if len(serverNames) > 0 {
		chain.FilterChainMatch = &listener.FilterChainMatch{ServerNames: serverNames}
	}
	return chain
}

func tcpSocketListener(item model.PkgListener, chains []*listener.FilterChain, filters []*listener.ListenerFilter) *listener.Listener {
	return &listener.Listener{
		Name: item.Name,
		Address: &core.Address{Address: &core.Address_SocketAddress{SocketAddress: &core.SocketAddress{
			Protocol: core.SocketAddress_TCP, Address: "0.0.0.0",
			PortSpecifier: &core.SocketAddress_PortValue{PortValue: item.Port},
		}}},
		ListenerFilters: filters,
		FilterChains:    chains,
	}
}

func websocketUpgrade() []*hcm.HttpConnectionManager_UpgradeConfig {
	return []*hcm.HttpConnectionManager_UpgradeConfig{{UpgradeType: "websocket"}}
}

func downstreamTLS(name string) *core.TransportSocket {
	return &core.TransportSocket{
		Name: "envoy.transport_sockets.tls",
		ConfigType: &core.TransportSocket_TypedConfig{TypedConfig: mustAny(&tlsv3.DownstreamTlsContext{
			CommonTlsContext: &tlsv3.CommonTlsContext{
				AlpnProtocols: []string{"h2", "http/1.1"},
				TlsParams: &tlsv3.TlsParameters{
					TlsMinimumProtocolVersion: tlsv3.TlsParameters_TLSv1_2,
					TlsMaximumProtocolVersion: tlsv3.TlsParameters_TLSv1_3,
				},
				TlsCertificateSdsSecretConfigs: []*tlsv3.SdsSecretConfig{{
					Name: name, SdsConfig: adsSource(),
				}},
			},
		})},
	}
}

func tlsSecret(name string, cert, key []byte) (*tlsv3.Secret, error) {
	if name == "" || len(cert) == 0 || len(key) == 0 {
		return nil, fmt.Errorf("https için yerel sertifika yok")
	}
	return &tlsv3.Secret{
		Name: name,
		Type: &tlsv3.Secret_TlsCertificate{TlsCertificate: &tlsv3.TlsCertificate{
			CertificateChain: &core.DataSource{Specifier: &core.DataSource_InlineBytes{InlineBytes: cert}},
			PrivateKey:       &core.DataSource{Specifier: &core.DataSource_InlineBytes{InlineBytes: key}},
		}},
	}, nil
}

func healthChecks(item model.PkgCluster) []*core.HealthCheck {
	base := &core.HealthCheck{
		Timeout: durationpb.New(time.Second), Interval: durationpb.New(2 * time.Second),
		UnhealthyThreshold: wrapperspb.UInt32(2), HealthyThreshold: wrapperspb.UInt32(1),
	}
	switch item.Health {
	case "none":
		return nil
	case "tcp":
		base.HealthChecker = &core.HealthCheck_TcpHealthCheck_{TcpHealthCheck: &core.HealthCheck_TcpHealthCheck{}}
	default:
		base.HealthChecker = &core.HealthCheck_HttpHealthCheck_{HttpHealthCheck: &core.HealthCheck_HttpHealthCheck{Path: "/healthz"}}
	}
	return []*core.HealthCheck{base}
}

func http2Upstream(item model.PkgCluster) *core.Http2ProtocolOptions {
	if !item.UpstreamHTTP2 {
		return nil
	}
	return &core.Http2ProtocolOptions{}
}
