package compiler

import (
	"fmt"
	"strconv"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"gateway-control/internal/model"
)

func Build(pkg model.Package, tlsCert, tlsKey []byte) (*cache.Snapshot, error) {
	if err := check(pkg); err != nil {
		return nil, err
	}
	version := strconv.FormatInt(pkg.Revision, 10)
	listeners := make([]types.Resource, 0, len(pkg.Listeners))
	routes := make([]types.Resource, 0, len(pkg.Listeners))
	clusters := make([]types.Resource, 0, len(pkg.Clusters))
	var secrets []types.Resource
	for _, item := range pkg.Listeners {
		switch item.Protocol {
		case "http", "https", "http2", "websocket", "sse", "grpc":
			built, err := httpListener(item)
			if err != nil {
				return nil, err
			}
			listeners = append(listeners, built)
			routes = append(routes, &route.RouteConfiguration{
				Name:         item.RouteConfigName,
				VirtualHosts: virtualHosts(pkg.VirtualHosts, item.Name),
			})
			if item.Protocol == "https" {
				secret, err := tlsSecret(item.TlsSecret, tlsCert, tlsKey)
				if err != nil {
					return nil, err
				}
				secrets = append(secrets, secret)
			}
		case "tcp", "udp", "tls_passthrough":
			built, err := l4Listener(item)
			if err != nil {
				return nil, err
			}
			listeners = append(listeners, built)
		default:
			return nil, fmt.Errorf("desteklenmeyen listener protokolü: %s", item.Protocol)
		}
	}
	for _, item := range pkg.Clusters {
		clusters = append(clusters, clusterConfig(item))
	}
	resources := map[resource.Type][]types.Resource{
		resource.ListenerType: listeners,
		resource.RouteType:    routes,
		resource.ClusterType:  clusters,
	}
	if len(secrets) > 0 {
		resources[resource.SecretType] = secrets
	}
	snapshot, err := cache.NewSnapshot(version, resources)
	if err != nil {
		return nil, err
	}
	if err := snapshot.Consistent(); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func check(pkg model.Package) error {
	for _, host := range pkg.VirtualHosts {
		if len(host.Domains) == 0 {
			return fmt.Errorf("virtual host domainsiz")
		}
		for _, domain := range host.Domains {
			if domain == "" || domain == "*" || domain[0] == '*' {
				return fmt.Errorf("joker domain kapalı")
			}
		}
	}
	if len(pkg.Listeners) == 0 && (len(pkg.VirtualHosts) > 0 || len(pkg.Clusters) > 0) {
		return fmt.Errorf("listener olmadan rota üretilemez")
	}
	return nil
}

func virtualHosts(hosts []model.PkgVHost, listenerName string) []*route.VirtualHost {
	out := make([]*route.VirtualHost, 0, len(hosts))
	for _, host := range hosts {
		if host.Listener != "" && host.Listener != listenerName {
			continue
		}
		routes := make([]*route.Route, 0, len(host.Routes))
		for _, item := range host.Routes {
			methods := item.Methods
			if len(methods) == 0 {
				methods = []string{""}
			}
			for _, method := range methods {
				match := &route.RouteMatch{}
				if item.MatchType == model.MatchExact {
					match.PathSpecifier = &route.RouteMatch_Path{Path: item.Path}
				} else {
					match.PathSpecifier = &route.RouteMatch_Prefix{Prefix: item.Path}
				}
				if method != "" {
					match.Headers = []*route.HeaderMatcher{{
						Name: ":method",
						HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
							StringMatch: &matcher.StringMatcher{
								MatchPattern: &matcher.StringMatcher_Exact{Exact: method},
							},
						},
					}}
				}
				if item.Grpc {
					match.Headers = append(match.Headers, &route.HeaderMatcher{
						Name: "content-type",
						HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
							StringMatch: &matcher.StringMatcher{
								MatchPattern: &matcher.StringMatcher_Prefix{Prefix: "application/grpc"},
							},
						},
					})
				}
				name := item.Name
				if method != "" {
					name += "-" + method
				}
				action := &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{Cluster: item.Cluster},
					Timeout:          durationpb.New(time.Duration(item.TimeoutMs) * time.Millisecond),
				}
				if host.Protocol == "http" || host.Protocol == "https" || host.Protocol == "websocket" {
					action.UpgradeConfigs = []*route.RouteAction_UpgradeConfig{{UpgradeType: "websocket"}}
				}
				routes = append(routes, &route.Route{
					Name:   name,
					Match:  match,
					Action: &route.Route_Route{Route: action},
				})
			}
		}
		out = append(out, &route.VirtualHost{
			Name:    host.Name,
			Domains: append([]string(nil), host.Domains...),
			Routes:  routes,
		})
	}
	return out
}

func clusterConfig(item model.PkgCluster) *cluster.Cluster {
	policy := cluster.Cluster_ROUND_ROBIN
	if item.Algorithm == model.AlgoLeastRequest {
		policy = cluster.Cluster_LEAST_REQUEST
	}
	protocol := core.SocketAddress_TCP
	if item.Health == "none" {
		protocol = core.SocketAddress_UDP
	}
	endpoints := make([]*endpoint.LbEndpoint, 0, len(item.Endpoints))
	for _, ep := range item.Endpoints {
		endpoints = append(endpoints, &endpoint.LbEndpoint{
			LoadBalancingWeight: wrapperspb.UInt32(ep.Weight),
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{Endpoint: &endpoint.Endpoint{
				Address: &core.Address{Address: &core.Address_SocketAddress{SocketAddress: &core.SocketAddress{
					Protocol:      protocol,
					Address:       ep.Address,
					PortSpecifier: &core.SocketAddress_PortValue{PortValue: ep.Port},
				}}},
			}},
		})
	}
	return &cluster.Cluster{
		Name:                 item.Name,
		ConnectTimeout:       durationpb.New(time.Duration(item.ConnectTimeoutMs) * time.Millisecond),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
		DnsLookupFamily:      cluster.Cluster_V4_PREFERRED,
		LbPolicy:             policy,
		LoadAssignment: &endpoint.ClusterLoadAssignment{
			ClusterName: item.Name,
			Endpoints: []*endpoint.LocalityLbEndpoints{{
				LbEndpoints: endpoints,
			}},
		},
		HealthChecks:         healthChecks(item),
		Http2ProtocolOptions: http2Upstream(item),
		CircuitBreakers: &cluster.CircuitBreakers{
			Thresholds: []*cluster.CircuitBreakers_Thresholds{{
				Priority:           core.RoutingPriority_DEFAULT,
				MaxConnections:     wrapperspb.UInt32(1024),
				MaxPendingRequests: wrapperspb.UInt32(1024),
				MaxRequests:        wrapperspb.UInt32(1024),
				MaxRetries:         wrapperspb.UInt32(0),
			}},
		},
	}
}

func adsSource() *core.ConfigSource {
	return &core.ConfigSource{
		ResourceApiVersion: core.ApiVersion_V3,
		ConfigSourceSpecifier: &core.ConfigSource_Ads{
			Ads: &core.AggregatedConfigSource{},
		},
	}
}

func mustAny(message proto.Message) *anypb.Any {
	value, err := anypb.New(message)
	if err != nil {
		panic(err)
	}
	return value
}
