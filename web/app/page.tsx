"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Dot, ErrorText, MetricCard, Notice, PageTitle } from "@/components/ui";
import { Icon } from "@/components/icon";
import { ServiceTable } from "@/components/service-table";
import { api } from "@/lib/api";
import type { Overview, Revision, Service } from "@/lib/types";
import { friendlyError, nodeState } from "@/lib/words";
import {useLanguage} from "@/lib/i18n";
import { formatTime, useTimeZone } from "@/lib/format";
export default function HomePage() {
  const router=useRouter();
  const {t}=useLanguage();
  const {timeZone}=useTimeZone();
  const [data,setData]=useState<Overview|null>(null);
  const [services,setServices]=useState<Service[]|null>(null);
  const [revisions,setRevisions]=useState<Revision[]>([]);
  const [error,setError]=useState("");
  const [loadedAt,setLoadedAt]=useState<string|null>(null);
  useEffect(()=>{
    if(!sessionStorage.getItem("gateway.token")){router.replace("/login");return;}
    let stop=false;
    async function load(){try{const [view,list,changes]=await Promise.all([api<Overview>("/api/v1/overview"),api<{items:Service[]}>("/api/v1/services?limit=5"),api<{items:Revision[]}>("/api/v1/revisions?limit=4")]);if(!stop){setData(view);setServices(list.items);setRevisions(changes.items);setLoadedAt(new Date().toISOString());setError("");}}catch(err){if(!stop)setError(friendlyError(err,"Veriler güncellenemedi. Son alınan durum gösteriliyor."));}}
    load();const timer=setInterval(load,5000);return()=>{stop=true;clearInterval(timer);};
  },[router]);
  const fresh=data?.nodes.filter(node=>node.heartbeat==="fresh").length ?? 0;
  const rejected=data?.nodes.filter(node=>node.nackRevision!=null && node.nackRevision===node.desiredRevision) ?? [];
  return <section>
    <div className="flex flex-wrap items-center justify-between gap-5"><div><p className="eyebrow mb-2 text-teal-700">{t("Operasyon merkezi")}</p><PageTitle title={t("Gateway genel bakış")} text="Servisleriniz, yayınlarınız ve trafik düğümleriniz. Hepsi bir arada." /></div><Link href="/services/new" className="btn-primary"><Icon name="plus" size={16} />{t("Yeni servis")}</Link></div>
    {error && <div className="mt-5"><ErrorText>{error}</ErrorText></div>}
    <div className="metric-grid"><MetricCard label={t("Yayındaki servisler")} value={data?.services.active} detail={data ? `${data.services.total} kayıtlı servisten yayımlananlar` : "Servisler sorgulanıyor"} icon="layers" /><MetricCard label={t("Bağlı düğümler")} value={data ? <>{fresh}<span className="ml-1 text-xl font-normal text-slate-300">/ {data.nodes.length}</span></> : null} detail="Güncel heartbeat gönderen düğümler" icon="nodes" /><MetricCard label={t("Aktif revizyon")} value={data ? (data.activeRevision!=null ? `r${data.activeRevision}` : "—") : null} detail="Merkezdeki son konfigürasyon sürümü" icon="history" /><MetricCard label={t("Bekleyen taslaklar")} value={data?.pendingDrafts} detail="Henüz yayına alınmamış değişiklikler" icon="clock" /></div>
    <div className="dashboard-grid"><div className="space-y-5"><div className="gc-panel overflow-hidden"><div className="panel-heading"><div><h2>Servis rotaları</h2><p className="mt-1 text-[11px] text-slate-400">Alan adından hedef servise</p></div><Link href="/services" className="flex items-center gap-1 text-[11px] font-medium text-teal-700">Tüm servisler<Icon name="arrow" size={14} /></Link></div><ServiceTable items={services} compact /><div className="flow-strip"><span className="flow-step"><Icon name="globe" size={14} />İstemci</span><span className="flow-arrow" /><span className="flow-step"><Icon name="shield" size={14} />HTTPS girişi</span><span className="flow-arrow" /><span className="flow-step"><Icon name="layers" size={14} />Envoy</span><span className="flow-arrow" /><span className="flow-step"><Icon name="nodes" size={14} />Backend</span></div></div>
    <div className="gc-panel overflow-hidden"><div className="panel-heading"><h2>Gateway düğümleri</h2><Link href="/nodes" className="text-[11px] text-teal-700">Ayrıntıları gör →</Link></div><div className="grid gap-0 divide-y divide-slate-100 sm:grid-cols-2 sm:divide-x sm:divide-y-0">{data?.nodes.map(node=>{const state=nodeState(node);return <div key={node.id} className="p-5"><div className="flex items-center justify-between"><span className="flex items-center gap-2 text-xs font-semibold"><Icon name="nodes" size={16} className="text-slate-400" />{node.name}</span><span className={`status-pill status-${state.tone}`}>{node.heartbeat==="fresh" ? "Bağlı" : "Güncel değil"}</span></div><p className="mt-4 flex items-center text-[11px] text-slate-500"><Dot tone={state.tone} />{state.text}</p><div className="mt-4 flex justify-between text-[10px] text-slate-400"><span>İstenen <strong className="mono ml-1 font-medium text-slate-600">r{node.desiredRevision??"—"}</strong></span><span>Onaylanan <strong className="mono ml-1 font-medium text-slate-600">r{node.ackedRevision??"—"}</strong></span></div></div>;})}{!data && <div className="p-6 text-xs text-slate-400">Düğüm bilgileri yükleniyor…</div>}</div></div></div>
    <div className="space-y-5"><div className="rounded-xl border border-[#27434c] bg-[#142b34] p-6 text-white"><div className="flex items-center justify-between"><span className="eyebrow text-teal-300">Test laboratuvarı</span><Icon name="lab" className="text-teal-300" size={22} /></div><h2 className="mt-5 text-xl font-medium tracking-tight">Yönlendirmeyi<br />gerçek isteklerle doğrula.</h2><p className="mt-3 text-xs leading-6 text-slate-400">Public uç noktaları kontrol et; HTTP durumunu, yanıt süresini ve cevap veren backend’i gör.</p><Link href="/test-lab" className="mt-6 flex items-center justify-between rounded-md border border-slate-500/50 px-3 py-2.5 text-xs text-teal-100 hover:bg-slate-600/30">Test ortamını aç<Icon name="arrow" size={16} /></Link></div>
    <div className="gc-panel"><div className="panel-heading"><h2>Son yayınlar</h2><Icon name="history" size={16} className="text-slate-400" /></div><div className="px-5">{revisions.map((revision,i)=><Link key={revision.id} href={`/changes/${revision.id}`} className="flex gap-3 border-b border-slate-100 py-4 last:border-0"><span className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${i===0?"bg-teal-500":"bg-slate-300"}`} /><div className="min-w-0"><p className="text-xs font-semibold">Revizyon {revision.id}<span className="ml-2 text-[9px] font-normal text-slate-400">{revision.createdBy}</span></p><p className="mt-1.5 truncate text-[11px] text-slate-500">{revision.summary}</p><p className="mt-2 text-[9px] text-slate-400">{formatTime(revision.createdAt,timeZone)}</p></div></Link>)}{!revisions.length && <p className="py-7 text-xs text-slate-400">Henüz yayın kaydı yok.</p>}</div></div></div></div>
    <div className="mt-5 space-y-3">{rejected.map(node=><ErrorText key={node.id}>{node.name} son sürümü reddetti: {node.nackDetail}</ErrorText>)}{data && fresh<data.nodes.length && <Notice>Bazı gateway düğümlerinden güncel heartbeat alınamıyor. Düğüm durumlarını kontrol edin.</Notice>}</div>
    <p className="mt-5 flex flex-wrap items-center gap-2 text-[10px] text-slate-400"><Icon name="refresh" size={12} />5 saniyede bir yenilenir{loadedAt && <span>· Son güncelleme {formatTime(loadedAt,timeZone)}</span>}<span className="ml-auto">Trafik metrikleri henüz bağlı değil.</span></p>
  </section>;
}
