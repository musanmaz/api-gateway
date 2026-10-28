"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ErrorText, PageTitle } from "@/components/ui";
import { Icon } from "@/components/icon";
import { ServiceTable } from "@/components/service-table";
import { api } from "@/lib/api";
import type { Service } from "@/lib/types";
import { friendlyError } from "@/lib/words";
import {useLanguage} from "@/lib/i18n";
export default function ServicesPage() {
  const router=useRouter();
  const {t}=useLanguage();
  const [items,setItems]=useState<Service[]|null>(null);
  const [query,setQuery]=useState("");
  const [status,setStatus]=useState("all");
  const [error,setError]=useState("");
  const [page,setPage]=useState(0);
  const [total,setTotal]=useState(0);
  useEffect(()=>{if(!sessionStorage.getItem("gateway.token"))router.replace("/login");},[router]);
  useEffect(()=>{let stop=false;const timer=setTimeout(()=>{api<{items:Service[];total:number}>(`/api/v1/services?limit=20&offset=${page*20}&q=${encodeURIComponent(query)}`).then(result=>{if(!stop){setItems(result.items);setTotal(result.total);setError("");}}).catch(err=>{if(!stop)setError(friendlyError(err,"Servisler alınamadı"));});},180);return()=>{stop=true;clearTimeout(timer);};},[query,page]);
  const filtered=items?.filter(item=>status==="all"||item.status===status)??null;
  return <section><div className="mb-7 flex flex-wrap items-center justify-between gap-4"><PageTitle title={t("Servisler")} text="Alan adları, hedef sunucular ve yayın durumları." /><Link href="/services/new" className="btn-primary"><Icon name="plus" size={16} />{t("Yeni servis")}</Link></div>{error && <div className="mb-4"><ErrorText>{error}</ErrorText></div>}<div className="gc-panel overflow-hidden"><div className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-100 p-5"><div className="flex gap-1 rounded-lg bg-slate-100 p-1" role="group" aria-label="Yayın durumu">{[{id:"all",label:t("Tüm servisler")},{id:"active",label:t("Yayımlanan")},{id:"draft",label:t("Taslak")}].map(tab=><button key={tab.id} onClick={()=>setStatus(tab.id)} aria-pressed={status===tab.id} className={`rounded-md px-3 py-2 text-[11px] font-medium ${status===tab.id?"bg-white text-slate-800 shadow-sm":"text-slate-500 hover:text-slate-800"}`}>{tab.label}</button>)}</div><label className="relative"><Icon name="search" className="absolute left-3 top-3 text-slate-400" size={15} /><input aria-label="Servis ara" value={query} onChange={e=>{setQuery(e.target.value);setPage(0);}} placeholder="Alan adı, servis veya sorumlu ara…" className="w-72 max-w-full rounded-lg border border-slate-200 py-2 pl-9 pr-3 text-xs" /></label></div>{filtered?.length===0 && items?.length ? <p className="p-10 text-center text-xs text-slate-500">Bu sayfada seçili duruma uygun servis yok.</p> : <ServiceTable items={filtered} />}<div className="flex items-center justify-between border-t border-slate-100 px-5 py-4 text-[10px] text-slate-400"><span>{items ? `${Number.isFinite(total)?total:items.length} kayıt · Sayfa ${page+1}` : "Yükleniyor…"}</span><div className="flex items-center gap-3"><button disabled={page===0} className="disabled:opacity-30" onClick={()=>setPage(page-1)}>← Önceki</button><button disabled={!items || items.length<20} className="disabled:opacity-30" onClick={()=>setPage(page+1)}>Sonraki →</button></div></div></div><p className="mt-4 text-[11px] text-slate-400">Yayın durumu konfigürasyonu belirtir. Canlı yanıtları Test laboratuvarı ekranından kontrol edebilirsiniz.</p></section>;
}
