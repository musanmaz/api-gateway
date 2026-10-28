"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { TryPanel } from "@/components/try-panel";
import { Badge, ErrorText, Notice, PageTitle, Panel, PrimaryButton, SecondaryButton, Success } from "@/components/ui";
import { api } from "@/lib/api";
import type { Bootstrap, Overview, Route, Service, ServiceSecurityStatus } from "@/lib/types";
import { ServiceSecurityOverview } from "@/components/service-security";
import {ServiceTraffic} from "@/components/service-traffic";
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
  const [securityReport, setSecurityReport] = useState<ServiceSecurityStatus | null>(null);
  const [securityError, setSecurityError] = useState("");
  const [securityLoading, setSecurityLoading] = useState(true);
  const [bootstrap, setBootstrap] = useState<Bootstrap | null>(null);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [node, setNode] = useState("");
  const [revision, setRevision] = useState<number | null>(null);
  const [newKey, setNewKey] = useState("");
  const [wafMode,setWafMode]=useState<"off"|"detect"|"block">("off");
  const [rateRps, setRateRps] = useState("0");
  const [routeRps,setRouteRps]=useState<Record<string,string>>({});
  const [allowedIPs,setAllowedIPs]=useState("");
  const [blockedIPs,setBlockedIPs]=useState("");

  useEffect(() => {
    if (!sessionStorage.getItem("gateway.token")) router.replace("/login");
  }, [router]);

  async function refreshSecurityStatus(){
    try {
      const result=await api<ServiceSecurityStatus>("/api/v1/services/"+params.id+"/security-status");
      setSecurityReport(result);
      setSecurityError("");
    }catch(err){
      setSecurityError(friendlyError(err,"Uygulanan güvenlik politikası alınamadı"));
    }finally{setSecurityLoading(false);}
  }
  async function load() {
    const [item, boot, view] = await Promise.all([api<Service>(`/api/v1/services/${params.id}`), api<Bootstrap>("/api/v1/bootstrap"), api<Overview>("/api/v1/overview")]);
    setService(item);
    await refreshSecurityStatus();
    setWafMode(item.wafMode??"off");
    setRateRps(String(item.rateLimitRps ?? 0));
    setAllowedIPs((item.ipAllowlist??[]).join("\n"));
    setBlockedIPs((item.ipDenylist??[]).join("\n"));
    setRouteRps(Object.fromEntries(item.routes.map(r=>[r.id??r.path,String(r.rateLimitRps??0)])));
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

  async function changeAuth(mode: "api_key" | "public") {
    if (!service) return;
    const question = mode === "api_key"
      ? "Yeni bir API Key üretilsin mi? Yeniden yayından sonra eski anahtar geçersiz olur."
      : "API Key korumasını kapatıp servisi Public yapmak istediğinizden emin misiniz?";
    if (!window.confirm(question)) return;
    setPending(true);
    setError("");
    setMessage("");
    setNewKey("");
    try {
      const path = "/api/v1/services/" + service.id + (mode === "api_key" ? "/auth-key" : "/auth-public");
      const result = await api<{key?: string;service:Service}>(path, {method:"POST"});
      setService(result.service);
      await refreshSecurityStatus();
      if (result.key) setNewKey(result.key);
      setMessage("Güvenlik ayarı kaydedildi. Uygulamak için aşağıdan tüm gateway sunucularına yayınlayın.");
    } catch (err) {
      setError(friendlyError(err, "Güvenlik ayarı kaydedilemedi"));
    } finally {
      setPending(false);
    }
  }

  async function saveWAF(){
    if (!service) return;
    if (wafMode==="block" && !window.confirm("CRS WAF engelleme modu false positive yaratabilir. İzleme modunda doğruladınız mı?"))return;
    setPending(true);setMessage("");setError("");
    try {
      const result=await api<{service:Service}>("/api/v1/services/"+service.id+"/waf",{
        method:"POST",body:JSON.stringify({mode:wafMode})
      });
      setService(result.service);
      await refreshSecurityStatus();
      setMessage("WAF politikası kaydedildi. Etkinleştirmek için aşağıdan tüm düğümlere yayınlayın.");
    }catch(err){setError(friendlyError(err,"WAF ayarı kaydedilemedi"));}
    finally{setPending(false);}
  }

  async function saveRateLimit() {
    if (!service) return;
    const value=Number(rateRps);
    if (!Number.isInteger(value)||value<0||value>10000) {setError("Rate limit 0-10000 arasında olmalı");return;}
    setPending(true);setError("");setMessage("");
    try {
      const updated = await api<{service:Service}>(`/api/v1/services/${service.id}/rate-limit`,{
        method:"POST",body:JSON.stringify({requestsPerSecond:value}),
      });
      setService(updated.service);
      await refreshSecurityStatus();
      setMessage("Rate limit kaydedildi. Etkinleştirmek için tüm sunuculara yayınlayın.");
    } catch (err) {setError(friendlyError(err,"Rate limit kaydedilemedi"));}
    finally {setPending(false);}
  }

  async function saveRouteLimit(route:Route){
    if (!service || !route.id) return;
    const n=Number(routeRps[route.id]??"0");
    if (!Number.isInteger(n)||n<0||n>10000){setError("Endpoint kotası 0-10000 olmalı");return;}
    setPending(true);setError("");setMessage("");
    try {
      const result=await api<{service:Service}>("/api/v1/services/"+service.id+"/routes/"+route.id+"/rate-limit",{
        method:"POST",body:JSON.stringify({requestsPerSecond:n})
      });
      setService(result.service);
      await refreshSecurityStatus();
      setMessage("Endpoint kotası kaydedildi. Yayınlama sonrası aktif olur.");
    }catch(err){setError(friendlyError(err,"Endpoint kotası güncellenemedi"));}
    finally{setPending(false);}
  }

  async function saveIPPolicy(){
    if (!service) return;
    const list=(str:string)=>str.split(/[\n,; ]+/).map(x=>x.trim()).filter(Boolean);
    setPending(true);setError("");setMessage("");
    try {
      const result=await api<{service:Service}>("/api/v1/services/"+service.id+"/ip-policy",{
        method:"POST",body:JSON.stringify({allow:list(allowedIPs),deny:list(blockedIPs)})
      });
      setService(result.service);
      await refreshSecurityStatus();
      setMessage("IP politikası kaydedildi; etkinleştirmek için yayınlayın.");
    }catch(err){setError(friendlyError(err,"IP politikası kaydedilemedi"));}
    finally{setPending(false);}
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
        <div className="mt-6 space-y-4">
          <Panel className="space-y-3">
            <p>
              <Badge value={service.status} /> <span className="text-sm text-slate-600">{protocolText[protocol]?.title ?? protocol}</span>
            </p>
            {service.gatewayBaseUrl ? (
              <div className="rounded-xl border border-teal-100 bg-teal-50/40 p-4">
                <h2 className="text-sm font-semibold text-slate-800">Merkezi Gateway erişim linkleri</h2>
                <p className="mt-1 text-xs text-slate-500">Tüm istekler api-gateway.mehmet.tech üzerinden yönlendirilir.</p>
                <div className="mt-3 space-y-3">{service.routes.map(route => {
                  const url = service.gatewayBaseUrl + route.path;
                  return <div key={route.path + route.matchType} className="flex flex-wrap items-center gap-3">
                    {draft ? <code className="break-all text-xs text-slate-500">{url} (yayın sonrası)</code> :
                      <a className="break-all text-sm font-medium text-teal-700 underline" href={url} target="_blank" rel="noreferrer">{url}</a>}
                    <button type="button" className="rounded-md border border-teal-200 px-3 py-1.5 text-xs text-teal-700"
                      onClick={() => navigator.clipboard.writeText(url)}>Linki kopyala</button>
                  </div>;
                })}</div>
              </div>
            ) : <Notice>Bu bağlantı türü için merkezi HTTP erişim linki tanımlı değil.</Notice>}
            <ul className="space-y-1">
              {service.routes.map((route) => (
                <li key={`${route.matchType}${route.path}`}>{routeText(route)}</li>
              ))}
            </ul>
            {draft ? <Notice>Bu servis henüz yayında değil. Kimse bu adrese erişemez.</Notice> : null}
            {unpublished ? <Notice>Kaydedilmiş ama yayına alınmamış değişiklikler var. Canlı trafik hâlâ eski ayarı kullanıyor.</Notice> : null}
            <div className="flex flex-wrap gap-2">
              <Link href={`/services/new?service=${service.id}`} className="rounded-lg bg-teal-700 px-4 py-2 font-medium text-white shadow-sm">
                {draft || unpublished ? "Yayına almaya devam et" : "Değiştir"}
              </Link>
              {draft ? (
                <SecondaryButton type="button" disabled={pending} onClick={remove} className="text-red-800">
                  Taslağı sil
                </SecondaryButton>
              ) : null}
            </div>
          </Panel>
          {protocol === "http" && !draft ? <ServiceTraffic serviceId={service.id}/> : null}
          {protocol === "http" ? <ServiceSecurityOverview
            service={service} report={securityReport} loading={securityLoading} error={securityError} /> : null}
          {protocol === "http" ? <div id="security-settings" className="scroll-mt-8 space-y-4">
            <h2 className="text-lg font-semibold">Güvenlik politikası ayarları</h2>
            <Panel className="space-y-4">
              <h3 className="text-base font-medium">Servis güvenliği</h3>
              <p className="text-xs text-slate-600">Public veya API Key korumalı erişim. Koruma açıldığında X-Gateway-Key header gereklidir.</p>
              <p className="text-sm font-semibold">{service.authMode === "api_key" ? "API Key zorunlu" : "Public — kimlik doğrulaması yok"}</p>
              <div className="flex flex-wrap gap-2">
                <SecondaryButton disabled={pending} onClick={() => changeAuth("api_key")}>
                  {service.authMode === "api_key" ? "API Key yenile" : "API Key oluştur ve korumayı aç"}
                </SecondaryButton>
                {service.authMode === "api_key" ? <SecondaryButton disabled={pending} onClick={() => changeAuth("public")}>
                  Auth kapat (Public)
                </SecondaryButton> : null}
              </div>
              {newKey ? <div className="rounded-lg border border-amber-300 bg-amber-50 p-4">
                <p className="text-sm font-semibold">Yeni API Key — yalnızca bir kez gösterilir</p>
                <p className="mt-1 text-xs">Anahtarı şimdi güvenli bir kasaya kaydedin. Bu sayfayı yenileyince kaybolur.</p>
                <code className="mt-3 block break-all select-all rounded bg-white p-3 text-xs">{newKey}</code>
                <button type="button" className="mt-3 rounded border px-3 py-2 text-xs" onClick={() => navigator.clipboard.writeText(newKey)}>Anahtarı kopyala</button>
                <p className="mt-2 text-xs text-amber-900">İstemci isteği örneği: X-Gateway-Key: [ANAHTAR]</p>
              </div> : null}
              {message && <Success>{message}</Success>}
              {unpublished && <Notice>Yeni güvenlik ayarını tüm gateway node'larına göndermeden etkinleşmez.</Notice>}
            </Panel>
          {protocol==="http" ? <Panel className="space-y-3">
            <h2 className="text-lg font-medium">OWASP CRS / Coraza WAF</h2>
            <p className="text-xs text-slate-600">İzle modunda istek engellenmez, saldırı eşleşmeleri sayılır. Engelle modunda CRS anomali eşleşmeleri HTTP 403 döndürür. WAF aktif rotalarda 1 MiB üstü istekler HTTP 413 alır.</p>
            <div className="flex flex-wrap gap-3 items-center">
              <select aria-label="WAF çalışma modu" className="rounded-lg border border-slate-300 px-3 py-2 text-sm"
                value={wafMode} onChange={e=>setWafMode(e.target.value as "off"|"detect"|"block")}>
                <option value="off">Kapalı / Off</option>
                <option value="detect">İzle / Detect-only</option>
                <option value="block">Engelle / Block</option>
              </select>
              <SecondaryButton type="button" disabled={pending} onClick={saveWAF}>WAF politikasını kaydet</SecondaryButton>
            </div>
            <p className="text-xs text-amber-700">Önce izleme modunda normal trafiğinizi test edin. WAF açık rotalarda request-body tamponlama gecikme ve bellek tüketimini artırır.</p>
          </Panel>:null}
          {protocol === "http" ? <Panel className="space-y-3">
            <h2 className="text-lg font-medium">Rate limiting</h2>
            <p className="text-xs text-slate-600">Servis başına, Envoy düğümü başına istek/saniye. 0 limitsiz; aşılırsa 429 döner. Dağıtık global kota değildir.</p>
            <div className="flex flex-wrap items-center gap-3">
              <input aria-label="İstek/saniye" className="w-36 rounded-lg border border-slate-300 px-3 py-2 text-sm"
                 type="number" min="0" max="10000" value={rateRps} onChange={e=>setRateRps(e.target.value)}/>
              <SecondaryButton type="button" disabled={pending} onClick={saveRateLimit}>Rate limit kaydet</SecondaryButton>
            </div>
          </Panel> : null}
          {protocol==="http" ? <Panel className="space-y-4">
            <h2 className="text-lg font-medium">Endpoint bazlı rate limit</h2>
            <p className="text-xs text-slate-500">0 = servis genel kotasını kullan. Endpoint kota değeri, servis kotasının yerini alır; düğüm başına geçerlidir.</p>
            {service.routes.map(route=><div key={route.id??route.path} className="flex flex-wrap items-center gap-3 border-t border-slate-100 pt-3">
              <code className="flex-1 min-w-32 break-all text-xs">{route.methods.join(",")||"ANY"} {route.path}</code>
              <input aria-label={route.path+" endpoint requests per second"} type="number" min="0" max="10000"
               className="w-28 rounded-lg border border-slate-300 px-2 py-2 text-xs"
               value={routeRps[route.id??route.path]??"0"}
               onChange={e=>setRouteRps(prev=>({...prev,[route.id??route.path]:e.target.value}))} />
              <SecondaryButton type="button" disabled={pending || !route.id} onClick={()=>saveRouteLimit(route)}>Kaydet</SecondaryButton>
            </div>)}
          </Panel>:null}
          {protocol==="http" ? <Panel className="space-y-4">
            <h2 className="text-lg font-medium">IP allowlist / denylist</h2>
            <p className="text-xs text-slate-500">IP veya CIDR, her satıra bir kayıt. Denylist önceliklidir. Allowlist doluysa yalnız listedeki IP'lere izin verilir. Giriş adresi güvenilir proxy zincirinden hesaplanır. Yanlış allowlist sizi kilitleyebilir.</p>
            <div className="grid gap-3 sm:grid-cols-2">
              <label className="text-xs font-semibold">Allowlist
                <textarea className="mt-1 block min-h-28 w-full rounded-lg border border-slate-300 p-2 font-mono text-xs" value={allowedIPs} onChange={e=>setAllowedIPs(e.target.value)} placeholder="192.0.2.10/32" />
              </label>
              <label className="text-xs font-semibold">Denylist
                <textarea className="mt-1 block min-h-28 w-full rounded-lg border border-slate-300 p-2 font-mono text-xs" value={blockedIPs} onChange={e=>setBlockedIPs(e.target.value)} placeholder="203.0.113.0/24" />
              </label>
            </div>
            <SecondaryButton type="button" disabled={pending} onClick={saveIPPolicy}>IP politikasını kaydet</SecondaryButton>
          </Panel>:null}
          </div>:null}
          {!draft ? (
            <Panel className="space-y-3">
              <h2 className="text-lg font-medium">Yeniden gönder</h2>
              <p className="text-sm text-slate-600">Bir sunucu eski ayarda kaldıysa veya önce tek sunucuda denediyseniz buradan gönderin.</p>
              {message ? <Success>{message}</Success> : null}
              <div className="flex flex-wrap items-center gap-2">
                <PrimaryButton type="button" disabled={pending} onClick={() => publish("")}>
                  Tüm sunuculara gönder
                </PrimaryButton>
                <select aria-label="Tek sunucu" value={node} onChange={(event) => setNode(event.target.value)} className="rounded-lg border border-slate-300 px-3 py-2">
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
          {!draft && bootstrap && !(service.hostnames[0] ?? "").endsWith(".gateway.internal") ? <TryPanel hostname={service.hostnames[0] ?? ""} path={service.routes[0]?.path ?? "/"} tryUrl={bootstrap.tryUrl} revision={revision} web={protocol === "http"} /> : null}
          <details className="rounded-2xl border border-slate-200 bg-white p-4 text-sm">
            <summary className="cursor-pointer">Teknik ayrıntılar</summary>
            <dl className="mt-3 grid grid-cols-[160px_1fr] gap-1">
              <dt className="text-slate-500">Kayıt adı</dt>
              <dd>{service.name}</dd>
              <dt className="text-slate-500">Kayıt sürümü</dt>
              <dd>{service.version}</dd>
              <dt className="text-slate-500">Dinleyici</dt>
              <dd>{listener ? `${listener.name} (port ${listener.port})` : service.listenerId}</dd>
              <dt className="text-slate-500">Önem</dt>
              <dd>{service.importance}</dd>
              {service.routes.map((route) => (
                <div key={`${route.path}-${route.upstream.name}`} className="contents">
                  <dt className="text-slate-500">Hedef grup</dt>
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
