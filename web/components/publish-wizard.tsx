"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { TryPanel } from "@/components/try-panel";
import { ErrorText, Panel, PageTitle, PrimaryButton, SecondaryButton, Success } from "@/components/ui";
import { api } from "@/lib/api";
import type { Bootstrap, DiffChange, Overview, Route, Service } from "@/lib/types";
import { cleanTarget, friendlyError, protocolText, slug } from "@/lib/words";

const STEPS = ["Adres", "Hedef", "Kontrol", "Yayına al", "Dene"];
const METHODS = ["GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"];

type Check = { valid: boolean; errors: string[]; changes: DiffChange[] };
export function PublishWizard({ serviceId }: { serviceId?: string }) {
  const router = useRouter();
  const [bootstrap, setBootstrap] = useState<Bootstrap | null>(null);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [step, setStep] = useState(1);
  const [hostname, setHostname] = useState("");
  const [owner, setOwner] = useState("");
  const [address, setAddress] = useState("");
  const [port, setPort] = useState("");
  const [path, setPath] = useState("/");
  const [listenerId, setListenerId] = useState("");
  const [matchType, setMatchType] = useState("prefix");
  const [methods, setMethods] = useState<string[]>([]);
  const [advanced, setAdvanced] = useState(false);
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
      setHostname(item.hostnames[0] ?? "");
      setOwner(item.owner);
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
      name: loaded?.name ?? slug(hostname, "servis"),
      owner: owner.trim(),
      importance: loaded?.importance ?? "medium",
      description: loaded?.description ?? "",
      environmentId: bootstrap!.environmentId,
      teamId: bootstrap!.teamId,
      fleetId: bootstrap!.fleetId,
      listenerId: listener!.id,
      hostnames: [hostname.trim().toLowerCase()],
      expectedVersion: saved?.version ?? 0,
      routes: [
        {
          path: routePath,
          matchType: layer4 ? "prefix" : matchType,
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
      const result = await api<{ valid: boolean; errors: string[]; service: Service }>(`/api/v1/services/${item.id}/validate`, { method: "POST" });
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
      if (!hostname.trim().includes(".")) return setError("Adres en az bir nokta içermeli. Örnek: magaza.local");
      if (!owner.trim()) return setError("Sorumlu kişi veya ekip boş olamaz.");
    }
    if (step === 2) {
      const target = cleanTarget(address, port);
      const number = Number(target.port);
      if (!target.address) return setError("Hedef sunucu adresini yazın. Örnek: backend-a veya 10.0.0.5");
      if (!Number.isInteger(number) || number < 1 || number > 65535) return setError("Port 1 ile 65535 arasında bir sayı olmalı. Örnek: 8080");
      if (!layer4 && !path.startsWith("/")) return setError("Yol / ile başlamalı. Tüm site için yalnızca / yazın.");
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
          <li key={label} className={`rounded-full px-3 py-1 ${index + 1 === step ? "bg-blue-600 text-white" : index + 1 < step ? "bg-blue-100 text-blue-900" : "bg-slate-200 text-slate-600"}`}>
            {index + 1}. {label}
          </li>
        ))}
      </ol>
      {error ? <ErrorText>{error}</ErrorText> : null}
      {!bootstrap && !error ? <p>Yükleniyor…</p> : null}
      {bootstrap && step === 1 ? (
        <Panel className="space-y-4">
          <Field id="hostname" label="İnsanlar hangi adresi yazacak?" hint="Alan adı. Örnek: magaza.local veya api.sirket.com" value={hostname} onChange={setHostname} placeholder="magaza.local" />
          <Field id="owner" label="Bu servisten kim sorumlu?" hint="Sorun olunca kime sorulacağı. Örnek: ödeme ekibi" value={owner} onChange={setOwner} placeholder="ödeme ekibi" />
        </Panel>
      ) : null}
      {bootstrap && step === 2 ? (
        <Panel className="space-y-4">
          <p className="text-sm text-stone-600">
            <strong>{hostname}</strong> adresine gelen istekler hangi sunucuya gitsin?
          </p>
          <div className="grid gap-4 sm:grid-cols-[1fr_140px]">
            <Field id="address" label="Hedef sunucu" hint="Ad veya IP. http:// yazmanıza gerek yok." value={address} onChange={setAddress} placeholder="backend-a" />
            <Field id="port" label="Port" hint="Genelde 80 veya 8080." value={port} onChange={setPort} placeholder="8080" />
          </div>
          {!layer4 ? <Field id="path" label="Hangi yol?" hint="Tüm site için / bırakın. Yalnızca /api ile başlayanlar için /api yazın." value={path} onChange={setPath} placeholder="/" /> : null}
          {extra.length > 0 ? <p className="text-sm text-stone-600">Bu serviste {extra.length} yol daha var. Onlar olduğu gibi kalır.</p> : null}
          <button type="button" className="text-sm underline" onClick={() => setAdvanced(!advanced)}>
            {advanced ? "Gelişmiş ayarları gizle" : "Gelişmiş ayarlar"}
          </button>
          {advanced ? (
            <div className="space-y-4 rounded-xl bg-stone-50 p-4">
              <label className="block text-sm" htmlFor="protocol">
                Bağlantı türü
                <select id="protocol" value={listenerId} onChange={(event) => setListenerId(event.target.value)} className="mt-1 block w-full rounded-lg border border-stone-300 bg-white px-3 py-2">
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
                  <select id="match" value={matchType} onChange={(event) => setMatchType(event.target.value)} className="mt-1 block w-full rounded-lg border border-stone-300 bg-white px-3 py-2">
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
              {check.changes.length === 0 ? <p className="mt-1 text-sm text-stone-600">Hiçbir şey değişmeyecek; yayındaki ayar zaten bu.</p> : null}
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
          <label className="flex gap-3 rounded-xl border border-stone-200 p-3">
            <input type="radio" name="mode" checked={mode === "all"} onChange={() => setMode("all")} />
            <span>
              <span className="block font-medium">Tüm sunucular</span>
              <span className="text-sm text-stone-600">Herkes yeni ayarı hemen kullanır.</span>
            </span>
          </label>
          <label className="flex gap-3 rounded-xl border border-stone-200 p-3">
            <input type="radio" name="mode" checked={mode === "one"} onChange={() => setMode("one")} />
            <span>
              <span className="block font-medium">Önce tek sunucuda dene</span>
              <span className="text-sm text-stone-600">Sorun çıkarsa yalnızca o sunucu etkilenir. Sonra tek tuşla hepsine açarsınız.</span>
            </span>
          </label>
          {mode === "one" ? (
            <select aria-label="Denenecek sunucu" value={node} onChange={(event) => setNode(event.target.value)} className="block rounded-lg border border-stone-300 px-3 py-2">
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
          <Success>{mode === "one" ? `Yalnızca ${node} sunucusunda yayında. Diğer sunucular eski ayarla çalışmaya devam ediyor.` : "Tüm sunucularda yayında."}</Success>
          <TryPanel hostname={hostname} path={routePath} tryUrl={bootstrap.tryUrl} revision={revision} web={protocol === "http"} />
          <div className="flex flex-wrap gap-2">
            {mode === "one" ? (
              <PrimaryButton type="button" disabled={pending} onClick={() => publish("")}>
                Sorun yok, tüm sunuculara aç
              </PrimaryButton>
            ) : null}
            <Link href={`/services/${saved.id}`} className="rounded-lg border border-stone-300 bg-white px-4 py-2">
              Servise git
            </Link>
            <Link href="/changes" className="rounded-lg border border-stone-300 bg-white px-4 py-2">
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
      <span className="block text-sm text-stone-600">{hint}</span>
      <input id={id} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} className="mt-2 block w-full rounded-lg border border-stone-300 px-3 py-2" />
    </label>
  );
}
