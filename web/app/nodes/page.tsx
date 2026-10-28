"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Dot, ErrorText, Notice, PageTitle } from "@/components/ui";
import { Icon } from "@/components/icon";
import { api } from "@/lib/api";
import { formatTime,useTimeZone } from "@/lib/format";
import type { Overview } from "@/lib/types";
import { friendlyError,nodeState,probeText } from "@/lib/words";
export default function SystemPage(){
 const router=useRouter();const{timeZone}=useTimeZone();const[data,setData]=useState<Overview|null>(null);const[error,setError]=useState("");
 useEffect(()=>{if(!sessionStorage.getItem("gateway.token")){router.replace("/login");return;}let stop=false;async function load(){try{const next=await api<Overview>("/api/v1/overview");if(!stop){setData(next);setError("");}}catch(err){if(!stop)setError(friendlyError(err,"Düğüm bilgileri yenilenemedi. Son alınan durum gösteriliyor."));}}load();const timer=setInterval(load,5000);return()=>{stop=true;clearInterval(timer);};},[router]);
 return <section><div className="mb-7"><PageTitle title="Gateway düğümleri" text="Konfigürasyon dağıtımını ve düğümlerin verdiği yanıtları takip edin." /></div>{error&&<div className="mb-4"><ErrorText>{error}</ErrorText></div>}<div className="lab-grid">{data?.nodes.map(node=>{const state=nodeState(node);return <article key={node.id} className="gc-panel overflow-hidden"><div className="panel-heading"><div className="flex items-center gap-3"><span className="service-symbol"><Icon name="nodes" /></span><div><h2>{node.name}</h2><p className="mt-1 text-[10px] text-slate-400">{node.location} · Envoy data plane</p></div></div><span className={`status-pill status-${state.tone}`}>{node.heartbeat==="fresh"?"Bağlı":"Güncel değil"}</span></div><div className="p-6"><p className="text-xs text-slate-600"><Dot tone={state.tone}/>{state.text}</p><div className="my-6 grid grid-cols-3 divide-x divide-slate-200 rounded-lg border border-slate-200 bg-slate-50 p-4">{[{label:"İstenen",value:node.desiredRevision},{label:"Teslim edilen",value:node.deliveredRevision},{label:"ACK",value:node.ackedRevision}].map(item=><div key={item.label} className="text-center"><p className="text-[9px] uppercase tracking-wider text-slate-400">{item.label}</p><p className="mono mt-2 text-lg">{item.value!=null?`r${item.value}`:"—"}</p></div>)}</div><div className="mb-5 grid grid-cols-3 gap-2 rounded-lg border border-slate-200 bg-white p-3 text-xs">
  {[
    ["Toplam HTTP",node.telemetry?.httpRequests],
    ["2xx",node.telemetry?.http2xx],
    ["4xx",node.telemetry?.http4xx],
    ["5xx",node.telemetry?.http5xx],
    ["WAF eşleşme",node.telemetry?.wafMatches],
    ["WAF engel",node.telemetry?.wafBlocked],
  ].map(([label,value])=><div key={String(label)}><span className="block text-[10px] text-slate-400">{label}</span><strong className="mt-1 block text-lg">{value??0}</strong></div>)}
  <p className="col-span-4 text-[10px] text-slate-400">Envoy başlatıldığından beri işlenen HTTP istekleri. Düğüm yeniden başladığında sayaçlar sıfırlanır.</p>
</div><div className="mb-5 grid grid-cols-2 gap-3 rounded-lg border border-slate-200 bg-white p-3 text-xs">
  <div><span className="block text-slate-400">Yetkilendirme kontrolleri</span><strong className="mt-1 block text-lg">{node.telemetry?.checks ?? 0}</strong></div>
  <div><span className="block text-slate-400">Reddedilen doğrulamalar</span><strong className="mt-1 block text-lg">{node.telemetry?.denied ?? 0}</strong></div>
  <p className="col-span-2 text-[10px] text-slate-400">Agent başladığından beri geçen auth kontrolleri; HTTP başarı/hata ya da gecikme ölçümü değildir.</p>
</div><p className="eyebrow mb-3 text-slate-400">Son servis kontrolleri</p>{node.probes.length?node.probes.map(probe=>{const state=probeText(probe);return <div key={`${probe.host}${probe.path}`} className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 py-3 text-[11px] last:border-0"><span className="mono text-slate-600">{probe.host}{probe.path}</span><span className="text-slate-500"><Dot tone={node.heartbeat==="fresh"?state.tone:"off"}/>{node.heartbeat==="fresh"?state.text:"Sonuç güncel değil"}</span></div>;}):<p className="text-xs text-slate-400">Henüz bir servis kontrolü yok.</p>}<p className="mt-5 border-t border-slate-100 pt-4 text-[10px] text-slate-400">Son heartbeat: {formatTime(node.lastHeartbeatAt,timeZone)}</p></div></article>;})}{!data&&!error&&<div className="skeleton h-80"/>}</div><div className="mt-5"><Notice>{data?.nodes.some(n=>n.name==="gw-speedtest-1" && n.heartbeat==="fresh") ? "Uzak speedtest düğümü bağlı. Fiziksel failover ve ingress HA testleri ayrıca tamamlanmalıdır." : "gw-local-1 ve gw-local-2 aynı ai.mehmet.tech sunucusundadır. speedtest.mehmet.tech düğümü ajan bağlantısı tamamlanana kadar çevrimdışı görünecektir."}</Notice></div></section>;
}
