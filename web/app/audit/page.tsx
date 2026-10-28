"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ErrorText, PageTitle } from "@/components/ui";
import { api } from "@/lib/api";
import { formatTime, useTimeZone } from "@/lib/format";
import type { AuditEvent } from "@/lib/types";
import { actionText, friendlyError } from "@/lib/words";

export default function AuditPage() {
  const router = useRouter();
  const { timeZone } = useTimeZone();
  const [items, setItems] = useState<AuditEvent[] | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    if (!sessionStorage.getItem("gateway.token")) router.replace("/login");
    api<{ items: AuditEvent[] }>("/api/v1/audit")
      .then((page) => setItems(page.items))
      .catch((err: unknown) => setError(friendlyError(err, "İşlem kaydı alınamadı")));
  }, [router]);
  return (
    <section>
      <PageTitle title="İşlem kaydı" text="Kim, ne zaman, ne yaptı." />
      {error ? <ErrorText>{error}</ErrorText> : null}
      {items === null && !error ? <p>Yükleniyor…</p> : null}
      {items && items.length === 0 ? <p className="text-slate-600">Henüz işlem yok.</p> : null}
      <ul className="space-y-2">
        {items?.map((item) => (
          <li key={item.id} className="rounded-2xl border border-slate-200 bg-white p-4 text-sm">
            <p>
              <span className="font-medium">{actionText[item.action] ?? item.action}</span> · {item.actor}
              {item.result !== "ok" ? ` · ${item.result}` : ""}
            </p>
            <p className="text-slate-600">
              {formatTime(item.createdAt, timeZone)} · {item.target}
            </p>
          </li>
        ))}
      </ul>
    </section>
  );
}
