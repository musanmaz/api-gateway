"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ErrorText, PageTitle } from "@/components/ui";
import { api } from "@/lib/api";
import { formatTime, useTimeZone } from "@/lib/format";
import type { Revision } from "@/lib/types";
import { friendlyError } from "@/lib/words";

export default function ChangesPage() {
  const router = useRouter();
  const { timeZone } = useTimeZone();
  const [items, setItems] = useState<Revision[] | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    if (!sessionStorage.getItem("gateway.token")) router.replace("/login");
    api<{ items: Revision[] }>("/api/v1/revisions")
      .then((page) => setItems(page.items))
      .catch((err: unknown) => setError(friendlyError(err, "Geçmiş alınamadı")));
  }, [router]);
  return (
    <section>
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <PageTitle title="Geçmiş" text="Her yayın bir sürüm olarak saklanır. Sorun çıkarsa bir sürüme tıklayıp ona geri dönebilirsiniz." />
        <Link href="/audit" className="text-sm underline">
          Ayrıntılı işlem kaydı
        </Link>
      </div>
      {error ? <ErrorText>{error}</ErrorText> : null}
      {items === null && !error ? <p>Yükleniyor…</p> : null}
      {items && items.length === 0 ? <p className="text-slate-600">Henüz hiçbir şey yayına alınmadı.</p> : null}
      <ul className="space-y-3">
        {items?.map((item, index) => (
          <li key={item.id}>
            <Link href={`/changes/${item.id}`} className="block rounded-2xl border border-slate-200 bg-white p-4 shadow-sm hover:border-teal-300">
              <span className="font-medium">
                Sürüm {item.id}
                {index === 0 ? " · en son" : ""}
              </span>
              <span className="mt-1 block">{item.summary}</span>
              <span className="mt-1 block text-sm text-slate-600">
                {item.createdBy} · {formatTime(item.createdAt, timeZone)}
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
