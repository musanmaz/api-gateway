"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { TryPanel } from "@/components/try-panel";
import { Badge, ErrorText, Notice, PageTitle, Panel, PrimaryButton, SecondaryButton, Success } from "@/components/ui";
import { api } from "@/lib/api";
import type { Bootstrap, Overview, Route, Service } from "@/lib/types";
import { friendlyError, protocolText } from "@/lib/words";

function routeText(route: Route) {
  const where = route.path === "/" && route.matchType === "prefix" ? "Tüm istekler" : route.matchType === "exact" ? `Yalnızca ${route.path} yolu` : `${route.path} ile başlayan istekler`;
  const methods = route.methods.length ? ` (${route.methods.join(", ")})` : "";
  const targets = route.upstream.endpoints.map((item) => `${item.address}:${item.port}`).join(", ");
  return `${where}${methods} → ${targets}`;
}

export default function ServiceDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [service, setService] = useState<Service | null>(null);
  const [bootstrap, setBootstrap] = useState<Bootstrap | null>(null);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [node, setNode] = useState("");
  const [revision, setRevision] = useState<number | null>(null);

  useEffect(() => {
    if (!sessionStorage.getItem("gateway.token")) router.replace("/login");
  }, [router]);

  async function load() {
    const [item, boot, view] = await Promise.all([api<Service>(`/api/v1/services/${params.id}`), api<Bootstrap>("/api/v1/bootstrap"), api<Overview>("/api/v1/overview")]);
    setService(item);
    setBootstrap(boot);
    setOverview(view);
  }

  useEffect(() => {
    load().catch((err: unknown) => setError(friendlyError(err, "Servis alınamadı")));
  }, [params.id]);

  async function remove() {
    if (!service || !window.confirm("Bu taslak silinsin mi? Yayındaki trafik etkilenmez.")) return;
    setPending(true);
    setError("");
    try {
      await api(`/api/v1/services/${service.id}`, { method: "DELETE" });
      router.push("/services");
    } catch (err) {
      setError(friendlyError(err, "Taslak silinemedi"));
      setPending(false);
    }
  }

  async function publish(target: string) {
    if (!service) return;
    setPending(true);
    setError("");
    setMessage("");
    try {
      const current = await api<Service>(`/api/v1/services/${service.id}`);
      const result = await api<{ revision: { id: number } }>(`/api/v1/services/${service.id}/publish`, {
        method: "POST",
        headers: { "Idempotency-Key": crypto.randomUUID() },
        body: JSON.stringify({ expectedVersion: current.version, ...(target ? { node: target } : {}) }),
      });
      setRevision(result.revision.id);
      setMessage(target ? `Yalnızca ${target} sunucusuna gönderildi.` : "Tüm sunuculara gönderildi.");
      await load();
    } catch (err) {
      setError(friendlyError(err, "Yayına alınamadı"));
    } finally {
      setPending(false);
    }
  }

  const listener = bootstrap?.listeners.find((item) => item.id === service?.listenerId);
  const protocol = listener?.protocol ?? "http";
  const draft = service?.status === "draft";
  const unpublished = service != null && !draft && service.version !== service.publishedVersion;

  return (
    <section>
      <PageTitle title={service ? (service.hostnames[0] ?? service.name) : "Servis"} text={service ? `Sorumlu: ${service.owner}` : undefined} />
      {error ? <ErrorText>{error}</ErrorText> : null}
      {!service && !error ? <p>Yükleniyor…</p> : null}
      {service ? (
        <div className="space-y-4">
          <Panel className="space-y-3">
            <p>
              <Badge value={service.status} /> <span className="text-sm text-stone-600">{protocolText[protocol]?.title ?? protocol}</span>
            </p>
            <ul className="space-y-1">
              {service.routes.map((route) => (
                <li key={`${route.matchType}${route.path}`}>{routeText(route)}</li>
              ))}
            </ul>
            {draft ? <Notice>Bu servis henüz yayında değil. Kimse bu adrese erişemez.</Notice> : null}
            {unpublished ? <Notice>Kaydedilmiş ama yayına alınmamış değişiklikler var. Canlı trafik hâlâ eski ayarı kullanıyor.</Notice> : null}
            <div className="flex flex-wrap gap-2">
              <Link href={`/services/new?service=${service.id}`} className="rounded-lg bg-blue-600 px-4 py-2 font-medium text-white shadow-sm">
                {draft || unpublished ? "Yayına almaya devam et" : "Değiştir"}
              </Link>
              {draft ? (
                <SecondaryButton type="button" disabled={pending} onClick={remove} className="text-red-800">
                  Taslağı sil
                </SecondaryButton>
              ) : null}
            </div>
          </Panel>
          {!draft ? (
            <Panel className="space-y-3">
              <h2 className="text-lg font-medium">Yeniden gönder</h2>
              <p className="text-sm text-stone-600">Bir sunucu eski ayarda kaldıysa veya önce tek sunucuda denediyseniz buradan gönderin.</p>
              {message ? <Success>{message}</Success> : null}
              <div className="flex flex-wrap items-center gap-2">
                <PrimaryButton type="button" disabled={pending} onClick={() => publish("")}>
                  Tüm sunuculara gönder
                </PrimaryButton>
                <select aria-label="Tek sunucu" value={node} onChange={(event) => setNode(event.target.value)} className="rounded-lg border border-stone-300 px-3 py-2">
                  <option value="">Tek sunucu seçin…</option>
                  {overview?.nodes.map((item) => (
                    <option key={item.id} value={item.name}>
                      {item.name}
                    </option>
                  ))}
                </select>
                <SecondaryButton type="button" disabled={pending || !node} onClick={() => publish(node)}>
                  Yalnızca bu sunucuya gönder
                </SecondaryButton>
              </div>
            </Panel>
          ) : null}
          {!draft && bootstrap ? <TryPanel hostname={service.hostnames[0] ?? ""} path={service.routes[0]?.path ?? "/"} tryUrl={bootstrap.tryUrl} revision={revision} web={protocol === "http"} /> : null}
          <details className="rounded-2xl border border-stone-200 bg-white p-4 text-sm">
            <summary className="cursor-pointer">Teknik ayrıntılar</summary>
            <dl className="mt-3 grid grid-cols-[160px_1fr] gap-1">
              <dt className="text-stone-500">Kayıt adı</dt>
              <dd>{service.name}</dd>
              <dt className="text-stone-500">Kayıt sürümü</dt>
              <dd>{service.version}</dd>
              <dt className="text-stone-500">Dinleyici</dt>
              <dd>{listener ? `${listener.name} (port ${listener.port})` : service.listenerId}</dd>
              <dt className="text-stone-500">Önem</dt>
              <dd>{service.importance}</dd>
              {service.routes.map((route) => (
                <div key={`${route.path}-${route.upstream.name}`} className="contents">
                  <dt className="text-stone-500">Hedef grup</dt>
                  <dd>
                    {route.upstream.name} · {route.upstream.algorithm} · bağlantı {route.upstream.connectTimeoutMs} ms · istek {route.upstream.requestTimeoutMs} ms
                  </dd>
                </div>
              ))}
            </dl>
          </details>
        </div>
      ) : null}
    </section>
  );
}
