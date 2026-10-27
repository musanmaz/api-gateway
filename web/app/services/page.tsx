"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Badge, ErrorText, PageTitle } from "@/components/ui";
import { api } from "@/lib/api";
import type { Service } from "@/lib/types";
import { friendlyError } from "@/lib/words";

export default function ServicesPage() {
  const router = useRouter();
  const [items, setItems] = useState<Service[] | null>(null);
  const [query, setQuery] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!sessionStorage.getItem("gateway.token")) router.replace("/login");
  }, [router]);

  useEffect(() => {
    let stop = false;
    api<{ items: Service[] }>(`/api/v1/services?q=${encodeURIComponent(query)}`)
      .then((page) => {
        if (!stop) setItems(page.items);
      })
      .catch((err: unknown) => {
        if (!stop) setError(friendlyError(err, "Servisler alınamadı"));
      });
    return () => {
      stop = true;
    };
  }, [query]);

  return (
    <section>
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <PageTitle title="Servisler" text="Her satır: insanların yazdığı adres ve isteklerin gittiği sunucu." />
        <Link href="/services/new" className="rounded-lg bg-blue-600 px-4 py-2 font-medium text-white shadow-sm">
          Yeni site veya API
        </Link>
      </div>
      <label className="mb-4 block text-sm" htmlFor="search">
        Ara
        <input id="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="adres, ad veya sorumlu" className="mt-1 block w-full max-w-md rounded-lg border border-stone-300 px-3 py-2" />
      </label>
      {error ? <ErrorText>{error}</ErrorText> : null}
      {items === null && !error ? <p>Yükleniyor…</p> : null}
      {items && items.length === 0 ? (
        <p className="text-stone-600">
          {query ? "Aramanızla eşleşen servis yok." : "Henüz servis yok. "}
          {!query ? (
            <Link href="/services/new" className="underline">
              İlk servisi yayına alın.
            </Link>
          ) : null}
        </p>
      ) : null}
      {items && items.length > 0 ? (
        <ul className="space-y-3">
          {items.map((service) => {
            const endpoint = service.routes[0]?.upstream.endpoints[0];
            return (
              <li key={service.id} className="rounded-2xl border border-stone-200 bg-white p-4">
                <div className="flex flex-wrap items-center gap-2">
                  <Link href={`/services/${service.id}`} className="text-lg font-medium underline-offset-2 hover:underline">
                    {service.hostnames[0] ?? service.name}
                  </Link>
                  <Badge value={service.status} />
                </div>
                <p className="mt-1 text-sm text-stone-600">
                  {endpoint ? `İstekler ${endpoint.address}:${endpoint.port} sunucusuna gider.` : "Hedef sunucu yok."} Sorumlu: {service.owner}
                </p>
                {service.status === "draft" ? (
                  <Link href={`/services/new?service=${service.id}`} className="mt-2 inline-block text-sm font-medium text-blue-700 underline">
                    Yayına almaya devam et
                  </Link>
                ) : null}
              </li>
            );
          })}
        </ul>
      ) : null}
    </section>
  );
}
