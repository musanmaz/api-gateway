"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Dot, ErrorText, PageTitle } from "@/components/ui";
import { api } from "@/lib/api";
import { formatTime, useTimeZone } from "@/lib/format";
import type { Overview } from "@/lib/types";
import { friendlyError, nodeState, probeText } from "@/lib/words";

export default function SystemPage() {
  const router = useRouter();
  const { timeZone } = useTimeZone();
  const [data, setData] = useState<Overview | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!sessionStorage.getItem("gateway.token")) {
      router.replace("/login");
      return;
    }
    let stop = false;
    async function load() {
      try {
        const next = await api<Overview>("/api/v1/overview");
        if (!stop) setData(next);
      } catch (err) {
        if (!stop) setError(friendlyError(err, "Sunucular alınamadı"));
      }
    }
    load();
    const timer = setInterval(load, 5000);
    return () => {
      stop = true;
      clearInterval(timer);
    };
  }, [router]);

  return (
    <section>
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <PageTitle title="Sistem" text="Trafiği taşıyan sunucular ve son durumları." />
        <Link href="/certificates" className="text-sm underline">
          Sertifikalar
        </Link>
      </div>
      {error ? <ErrorText>{error}</ErrorText> : null}
      {!data && !error ? <p>Yükleniyor…</p> : null}
      <div className="space-y-3">
        {data?.nodes.map((node) => {
          const state = nodeState(node);
          return (
            <article key={node.id} className="rounded-2xl border border-stone-200 bg-white p-4">
              <h2 className="text-lg font-medium">{node.name}</h2>
              <p className="mt-1">
                <Dot tone={state.tone} />
                {state.text}
              </p>
              <p className="mt-1 text-sm text-stone-600">
                {node.location ? `${node.location} · ` : ""}Son haber: {formatTime(node.lastHeartbeatAt, timeZone)} · Kullandığı sürüm: {node.ackedRevision ?? "yok"}
              </p>
              {node.probes.length > 0 ? (
                <ul className="mt-3 space-y-1 text-sm">
                  {node.probes.map((probe) => {
                    const result = probeText(probe);
                    return (
                      <li key={`${probe.host}${probe.path}`}>
                        <Dot tone={result.tone} />
                        {probe.host}
                        {probe.path}: {result.text}
                      </li>
                    );
                  })}
                </ul>
              ) : null}
            </article>
          );
        })}
      </div>
      <p className="mt-6 text-sm text-stone-600">Kayıt (log), alarm ve CPU/bellek ölçümleri bu sürümde bağlı değil.</p>
    </section>
  );
}
