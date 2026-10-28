import type { NodeInfo, Probe } from "@/lib/types";

export const protocolText: Record<string, { title: string; text: string }> = {
  http: { title: "Web sitesi veya API (HTTP)", text: "Çoğu durumda doğru seçim budur." },
  https: { title: "Şifreli web (HTTPS)", text: "Yerel geliştirme sertifikasıyla." },
  http2: { title: "HTTP/2", text: "Şifresiz HTTP/2." },
  websocket: { title: "WebSocket", text: "Canlı, çift yönlü bağlantı." },
  sse: { title: "Olay akışı (SSE)", text: "Sunucudan sürekli veri akışı." },
  grpc: { title: "gRPC", text: "Servisler arası çağrılar." },
  tcp: { title: "TCP", text: "Ham bağlantı. Yol yoktur." },
  udp: { title: "UDP", text: "Datagram aktarımı. Yol yoktur." },
  tls_passthrough: { title: "TLS geçişi", text: "Şifre çözülmeden iletilir." },
};

export const actionText: Record<string, string> = {
  "service.create": "Taslak oluşturuldu",
  "service.update": "Servis değiştirildi",
  "service.delete": "Taslak silindi",
  "service.publish": "Yayına alındı",
  "revision.rollback": "Eski sürüme dönüldü",
};

export type NodeState = { tone: "ok" | "wait" | "bad" | "off"; text: string };

export function nodeState(node: NodeInfo, revision?: number | null): NodeState {
  if (node.heartbeat === "unknown") return { tone: "off", text: "İlk heartbeat bekleniyor" };
  if (node.heartbeat !== "fresh") return { tone: "bad", text: "Ulaşılamıyor" };
  const target = revision ?? node.desiredRevision;
  if (target == null) return { tone: "off", text: "Henüz yayın yok" };
  if (node.desiredRevision != null && node.desiredRevision < target) return { tone: "off", text: "Bu sürüm bu sunucuya gönderilmedi" };
  if (node.nackRevision === target) return { tone: "bad", text: `Reddetti${node.nackDetail ? `: ${node.nackDetail}` : ""}` };
  if (node.ackedRevision != null && node.ackedRevision >= target) return { tone: "ok", text: "Güncel sürümü kullanıyor" };
  return { tone: "wait", text: "Yeni sürümü alıyor…" };
}

export function probeText(probe: Probe): NodeState {
  if (!probe.status) return { tone: "bad", text: "Hedef sunucuya ulaşılamadı" };
  if (probe.status === 404) return { tone: "bad", text: "Bulunamadı (404)" };
  if (probe.status >= 500) return { tone: "bad", text: `Hedef sunucu hata verdi (${probe.status})` };
  if (probe.status >= 400) return { tone: "bad", text: `İstek reddedildi (${probe.status})` };
  return { tone: "ok", text: `Yanıt verdi (${probe.status})` };
}

export function friendlyError(err: unknown, fallback: string) {
  const message = err instanceof Error ? err.message : "";
  if (message === "çakışma") return "Bu ad veya adres başka bir serviste kullanılıyor.";
  if (message === "kayıt başka bir işlemle değişti") return "Bu servis siz çalışırken değişti. Sayfayı yenileyip tekrar deneyin.";
  if (message === "bu role için kapalı") return "Bu işlem için yetkiniz yok. Salt okunur anahtarla giriş yapılmış.";
  if (message === "kayıt bulunamadı") return "Kayıt bulunamadı. Silinmiş olabilir.";
  return message || fallback;
}

export function slug(value: string, fallback: string) {
  const cleaned = value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 63)
    .replace(/-+$/g, "");
  return cleaned || fallback;
}

export function cleanTarget(address: string, port: string) {
  let host = address.trim().replace(/^[a-z]+:\/\//i, "").replace(/\/.*$/, "");
  let nextPort = port.trim();
  const match = host.match(/^([^:]+):(\d+)$/);
  if (match) {
    host = match[1];
    if (!nextPort) nextPort = match[2];
  }
  return { address: host, port: nextPort };
}
