"use client";
import {useEffect,useMemo,useState} from "react";
import {api} from "@/lib/api";
import {useLanguage} from "@/lib/i18n";
type Point={minute:number;requests:number;status2xx:number;status3xx:number;status4xx:number;status5xx:number;limited:number;wafMatches:number;wafBlocked:number;p95Ms:number;p99Ms:number};
type Report={serviceId:string;minutes:number;from:string;to:string;hasData:boolean;summary:Point;series:Point[]};
const fmt=(n:number)=>new Intl.NumberFormat("tr-TR",{maximumFractionDigits:1}).format(n);
export function ServiceTraffic({serviceId}:{serviceId:string}){
 const {language}=useLanguage();
 const [range,setRange]=useState(60);
 const [report,setReport]=useState<Report|null>(null);
 const [error,setError]=useState("");
 const [busy,setBusy]=useState(true);
 const tr=language==="tr";
 useEffect(()=>{
   let closed=false;
   async function load(){
     try{
       const result=await api<Report>("/api/v1/services/"+serviceId+"/traffic?minutes="+range);
       if(!closed){setReport(result);setError("");}
     }catch(err){if(!closed)setError(err instanceof Error?err.message:"API error")}
     finally{if(!closed)setBusy(false)}
   }
   setBusy(true);void load();
   const interval=setInterval(()=>void load(),15000);
   return ()=>{closed=true;clearInterval(interval)};
 },[serviceId,range]);
 const points=useMemo(()=>{
   if(!report)return [];
   const lookup=new Map(report.series.map(p=>[p.minute,p]));
   const end=Math.floor(new Date(report.to).getTime()/60000);
   const size=range===60?60:72;
   const step=range===60?1:20;
   return Array.from({length:size},(_,i)=>{
     const minute=end-(size-1-i)*step;
     let hits=0,failures=0;
     for(let j=0;j<step;j++){
       const p=lookup.get(minute-j);
       if(p){hits+=p.requests;failures+=p.status4xx+p.status5xx}
     }
     return {minute,hits,failures};
   });
 },[report,range]);
 const max=Math.max(1,...points.map(p=>p.hits));
 const avgRps=report?.summary?.requests ? report.summary.requests/(report.minutes*60) : 0;
 const tiles=[
   {name:tr?"Toplam istek":"Total requests",value:fmt(report?.summary?.requests??0)},
   {name:tr?"Ortalama RPS":"Avg RPS",value:fmt(avgRps)},
   {name:"2xx / 3xx",value:fmt((report?.summary?.status2xx??0)+(report?.summary?.status3xx??0))},
   {name:"4xx / 5xx",value:fmt((report?.summary?.status4xx??0)+(report?.summary?.status5xx??0))},
   {name:"429 "+(tr?"limit":"limited"),value:fmt(report?.summary?.limited??0)},
   {name:tr?"WAF eşleşme / engel":"WAF matched / blocked",value:fmt(report?.summary?.wafMatches??0)+" / "+fmt(report?.summary?.wafBlocked??0)},
   {name:"p95 "+(tr?"gecikme":"latency"),value:report?.hasData?"≤ "+fmt(report.summary.p95Ms)+" ms":"—"},
   {name:"p99 "+(tr?"gecikme":"latency"),value:report?.hasData?"≤ "+fmt(report.summary.p99Ms)+" ms":"—"},
 ];
 return <section className="space-y-4" aria-label={tr?"Servise özel trafik raporu":"Per-service traffic report"}>
   <div className="flex flex-wrap items-center justify-between gap-3">
     <div><h2 className="text-lg font-semibold">{tr?"Servise özel trafik ve güvenlik raporu":"Service traffic & security report"}</h2>
       <p className="mt-1 text-xs text-slate-500">{tr?"Envoy erişim kayıtlarından. Düğüm toplamı değil, yalnız bu servisin gerçek HTTP istekleri.":"From Envoy access logs. Only this service's HTTP requests, not total node traffic."}</p>
     </div>
     <select value={range} onChange={e=>setRange(Number(e.target.value))} aria-label={tr?"Rapor aralığı":"Report range"}
       className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs">
       <option value={60}>{tr?"Son 1 saat":"Last hour"}</option>
       <option value={1440}>{tr?"Son 24 saat":"Last 24 hours"}</option>
     </select>
   </div>
   {error?<p role="alert" className="rounded-lg border border-red-200 bg-red-50 p-3 text-xs text-red-700">{error}</p>:null}
   {busy&&!report?<p className="text-sm text-slate-500">{tr?"Metrikler yükleniyor…":"Loading metrics…"}</p>:null}
   {report&&!report.hasData?<p className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-xs text-slate-600">
      {tr?"Bu servis için henüz erişim kaydı yok. Yeni Envoy konfigürasyonu yayınlanıp ilk istek işlendiğinde rapor oluşur.":"No access records yet. Data appears after the new Envoy config is published and requests are processed."}
   </p>:null}
   <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
     {tiles.map(item=><div key={item.name} className="rounded-lg border border-slate-200 bg-white p-3">
       <p className="text-[11px] text-slate-500">{item.name}</p>
       <p className="mt-2 text-lg font-semibold tabular-nums">{item.value}</p>
     </div>)}
   </div>
   <div className="rounded-lg border border-slate-200 bg-white p-4">
     <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
       <h3 className="text-sm font-semibold">{tr?"Trafik geçmişi":"Traffic history"}</h3>
       <div className="flex gap-3 text-[11px] text-slate-600"><span><span className="mr-1 text-teal-600">●</span>{tr?"İstekler":"Requests"}</span><span><span className="mr-1 text-red-500">●</span>4xx+5xx</span></div>
     </div>
     <div className="flex h-32 items-end gap-[2px]" role="img" aria-label={tr?"İstek ve hata dağılımı":"Request and error distribution"}>
       {points.map(point=><div key={point.minute} className="group relative flex h-full min-w-0 flex-1 flex-col justify-end">
         <div className="relative w-full rounded-t-sm bg-teal-500/80" style={{height:String(100*point.hits/max)+"%",minHeight:point.hits?2:0}}>
           <div className="absolute bottom-0 w-full rounded-t-sm bg-red-500/80" style={{height:String(point.hits?100*point.failures/point.hits:0)+"%"}}/>
         </div>
         <span className="pointer-events-none absolute bottom-full left-0 z-10 hidden w-max rounded bg-slate-800 p-2 text-[10px] text-white group-hover:block">
           {new Date(point.minute*60000).toLocaleTimeString(language==="tr"?"tr-TR":"en-US",{hour:"2-digit",minute:"2-digit"})}: {point.hits} {tr?"istek":"requests"}, {point.failures} 4xx/5xx
         </span>
       </div>)}
     </div>
     <p className="mt-3 text-[11px] text-slate-500">{tr?"Dakikalık istek adetleri (24 saatte 20 dakikalık dilimler). 15 saniyede bir yenilenir.":"Request counts per minute (20-minute buckets for 24 hours). Refreshes every 15 seconds."}</p>
   </div>
   <p className="text-xs leading-5 text-slate-500">{tr
     ?"p95/p99, ölçüm aralığı üst sınırıyla yaklaşık olarak verilir. Geçmiş veriler kurulumdan itibaren oluşur; Envoy/agent kayıp kayıtları geriye dönük oluşturamaz. 429 yalnız Envoy kaynaklı limitleri kapsar."
     :"p95/p99 are approximate histogram upper bounds. History begins when collection starts; missing Envoy/agent logs cannot be recovered. 429 only covers gateway rate limits."}</p>
 </section>;
}
