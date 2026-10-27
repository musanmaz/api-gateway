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
export type Route = { path: string; matchType: string; methods: string[]; upstream: Upstream };
export type Service = {
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
