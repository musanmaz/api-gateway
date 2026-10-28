export type Protocol = { id: string; state: "available" | "unavailable"; reason?: string };

export type GatewayListener = { id: string; name: string; protocol: string; port: number };

export type Bootstrap = {
  environmentId: string;
  environment: string;
  teamId: string;
  team: string;
  fleetId: string;
  fleet: string;
  listener: GatewayListener;
  listeners: GatewayListener[];
  protocols: Protocol[];
  tryUrl: string;
};

export type Probe = { host: string; path: string; status: number; body: string; checkedAt: string };

export type NodeInfo = {
  telemetry?: {checks:number;denied:number;httpRequests:number;http2xx:number;http4xx:number;http5xx:number;wafMatches:number;wafBlocked:number};
  id: string;
  name: string;
  location: string;
  desiredRevision: number | null;
  deliveredRevision: number | null;
  ackedRevision: number | null;
  nackRevision: number | null;
  nackDetail: string;
  lastHeartbeatAt: string | null;
  heartbeat: "fresh" | "stale" | "unknown";
  trafficHealth: "unknown";
  probes: Probe[];
};

export type Overview = {
  services: { total: number; active: number; draft: number };
  nodes: NodeInfo[];
  telemetry: string;
  externalProbe: string;
  pendingDrafts: number;
  activeRevision: number | null;
};

export type Endpoint = { address: string; port: number; weight: number };
export type Upstream = {
  name: string;
  algorithm: string;
  connectTimeoutMs: number;
  requestTimeoutMs: number;
  endpoints: Endpoint[];
};
export type Route = { id?: string; rateLimitRps?: number; path: string; matchType: string; methods: string[]; upstream: Upstream };
export type Service = {
  wafMode?: "off"|"detect"|"block";
  ipAllowlist?: string[];
  ipDenylist?: string[];
  rateLimitRps?: number;
  upstreamAuthMode?: "none" | "basic" | "bearer" | "header";
  upstreamAuthUsername?: string;
  upstreamAuthHeader?: string;
  requiredClientHeader?: string;
  upstreamAuthConfigured?: boolean;
  authMode?: "public" | "api_key";
  gatewayBaseUrl?: string;
  id: string;
  name: string;
  listenerId: string;
  description: string;
  owner: string;
  importance: string;
  status: string;
  version: number;
  publishedVersion: number;
  hostnames: string[];
  routes: Route[];
  updatedAt: string;
};

export type DiffChange = { op: "add" | "remove" | "change"; host: string; path: string; before?: string; after?: string };

export type Revision = {
  id: number;
  contentHash: string;
  summary: string;
  createdBy: string;
  createdAt: string;
  payload?: string;
  parentRevisionId: number | null;
};

export type AuditEvent = {
  id: number;
  actor: string;
  action: string;
  target: string;
  before: unknown;
  after: unknown;
  result: string;
  createdAt: string;
};

export type LabTarget = { host: string; expectedInstance: string; url: string };
export type LabResult = LabTarget & { status: number; latencyMs: number; ok: boolean; body: string; error?: string; checkedAt: string };

export type ServiceRouteSecurity = {
  path: string;
  matchType: string;
  methods: string[];
  rateLimitRps: number;
};
export type ServiceSecurityPolicy = {
  wafMode: "off" | "detect" | "block";
  authMode: "public" | "api_key";
  upstreamAuthMode: "none" | "basic" | "bearer" | "header";
  upstreamAuthConfigured: boolean;
  requiredClientHeader: string;
  rateLimitRps: number;
  ipAllowlist: string[];
  ipDenylist: string[];
  routes: ServiceRouteSecurity[];
};
export type ServiceNodeSecurity = {
  name: string;
  location: string;
  heartbeat: "fresh" | "stale" | "unknown";
  desiredRevision: number | null;
  ackedRevision: number | null;
  state: "acknowledged" | "outdated" | "offline" | "not_deployed";
  applied: ServiceSecurityPolicy | null;
};
export type ServiceSecurityStatus = {
  serviceId: string;
  status: string;
  pending: boolean;
  saved: ServiceSecurityPolicy;
  nodes: ServiceNodeSecurity[];
};
