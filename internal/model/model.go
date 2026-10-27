package model

import "time"

const (
	RoleOperator = "operator"
	RoleViewer   = "viewer"
	RoleAgent    = "agent"

	StatusDraft  = "draft"
	StatusActive = "active"

	MatchPrefix = "prefix"
	MatchExact  = "exact"

	AlgoRoundRobin   = "round_robin"
	AlgoLeastRequest = "least_request"

	ImportanceLow    = "low"
	ImportanceMedium = "medium"
	ImportanceHigh   = "high"
)

type ServiceInput struct {
	Name            string       `json:"name"`
	Description     string       `json:"description"`
	Owner           string       `json:"owner"`
	Importance      string       `json:"importance"`
	EnvironmentID   string       `json:"environmentId"`
	TeamID          string       `json:"teamId"`
	FleetID         string       `json:"fleetId"`
	ListenerID      string       `json:"listenerId,omitempty"`
	Hostnames       []string     `json:"hostnames"`
	Routes          []RouteInput `json:"routes"`
	ExpectedVersion int          `json:"expectedVersion,omitempty"`
}

type RouteInput struct {
	Path      string        `json:"path"`
	MatchType string        `json:"matchType"`
	Methods   []string      `json:"methods"`
	Upstream  UpstreamInput `json:"upstream"`
}

type UpstreamInput struct {
	Name             string          `json:"name"`
	Algorithm        string          `json:"algorithm"`
	ConnectTimeoutMs int             `json:"connectTimeoutMs"`
	RequestTimeoutMs int             `json:"requestTimeoutMs"`
	Endpoints        []EndpointInput `json:"endpoints"`
}

type EndpointInput struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
	Weight  int    `json:"weight"`
}

type Service struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Owner            string    `json:"owner"`
	Importance       string    `json:"importance"`
	EnvironmentID    string    `json:"environmentId"`
	TeamID           string    `json:"teamId"`
	FleetID          string    `json:"fleetId"`
	ListenerID       string    `json:"listenerId"`
	Status           string    `json:"status"`
	Version          int       `json:"version"`
	PublishedVersion int       `json:"publishedVersion"`
	Hostnames        []string  `json:"hostnames"`
	Routes           []Route   `json:"routes"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type Route struct {
	ID        string   `json:"id"`
	Path      string   `json:"path"`
	MatchType string   `json:"matchType"`
	Methods   []string `json:"methods"`
	Upstream  Upstream `json:"upstream"`
}

type Upstream struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Algorithm        string     `json:"algorithm"`
	ConnectTimeoutMs int        `json:"connectTimeoutMs"`
	RequestTimeoutMs int        `json:"requestTimeoutMs"`
	Endpoints        []Endpoint `json:"endpoints"`
}

type Endpoint struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Port    int    `json:"port"`
	Weight  int    `json:"weight"`
}

type Listener struct {
	ID       string `json:"id"`
	FleetID  string `json:"fleetId"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}

type Node struct {
	ID                string     `json:"id"`
	FleetID           string     `json:"fleetId"`
	Name              string     `json:"name"`
	Location          string     `json:"location"`
	DesiredRevision   *int64     `json:"desiredRevision"`
	DeliveredRevision *int64     `json:"deliveredRevision"`
	AckedRevision     *int64     `json:"ackedRevision"`
	NackRevision      *int64     `json:"nackRevision"`
	NackDetail        string     `json:"nackDetail"`
	LastHeartbeatAt   *time.Time `json:"lastHeartbeatAt"`
	Heartbeat         string     `json:"heartbeat"`
	TrafficHealth     string     `json:"trafficHealth"`
	Probes            []Probe    `json:"probes"`
}

type Probe struct {
	Host      string `json:"host"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	Body      string `json:"body"`
	CheckedAt string `json:"checkedAt"`
}

type Revision struct {
	ID               int64     `json:"id"`
	FleetID          string    `json:"fleetId"`
	ParentRevisionID *int64    `json:"parentRevisionId"`
	ContentHash      string    `json:"contentHash"`
	KeyID            string    `json:"keyId"`
	Summary          string    `json:"summary"`
	CreatedBy        string    `json:"createdBy"`
	CreatedAt        time.Time `json:"createdAt"`
	Payload          string    `json:"payload,omitempty"`
	Signature        string    `json:"signature,omitempty"`
}

type AuditEvent struct {
	ID        int64     `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Before    any       `json:"before"`
	After     any       `json:"after"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"createdAt"`
}

type Package struct {
	Revision     int64         `json:"revision"`
	FleetID      string        `json:"fleetId"`
	IssuedAt     string        `json:"issuedAt"`
	Listeners    []PkgListener `json:"listeners"`
	VirtualHosts []PkgVHost    `json:"virtualHosts"`
	Clusters     []PkgCluster  `json:"clusters"`
}

type PkgListener struct {
	Name            string   `json:"name"`
	Protocol        string   `json:"protocol"`
	Port            uint32   `json:"port"`
	RouteConfigName string   `json:"routeConfigName"`
	TlsSecret       string   `json:"tlsSecret,omitempty"`
	ServerNames     []string `json:"serverNames,omitempty"`
	Cluster         string   `json:"cluster,omitempty"`
}

type PkgVHost struct {
	Name     string     `json:"name"`
	Listener string     `json:"listener,omitempty"`
	Protocol string     `json:"protocol,omitempty"`
	Domains  []string   `json:"domains"`
	Routes   []PkgRoute `json:"routes"`
}

type PkgRoute struct {
	Name      string   `json:"name"`
	MatchType string   `json:"matchType"`
	Path      string   `json:"path"`
	Methods   []string `json:"methods"`
	Cluster   string   `json:"cluster"`
	Priority  int      `json:"priority"`
	TimeoutMs int      `json:"timeoutMs"`
	Grpc      bool     `json:"grpc,omitempty"`
}

type PkgCluster struct {
	Name             string        `json:"name"`
	Algorithm        string        `json:"algorithm"`
	ConnectTimeoutMs int           `json:"connectTimeoutMs"`
	Endpoints        []PkgEndpoint `json:"endpoints"`
	UpstreamHTTP2    bool          `json:"upstreamHttp2,omitempty"`
	Health           string        `json:"health,omitempty"`
}

type PkgEndpoint struct {
	Address string `json:"address"`
	Port    uint32 `json:"port"`
	Weight  uint32 `json:"weight"`
}

type DiffChange struct {
	Op     string `json:"op"`
	Host   string `json:"host"`
	Path   string `json:"path"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type Envelope struct {
	Payload     []byte `json:"payload"`
	ContentHash string `json:"contentHash"`
	Signature   string `json:"signature"`
	KeyID       string `json:"keyId"`
	Revision    int64  `json:"revision"`
}
