"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ErrorText, Notice, Success } from "@/components/ui";
import { api } from "@/lib/api";
import type { Overview } from "@/lib/types";
import { friendlyError, nodeState } from "@/lib/words";

const actions = [
  { href: "/services", title: "Yayındakileri gör", text: "Hangi adres nereye gidiyor, kim sorumlu, çalışıyor mu." },
  { href: "/changes", title: "Bir değişikliği geri al", text: "Sorun çıkarsa önceki sürüme tek tuşla dönün." },
];

export default function HomePage() {
  const router = useRouter();
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
        if (!stop) setError(friendlyError(err, "Durum alınamadı"));
      }
    }
    load();
    const timer = setInterval(load, 5000);
    return () => {
      stop = true;
      clearInterval(timer);
    };
  }, [router]);

  const states = data?.nodes.map((node) => nodeState(node)) ?? [];
  const down = states.filter((state) => state.tone === "bad").length;
  const rejected = data?.nodes.filter((node) => node.nackRevision != null && node.nackRevision === node.desiredRevision) ?? [];

  return (
    <section>
      <Link href="/services/new" className="block rounded-3xl bg-slate-900 px-8 py-10 text-white shadow-sm">
        <span className="text-sm text-slate-300">Yeni yayın</span>
        <span className="mt-2 block text-3xl font-semibold tracking-tight">Bir site veya API adresini yayına alın</span>
        <span className="mt-3 block max-w-xl text-slate-300">Adresi ve hedef sunucuyu yazın. Panel kontrol eder, sonra siz yayına alırsınız.</span>
      </Link>
      <div className="mt-4 grid gap-4 md:grid-cols-2">
        {actions.map((item) => (
          <Link key={item.href} href={item.href} className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm hover:border-slate-400">
            <span className="block text-lg font-medium">{item.title}</span>
            <span className="mt-2 block text-sm text-slate-600">{item.text}</span>
          </Link>
        ))}
      </div>
      <div className="mt-8 space-y-3">
        {error ? <ErrorText>{error}</ErrorText> : null}
        {!data && !error ? <p className="text-sm">Durum kontrol ediliyor…</p> : null}
        {data && down === 0 ? (
          <Success>
            {data.services.active} servis yayında, {data.nodes.length} sunucunun hepsi çalışıyor.
          </Success>
        ) : null}
        {data && down > 0 ? (
          <ErrorText>
            {data.nodes.length} sunucudan {down} tanesinde sorun var. <Link href="/nodes" className="underline">Sistem sayfasına bakın.</Link>
          </ErrorText>
        ) : null}
        {rejected.map((node) => (
          <ErrorText key={node.id}>
            {node.name} son sürümü reddetti ve önceki çalışan ayarla devam ediyor. {node.nackDetail}
          </ErrorText>
        ))}
        {data && data.pendingDrafts > 0 ? (
          <Notice>
            Yayına alınmamış {data.pendingDrafts} taslak var. <Link href="/services" className="underline">Servisler sayfasından devam edin.</Link>
          </Notice>
        ) : null}
      </div>
    </section>
  );
}
