"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ErrorText, PageTitle, Panel, PrimaryButton, Success } from "@/components/ui";
import { api } from "@/lib/api";
import { formatTime, useTimeZone } from "@/lib/format";
import type { Revision } from "@/lib/types";
import { friendlyError } from "@/lib/words";

export default function ChangeDetailPage() {
  const params = useParams<{ id: string }>();
  const { timeZone } = useTimeZone();
  const [item, setItem] = useState<Revision | null>(null);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  useEffect(() => {
    api<Revision>(`/api/v1/revisions/${params.id}`)
      .then(setItem)
      .catch((err: unknown) => setError(friendlyError(err, "Sürüm alınamadı")));
  }, [params.id]);
  let pretty = item?.payload ?? "";
  if (item?.payload) {
    try {
      pretty = JSON.stringify(JSON.parse(item.payload), null, 2);
    } catch {
      pretty = item.payload;
    }
  }

  async function rollback() {
    if (!item || !window.confirm(`Tüm sunucular sürüm ${item.id} ayarına dönsün mü?`)) return;
    setPending(true);
    setError("");
    try {
      const result = await api<{ revision: Revision }>(`/api/v1/revisions/${item.id}/rollback`, { method: "POST" });
      setMessage(`Tüm sunuculara sürüm ${item.id} ayarı gönderildi (yeni sürüm ${result.revision.id}). Servis listesindeki kayıtlar değişmez; bir sonraki yayın onları yeniden kullanır.`);
    } catch (err) {
      setError(friendlyError(err, "Geri dönülemedi"));
    } finally {
      setPending(false);
    }
  }

  return (
    <section>
      <PageTitle title={item ? `Sürüm ${item.id}` : "Sürüm"} text={item ? `${item.createdBy} · ${formatTime(item.createdAt, timeZone)}` : undefined} />
      {error ? <ErrorText>{error}</ErrorText> : null}
      {!item && !error ? <p>Yükleniyor…</p> : null}
      {item ? (
        <div className="space-y-4">
          <Panel className="space-y-3">
            <p>{item.summary}</p>
            {message ? (
              <Success>
                {message}{" "}
                <Link href="/nodes" className="underline">
                  Sunucuların aldığını kontrol edin.
                </Link>
              </Success>
            ) : (
              <PrimaryButton type="button" disabled={pending} onClick={rollback}>
                {pending ? "Gönderiliyor…" : "Bu sürüme geri dön"}
              </PrimaryButton>
            )}
          </Panel>
          <details className="rounded-2xl border border-stone-200 bg-white p-4 text-sm">
            <summary className="cursor-pointer">Teknik ayrıntılar</summary>
            <p className="mt-3 text-stone-600">Önceki sürüm: {item.parentRevisionId ?? "yok"}</p>
            <p className="mt-1 break-all text-xs text-stone-500">İçerik özeti: {item.contentHash}</p>
            <pre className="mt-3 overflow-auto rounded-lg bg-stone-950 p-3 text-xs text-stone-100">{pretty}</pre>
          </details>
        </div>
      ) : null}
    </section>
  );
}
