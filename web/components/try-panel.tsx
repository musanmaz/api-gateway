"use client";

import { useEffect, useState } from "react";
import { Dot, Panel } from "@/components/ui";
import { api } from "@/lib/api";
import type { Overview } from "@/lib/types";
import { nodeState, probeText } from "@/lib/words";

export function TryPanel({ hostname, path, tryUrl, revision, web }: { hostname: string; path: string; tryUrl: string; revision?: number | null; web: boolean }) {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let stop = false;
    async function load() {
      try {
        const next = await api<Overview>("/api/v1/overview");
        if (!stop) setOverview(next);
      } catch {
        // The next tick retries; the panel keeps the last known state.
      }
    }
    load();
    const timer = setInterval(load, 2000);
    return () => {
      stop = true;
      clearInterval(timer);
    };
  }, []);

  const publicHTTPS = tryUrl.startsWith("https://");
  const publicURL = `https://${hostname}${path}`;
  const command = publicHTTPS ? `curl -sS '${publicURL}'` : `curl -H 'Host: ${hostname}' '${tryUrl || "http://GATEWAY_ADRESI"}${path}'`;

  return (
    <Panel className="space-y-4">
      <div>
        <h2 className="text-lg font-medium">Dene</h2>
        <p className="mt-1 text-sm text-stone-600">Sunucular yeni sürümü birkaç saniye içinde alır. Bu liste kendiliğinden yenilenir.</p>
      </div>
      {!overview ? <p className="text-sm">Sunucular kontrol ediliyor…</p> : null}
      <ul className="space-y-3">
        {overview?.nodes.map((node) => {
          const state = nodeState(node, revision);
          const probe = node.probes.find((item) => item.host === hostname && item.path === path) ?? node.probes.find((item) => item.host === hostname);
          const result = probe ? probeText(probe) : null;
          return (
            <li key={node.id} className="rounded-xl border border-stone-200 p-3">
              <p className="font-medium">{node.name}</p>
              <p className="mt-1 text-sm">
                <Dot tone={state.tone} />
                {state.text}
              </p>
              {web && state.tone === "ok" ? (
                <p className="mt-1 text-sm">
                  <Dot tone={result?.tone ?? "wait"} />
                  {result ? `${hostname}${probe?.path}: ${result.text}` : "İlk deneme bekleniyor…"}
                </p>
              ) : null}
            </li>
          );
        })}
      </ul>
      {web ? (
        <div className="rounded-xl bg-stone-50 p-3 text-sm">
          <p className="font-medium">Kendi bilgisayarınızdan denemek için</p>
          <p className="mt-1 text-stone-600">
            {publicHTTPS ? <>Alan adınız gateway girişine çözülüyorsa <a className="text-teal-700 underline" href={publicURL} target="_blank" rel="noreferrer">HTTPS adresini açın</a> veya komutu çalıştırın. TLS doğrulaması alan adıyla yapılır.</> : <>Terminalde Host başlığını koruyan bu komutu kullanın. Gateway IP’sini doğrudan açmak, tanımlı alan adı gönderilmediği için 404 döndürebilir.</>}
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <code className="block overflow-x-auto rounded-lg bg-stone-900 px-3 py-2 text-stone-100">{command}</code>
            <button
              type="button"
              className="rounded-lg border border-stone-300 bg-white px-3 py-1"
              onClick={async () => {
                await navigator.clipboard.writeText(command);
                setCopied(true);
                setTimeout(() => setCopied(false), 1500);
              }}
            >
              {copied ? "Kopyalandı" : "Kopyala"}
            </button>
          </div>
          {!tryUrl ? <p className="mt-2 text-stone-600">Gateway adresi yapılandırılmamış. Yöneticiniz API için GATEWAY_TRY_URL değerini ayarlayabilir.</p> : null}
        </div>
      ) : (
        <p className="text-sm text-stone-600">Bu bağlantı türünde otomatik deneme yok. Sunucu durumu yukarıdadır.</p>
      )}
    </Panel>
  );
}
