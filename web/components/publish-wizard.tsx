"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { TryPanel } from "@/components/try-panel";
import { ErrorText, Panel, PageTitle, PrimaryButton, SecondaryButton, Success } from "@/components/ui";
import { api } from "@/lib/api";
import type { Bootstrap, DiffChange, Overview, Route, Service } from "@/lib/types";
import { cleanTarget, friendlyError, protocolText, slug } from "@/lib/words";
import { useLanguage } from "@/lib/i18n";

const STEPS = ["Adres", "Hedef", "Kontrol", "Yayına al", "Dene"];
const METHODS = ["GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"];

type Check = { valid: boolean; errors: string[]; changes: DiffChange[] };
export function PublishWizard({ serviceId }: { serviceId?: string }) {
  const router = useRouter();
  const {t}=useLanguage();
  const [bootstrap, setBootstrap] = useState<Bootstrap | null>(null);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [step, setStep] = useState(1);
  const [hostname, setHostname] = useState("");
  const [serviceName, setServiceName] = useState("");
  const [owner, setOwner] = useState("");
  const [address, setAddress] = useState("");
  const [port, setPort] = useState("");
  const [path, setPath] = useState("/");
  const [listenerId, setListenerId] = useState("");
  const [matchType, setMatchType] = useState("prefix");
  const [methods, setMethods] = useState<string[]>([]);
  const [advanced, setAdvanced] = useState(false);
  const [rateRps, setRateRps] = useState("0");
  const [backendAuth, setBackendAuth] = useState<"none"|"basic"|"bearer"|"header">("none");
  const [backendUsername, setBackendUsername] = useState("");
  const [backendSecret, setBackendSecret] = useState("");
  const [backendHeader, setBackendHeader] = useState("X-API-Key");
  const [requiredHeader, setRequiredHeader] = useState("");
  const [loaded, setLoaded] = useState<Service | null>(null);
  const [extra, setExtra] = useState<Route[]>([]);
  const [saved, setSaved] = useState<{ id: string; version: number } | null>(null);
  const [check, setCheck] = useState<Check | null>(null);
  const [mode, setMode] = useState<"all" | "one">("all");
  const [node, setNode] = useState("");
  const [revision, setRevision] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    if (!sessionStorage.getItem("gateway.token")) {
      router.replace("/login");
      return;
    }
    async function load() {
      const [boot, view] = await Promise.all([api<Bootstrap>("/api/v1/bootstrap"), api<Overview>("/api/v1/overview")]);
      setBootstrap(boot);
      setOverview(view);
      setNode(view.nodes[0]?.name ?? "");
      if (!serviceId) {
        setListenerId(boot.listener.id);
        return;
      }
      const item = await api<Service>(`/api/v1/services/${serviceId}`);
      const first = item.routes[0];
      setLoaded(item);
      setSaved({ id: item.id, version: item.version });
      setHostname((item.hostnames[0] ?? "").endsWith(".gateway.internal") ? "" : item.hostnames[0] ?? "");
      setServiceName(item.name);
      setOwner(item.owner);
      setRateRps(String(item.rateLimitRps ?? 0));
      setBackendAuth(item.upstreamAuthMode ?? "none");
      setBackendUsername(item.upstreamAuthUsername ?? "");
      setBackendHeader(item.upstreamAuthHeader || "X-API-Key");
      setRequiredHeader(item.requiredClientHeader ?? "");
      setListenerId(item.listenerId);
      if (first) {
        setPath(first.path);
        setMatchType(first.matchType);
        setMethods(first.methods);
        setAddress(first.upstream.endpoints[0]?.address ?? "");
        setPort(String(first.upstream.endpoints[0]?.port ?? ""));
      }
      setExtra(item.routes.slice(1));
      if (item.status === "draft") setStep(3);
    }
    load().catch((err: unknown) => setError(friendlyError(err, "Bilgiler alınamadı")));
  }, [router, serviceId]);

  const listener = bootstrap?.listeners.find((item) => item.id === listenerId) ?? bootstrap?.listener;
  const protocol = listener?.protocol ?? "http";
  const layer4 = protocol === "tcp" || protocol === "udp" || protocol === "tls_passthrough";
  const routePath = layer4 ? "/" : path;

  useEffect(() => {
    if (step === 3 && bootstrap && !check && !pending) runCheck();
  }, [step, bootstrap]);

  function body() {
    const target = cleanTarget(address, port);
    return {
      name: loaded?.name ?? slug(serviceName, "servis"),
      owner: owner.trim(),
      importance: loaded?.importance ?? "medium",
      description: loaded?.description ?? "",
      environmentId: bootstrap!.environmentId,
      teamId: bootstrap!.teamId,
      fleetId: bootstrap!.fleetId,
      listenerId: listener!.id,
      hostnames: [hostname.trim().toLowerCase() || (loaded ? (loaded.hostnames[0] ?? "") : slug(serviceName, "servis") + ".gateway.internal")],
      expectedVersion: saved?.version ?? 0,
      routes: [
        {
          path: routePath,
          matchType: layer4 ? "prefix" : matchType,
          rateLimitRps: loaded?.routes[0]?.rateLimitRps ?? 0,
          methods: layer4 || protocol === "grpc" ? [] : methods,
          upstream: {
            name: slug(target.address, "hedef"),
            algorithm: "round_robin",
            endpoints: [{ address: target.address, port: Number(target.port), weight: 1 }],
          },
        },
        ...extra,
      ],
    };
  }

  async function runCheck() {
    setPending(true);
    setError("");
    try {
      const payload = body();
      const item = saved
        ? await api<Service>(`/api/v1/services/${saved.id}`, { method: "PUT", body: JSON.stringify(payload) })
        : await api<Service>("/api/v1/services", { method: "POST", body: JSON.stringify(payload) });
      const authChanged = !loaded ||
        backendAuth !== (loaded.upstreamAuthMode ?? "none") ||
        backendUsername !== (loaded.upstreamAuthUsername ?? "") ||
        (backendAuth === "header" && backendHeader !== (loaded.upstreamAuthHeader ?? "")) ||
        requiredHeader !== (loaded.requiredClientHeader ?? "") || backendSecret.length > 0;
      let configured = item;
      if (authChanged && (backendAuth !== "none" || requiredHeader.trim() || loaded?.upstreamAuthConfigured)) {
        const security = await api<{service:Service}>(`/api/v1/services/${item.id}/upstream-auth`, {
          method: "POST",
          body: JSON.stringify({
            mode: backendAuth, username: backendAuth === "basic" ? backendUsername : "",
            headerName: backendAuth === "header" ? backendHeader : "",
            requiredClientHeader: requiredHeader.trim(), secret: backendSecret
          }),
        });
        configured = security.service;
        setBackendSecret("");
      }
      if (Number(rateRps) !== (loaded?.rateLimitRps ?? 0)) {
        const limit = await api<{service:Service}>(`/api/v1/services/${configured.id}/rate-limit`, {
          method:"POST",
          body:JSON.stringify({requestsPerSecond:Number(rateRps)}),
        });
        configured=limit.service;
      }
      const result = await api<{ valid: boolean; errors: string[]; service: Service }>(`/api/v1/services/${configured.id}/validate`, { method: "POST" });
      const diff = await api<{ changes: DiffChange[] }>(`/api/v1/services/${item.id}/diff`);
      setSaved({ id: item.id, version: result.service.version });
      setCheck({ valid: result.valid, errors: result.errors ?? [], changes: diff.changes });
    } catch (err) {
      setCheck({ valid: false, errors: [friendlyError(err, "Kontrol yapılamadı")], changes: [] });
    } finally {
      setPending(false);
    }
  }

  async function publish(target: string) {
    if (!saved) return;
    setPending(true);
    setError("");
    try {
      const current = await api<Service>(`/api/v1/services/${saved.id}`);
      const result = await api<{ revision: { id: number } }>(`/api/v1/services/${saved.id}/publish`, {
        method: "POST",
        headers: { "Idempotency-Key": crypto.randomUUID() },
        body: JSON.stringify({ expectedVersion: current.version, ...(target ? { node: target } : {}) }),
      });
      setRevision(result.revision.id);
      setMode(target ? "one" : "all");
      setStep(5);
    } catch (err) {
      setError(friendlyError(err, "Yayına alınamadı"));
    } finally {
      setPending(false);
    }
  }

  function next() {
    setError("");
    if (step === 1) {
      if (!loaded && !serviceName.trim()) return setError("Servis kısa adını girin. Örnek: api3");
      if (hostname.trim() && !hostname.trim().includes(".")) return setError("Özel alan adı en az bir nokta içermeli.");
      if (!owner.trim()) return setError("Sorumlu kişi veya ekip boş olamaz.");
    }
    if (step === 2) {
      const target = cleanTarget(address, port);
      const number = Number(target.port);
      if (!target.address) return setError("Hedef sunucu adresini yazın. Örnek: backend-a veya 10.0.0.5");
      if (!Number.isInteger(number) || number < 1 || number > 65535) return setError("Port 1 ile 65535 arasında bir sayı olmalı. Örnek: 8080");
      if (!layer4 && !path.startsWith("/")) return setError("Yol / ile başlamalı. Tüm site için yalnızca / yazın.");
      if (backendAuth === "basic" && !backendUsername.trim()) return setError("Basic Auth için kullanıcı adı zorunlu.");
      if (backendAuth === "header" && !backendHeader.trim()) return setError("Özel header adı zorunlu.");
      if (backendAuth !== "none" && !backendSecret && (!loaded?.upstreamAuthConfigured || loaded.upstreamAuthMode !== backendAuth)) {
        return setError("Seçtiğiniz kimlik doğrulama türü için parola veya token zorunlu.");
      }
      if (!Number.isInteger(Number(rateRps)) || Number(rateRps)<0 || Number(rateRps)>10000) return setError("Rate limit 0-10000 istek/saniye olmalı.");
      setAddress(target.address);
      setPort(target.port);
      setCheck(null);
    }
    setStep(step + 1);
  }

  return (
    <section>
      <PageTitle title={loaded ? `${loaded.hostnames[0] ?? loaded.name} servisini güncelle` : "Yeni site veya API'yi yayına al"} text="Beş kısa adım. Son adıma kadar canlı trafik değişmez." />
      <ol className="mb-6 flex flex-wrap gap-2 text-sm">
        {STEPS.map((label, index) => (
          <li key={label} className={`rounded-full px-3 py-1 ${index + 1 === step ? "bg-teal-700 text-white" : index + 1 < step ? "bg-teal-100 text-teal-900" : "bg-slate-200 text-slate-600"}`}>
            {index + 1}. {t(label)}
          </li>
        ))}
      </ol>
      {error ? <ErrorText>{error}</ErrorText> : null}
      {!bootstrap && !error ? <p>Yükleniyor…</p> : null}
      {bootstrap && step === 1 ? (
        <Panel className="space-y-4">
          {loaded ? <p className="text-sm text-slate-600">Servis kısa adı: <strong>{loaded.name}</strong> (yayın adresi sabittir)</p> :
            <Field id="serviceName" label={t("Servis kısa adı")} hint="Gateway adresinde kullanılır. Örneğin api3 → /api/api3/ (a-z, 0-9 ve tire)" value={serviceName} onChange={setServiceName} placeholder="api3" />}
          <Field id="hostname" label={t("Ek özel alan adı (isteğe bağlı)")} hint="Boş bırakabilirsiniz. Servise yalnızca merkezi gateway üzerinden ulaşılabilir." value={hostname} onChange={setHostname} placeholder="ornek-api.mehmet.tech" />
          <Field id="owner" label={t("Bu servisten kim sorumlu?")} hint="Sorun olunca kime sorulacağı. Örnek: ödeme ekibi" value={owner} onChange={setOwner} placeholder="ödeme ekibi" />
        </Panel>
      ) : null}
      {bootstrap && step === 2 ? (
        <Panel className="space-y-4">
          <p className="text-sm text-slate-600">
            <strong>{loaded?.name ?? slug(serviceName, "servis")}</strong> servisine gelen istekler hangi sunucuya gitsin?
          </p>
          <div className="grid gap-4 sm:grid-cols-[1fr_140px]">
            <Field id="address" label={t("Hedef sunucu")} hint="Ad veya IP. http:// yazmanıza gerek yok." value={address} onChange={setAddress} placeholder="backend-a" />
            <Field id="port" label={t("Port")} hint="Genelde 80 veya 8080." value={port} onChange={setPort} placeholder="8080" />
          </div>
          {!layer4 ? <Field id="path" label={t("Hangi yol?")} hint="Tüm site için / bırakın. Yalnızca /api ile başlayanlar için /api yazın." value={path} onChange={setPath} placeholder="/" /> : null}
          {extra.length > 0 ? <p className="text-sm text-slate-600">Bu serviste {extra.length} yol daha var. Onlar olduğu gibi kalır.</p> : null}
          {!layer4 && protocol === "http" ? <div className="rounded-xl border border-slate-200 bg-white p-5 space-y-4">
            <div><h2 className="text-base font-semibold">Backend kimlik doğrulaması ve header'lar</h2>
              <p className="mt-1 text-xs text-slate-500">Gateway'in hedef sunucuya gönderdiği kimlik bilgilerini burada tanımlayın. Bu, istemciden API Key istemekten farklıdır.</p></div>
            <label className="block text-sm">Backend auth türü
              <select aria-label="Backend auth türü" value={backendAuth} onChange={e => { setBackendAuth(e.target.value as typeof backendAuth);setBackendSecret("");}} className="mt-2 w-full rounded-lg border border-slate-300 bg-white px-3 py-2">
                <option value="none">Yok — istek olduğu gibi yönlendirilir</option>
                <option value="basic">Basic Auth — kullanıcı adı / parola</option>
                <option value="bearer">Bearer Token — Authorization header</option>
                <option value="header">Özel header — örn. X-API-Key</option>
              </select>
            </label>
            {backendAuth === "basic" ? <Field id="backend-username" label="Backend kullanıcı adı" hint="Yalnızca gateway tarafından backend'e gönderilir." value={backendUsername} onChange={setBackendUsername} placeholder="api-user" /> : null}
            {backendAuth === "header" ? <Field id="backend-header" label="Header adı" hint="Örn. X-API-Key, X-Client-Token. Authorization için Bearer/Basic kullanın." value={backendHeader} onChange={setBackendHeader} placeholder="X-API-Key" /> : null}
            {backendAuth !== "none" ? <label className="block text-sm">
              {backendAuth === "basic" ? "Backend parolası" : backendAuth === "bearer" ? "Bearer token" : "Header değeri"}
              <input type="password" autoComplete="off" value={backendSecret} onChange={e => setBackendSecret(e.target.value)}
                placeholder={loaded?.upstreamAuthConfigured && loaded.upstreamAuthMode === backendAuth ? "Kayıtlı gizli değer var — değiştirmek için yenisini yazın" : "Gizli değeri girin"}
                className="mt-2 block w-full rounded-lg border border-slate-300 px-3 py-2" />
              <span className="mt-1 block text-xs text-slate-500">Gizli değer şifreli saklanır; kayıt sonrası bir daha görüntülenmez. Boş bırakmak mevcut değeri korur (aynı auth türünde).</span>
            </label> : null}
            <Field id="required-client-header" label={t("İstemciden zorunlu header (isteğe bağlı)")}
              hint="İstemciden istek gelirken bu header yoksa gateway 401 döner. Örn. X-Client-Id. Değeri kontrol etmez, yalnız dolu olmasını ister."
              value={requiredHeader} onChange={setRequiredHeader} placeholder="X-Client-Id" />
            <p className="text-xs text-slate-500">Bu ayarlar kaydedildikten sonra yayınlama adımında iki Envoy node'una dağıtılır.</p>
          </div> : null}
          {!layer4 && protocol === "http" ? <div className="rounded-xl border border-slate-200 bg-white p-5">
            <label className="block text-sm font-semibold">Rate limit (istek/saniye, Envoy düğümü başına)
              <input type="number" min="0" max="10000" step="1" value={rateRps} onChange={e=>setRateRps(e.target.value)}
                className="mt-2 block w-full rounded-lg border border-slate-300 px-3 py-2" />
              <span className="mt-2 block text-xs font-normal text-slate-500">0 = limitsiz; örneğin 20 değeri tek Envoy üzerinde saniyede yaklaşık 20 isteğe izin verir. Aşılırsa 429. Bu global dağıtık kota değildir.</span>
            </label>
          </div> : null}
          <button type="button" className="text-sm underline" onClick={() => setAdvanced(!advanced)}>
            {advanced ? t("Gelişmiş ayarları gizle") : t("Gelişmiş ayarlar")}
          </button>
          {advanced ? (
            <div className="space-y-4 rounded-xl bg-slate-50 p-4">
              <label className="block text-sm" htmlFor="protocol">
                Bağlantı türü
                <select id="protocol" value={listenerId} onChange={(event) => setListenerId(event.target.value)} className="mt-1 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2">
                  {bootstrap.listeners.map((item) => (
                    <option key={item.id} value={item.id}>
                      {protocolText[item.protocol]?.title ?? item.protocol} · {protocolText[item.protocol]?.text ?? ""}
                    </option>
                  ))}
                </select>
              </label>
              {!layer4 ? (
                <label className="block text-sm" htmlFor="match">
                  Yol eşleşmesi
                  <select id="match" value={matchType} onChange={(event) => setMatchType(event.target.value)} className="mt-1 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2">
                    <option value="prefix">Bu yolla başlayan her şey</option>
                    <option value="exact">Yalnızca tam bu yol</option>
                  </select>
                </label>
              ) : null}
              {!layer4 && protocol !== "grpc" ? (
                <fieldset>
                  <legend className="text-sm">İzin verilen istek türleri. Hiçbiri seçili değilse hepsi geçer.</legend>
                  <div className="mt-2 flex flex-wrap gap-3">
                    {METHODS.map((method) => (
                      <label key={method} className="text-sm">
                        <input type="checkbox" className="mr-1" checked={methods.includes(method)} onChange={() => setMethods(methods.includes(method) ? methods.filter((item) => item !== method) : [...methods, method])} />
                        {method}
                      </label>
                    ))}
                  </div>
                </fieldset>
              ) : null}
            </div>
          ) : null}
        </Panel>
      ) : null}
      {bootstrap && step === 3 ? (
        <Panel className="space-y-3">
          {pending || !check ? <p>Kaydediliyor ve kontrol ediliyor…</p> : null}
          {check && check.valid ? <Success>Kontrol geçti. Taslak kaydedildi; canlı trafik henüz değişmedi.</Success> : null}
          {check && !check.valid ? (
            <>
              <ErrorText>Yayına almadan önce düzeltilmesi gerekenler var.</ErrorText>
              <ul className="list-disc pl-5 text-sm">
                {check.errors.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </>
          ) : null}
          {check && check.valid ? (
            <div>
              <h2 className="font-medium">Yayına alınınca ne değişecek?</h2>
              {check.changes.length === 0 ? <p className="mt-1 text-sm text-slate-600">Hiçbir şey değişmeyecek; yayındaki ayar zaten bu.</p> : null}
              <ul className="mt-2 space-y-1 text-sm">
                {check.changes.map((change) => (
                  <li key={`${change.op}${change.host}${change.path}`}>
                    <span className="font-medium">{change.op === "add" ? "Eklenecek" : change.op === "remove" ? "Kaldırılacak" : "Değişecek"}:</span> {change.host}
                    {change.path} → {(change.after ?? change.before ?? "").split("→").pop()?.trim()}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </Panel>
      ) : null}
      {bootstrap && step === 4 ? (
        <Panel className="space-y-4">
          <p>Nereye yayına alalım?</p>
          <label className="flex gap-3 rounded-xl border border-slate-200 p-3">
            <input type="radio" name="mode" checked={mode === "all"} onChange={() => setMode("all")} />
            <span>
              <span className="block font-medium">Tüm sunucular</span>
              <span className="text-sm text-slate-600">Herkes yeni ayarı hemen kullanır.</span>
            </span>
          </label>
          <label className="flex gap-3 rounded-xl border border-slate-200 p-3">
            <input type="radio" name="mode" checked={mode === "one"} onChange={() => setMode("one")} />
            <span>
              <span className="block font-medium">Önce tek sunucuda dene</span>
              <span className="text-sm text-slate-600">Sorun çıkarsa yalnızca o sunucu etkilenir. Sonra tek tuşla hepsine açarsınız.</span>
            </span>
          </label>
          {mode === "one" ? (
            <select aria-label="Denenecek sunucu" value={node} onChange={(event) => setNode(event.target.value)} className="block rounded-lg border border-slate-300 px-3 py-2">
              {overview?.nodes.map((item) => (
                <option key={item.id} value={item.name}>
                  {item.name}
                </option>
              ))}
            </select>
          ) : null}
        </Panel>
      ) : null}
      {bootstrap && step === 5 && saved ? (
        <div className="space-y-4">
          <Success>{mode === "one" ? `Yayın yalnızca ${node} sunucusuna gönderildi. Uygulanma durumunu aşağıdan izleyin.` : "Yayın tüm sunuculara gönderildi. Uygulanma durumunu aşağıdan izleyin."}</Success>
          {protocol === "http" ? <div className="gc-panel p-5"><p className="text-xs font-medium text-slate-500">Merkezi gateway erişim linki</p><a className="mt-2 inline-block break-all text-sm font-medium text-teal-700 underline" href={`https://api-gateway.mehmet.tech/api/${loaded?.name ?? slug(serviceName, "servis")}${routePath}`} target="_blank" rel="noreferrer">https://api-gateway.mehmet.tech/api/{loaded?.name ?? slug(serviceName, "servis")}{routePath}</a><p className="mt-2 text-xs text-slate-500">Gateway üzerinden yayınlandı. Özel DNS gerekmez.</p></div> : null}
          {hostname && protocol === "http" ? <TryPanel hostname={hostname} path={routePath} tryUrl={bootstrap.tryUrl} revision={revision} web /> : null}
          <div className="flex flex-wrap gap-2">
            {mode === "one" ? (
              <PrimaryButton type="button" disabled={pending} onClick={() => publish("")}>
                Sorun yok, tüm sunuculara aç
              </PrimaryButton>
            ) : null}
            <Link href={`/services/${saved.id}`} className="rounded-lg border border-slate-300 bg-white px-4 py-2">
              Servise git
            </Link>
            <Link href="/changes" className="rounded-lg border border-slate-300 bg-white px-4 py-2">
              Sorun varsa geri al
            </Link>
          </div>
        </div>
      ) : null}
      {bootstrap && step < 5 ? (
        <div className="mt-6 flex flex-wrap gap-2">
          {step > 1 ? (
            <SecondaryButton type="button" disabled={pending} onClick={() => setStep(step === 3 ? 2 : step - 1)}>
              Geri
            </SecondaryButton>
          ) : null}
          {step < 3 ? (
            <PrimaryButton type="button" onClick={next}>
              Devam
            </PrimaryButton>
          ) : null}
          {step === 3 ? (
            <PrimaryButton type="button" disabled={pending || !check?.valid} onClick={() => setStep(4)}>
              Devam
            </PrimaryButton>
          ) : null}
          {step === 4 ? (
            <PrimaryButton type="button" disabled={pending || (mode === "one" && !node)} onClick={() => publish(mode === "one" ? node : "")}>
              {pending ? "Yayına alınıyor…" : "Yayına al"}
            </PrimaryButton>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

function Field({ id, label, hint, value, onChange, placeholder }: { id: string; label: string; hint: string; value: string; onChange: (value: string) => void; placeholder?: string }) {
  return (
    <label className="block" htmlFor={id}>
      <span className="block font-medium">{label}</span>
      <span className="block text-sm text-slate-600">{hint}</span>
      <input id={id} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} className="mt-2 block w-full rounded-lg border border-slate-300 px-3 py-2" />
    </label>
  );
}
