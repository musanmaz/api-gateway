"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ErrorText, Notice, PageTitle, Panel } from "@/components/ui";
import { api } from "@/lib/api";
import type { Bootstrap } from "@/lib/types";
import { friendlyError, protocolText } from "@/lib/words";

const localAddress: Record<string, string> = {
  http: "http://127.0.0.1:10000",
  https: "https://127.0.0.1:18443",
  http2: "127.0.0.1:10012",
  websocket: "127.0.0.1:10013",
  grpc: "127.0.0.1:10014",
  sse: "127.0.0.1:10015",
  tcp: "127.0.0.1:10020",
  udp: "127.0.0.1:10021",
  tls_passthrough: "127.0.0.1:18444",
};

export default function CertificatesPage() {
  const router = useRouter();
  const [data, setData] = useState<Bootstrap | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!sessionStorage.getItem("gateway.token")) {
      router.replace("/login");
      return;
    }
    api<Bootstrap>("/api/v1/bootstrap")
      .then(setData)
      .catch((err: unknown) => setError(friendlyError(err, "Dinleyiciler alınamadı")));
  }, [router]);

  return (
    <section>
      <PageTitle title="Sertifikalar ve dinleyiciler" text="Her bağlantı türünün kendi kapısı vardır. HTTPS kapısı yerel geliştirme sertifikasını kullanır." />
      {error ? <ErrorText>{error}</ErrorText> : null}
      {!data && !error ? <p>Yükleniyor…</p> : null}
      {data ? (
        <div className="space-y-4">
          <Panel className="space-y-2">
            <h2 className="text-lg font-medium">Yerel sertifika</h2>
            <p className="text-sm text-slate-600">
              <code>make bootstrap</code> sertifikayı <code>.local/certs/server.crt</code> dosyasına yazar. Adı <code>*.test.local</code> biçimindedir. Tarayıcı bu sertifikaya güvenmez; bu beklenen durumdur. Özel anahtar panelde gösterilmez.
            </p>
            <p className="text-sm text-slate-600">Sertifikanın kime ait olduğunu ve bitiş tarihini görmek için proje klasöründe:</p>
            <code className="block overflow-x-auto rounded-lg bg-slate-900 px-3 py-2 text-sm text-slate-100">openssl x509 -in .local/certs/server.crt -noout -subject -dates</code>
            <Notice>Bu genel bir sertifika otoritesi değildir. Üretim sertifikası buradan bağlanmaz.</Notice>
          </Panel>
          <div className="grid gap-3 sm:grid-cols-2">
            {data.listeners.map((item) => {
              const text = protocolText[item.protocol];
              const state = data.protocols.find((protocol) => protocol.id === item.protocol)?.state;
              return (
                <article key={item.id} className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
                  <div className="flex items-start justify-between gap-3">
                    <h2 className="font-medium">{text?.title ?? item.protocol}</h2>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${state === "available" ? "bg-teal-100 text-teal-950" : "bg-slate-200 text-slate-700"}`}>
                      {state === "available" ? "açık" : "kapalı"}
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-slate-600">{text?.text}</p>
                  <p className="mt-3 text-sm">
                    Bu makineden: <span className="font-medium">{localAddress[item.protocol] ?? "adres yok"}</span>
                  </p>
                  <p className="text-xs text-slate-500">
                    {item.name} · konteyner portu {item.port}
                  </p>
                </article>
              );
            })}
          </div>
        </div>
      ) : null}
    </section>
  );
}
