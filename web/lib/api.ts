export class ApiError extends Error {}

export function token(): string {
  if (typeof window === "undefined") return "";
  return sessionStorage.getItem("gateway.token") ?? "";
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  const current = token();
  if (current) headers.set("Authorization", `Bearer ${current}`);
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(`/gateway-api${path}`, { ...init, headers, cache: "no-store" });
  if (response.status === 401) {
    sessionStorage.removeItem("gateway.token");
    if (window.location.pathname !== "/login") window.location.href = "/login";
    throw new ApiError("Oturum doğrulanamadı");
  }
  if (response.status === 204) return undefined as T;
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    const details = Array.isArray(body.details) ? body.details.join(" ") : "";
    throw new ApiError(body.error ? `${body.error}${details ? `: ${details}` : ""}` : `HTTP ${response.status}`);
  }
  return body as T;
}
