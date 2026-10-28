"use client";
import Link from "next/link";
import type {ReactNode} from "react";
import type {Service,ServiceSecurityPolicy,ServiceSecurityStatus} from "@/lib/types";
import {useLanguage} from "@/lib/i18n";

function Tag({value,tone="neutral"}:{value:string;tone?:"safe"|"warning"|"neutral"}) {
 const style=tone==="safe"?"border-teal-200 bg-teal-50 text-teal-800":tone==="warning"?"border-amber-200 bg-amber-50 text-amber-800":"border-slate-200 bg-slate-50 text-slate-600";
 return <span className={"inline-flex items-center rounded-md border px-2 py-1 text-[11px] font-medium "+style}>{value}</span>;
}
function wafTitle(mode:string,tr:(s:string)=>string){
 return mode==="block"?tr("Engelle"):mode==="detect"?tr("İzle"):tr("Kapalı");
}
function quotaTitle(n:number,tr:(s:string)=>string){
 return n>0?String(n)+" "+tr("istek/sn/düğüm"):tr("Limitsiz");
}
export function ServiceSecurityTags({service}:{service:Service}){
 const {t}=useLanguage();
 const waf=service.wafMode??"off";
 const limit=(service.rateLimitRps??0)>0||service.routes.some(r=>(r.rateLimitRps??0)>0);
 const cidr=(service.ipAllowlist?.length??0)+(service.ipDenylist?.length??0);
 const pending=service.status==="active"&&service.version!==service.publishedVersion;
 return <div className="flex max-w-[300px] flex-wrap gap-1.5" aria-label={t("Kayıtlı güvenlik politikaları")}>
  <Tag value={"WAF: "+wafTitle(waf,t)} tone={waf==="block"?"safe":waf==="detect"?"warning":"neutral"}/>
  <Tag value={limit?t("Rate limit açık"):t("Kota yok")} tone={limit?"safe":"neutral"}/>
  <Tag value={service.authMode==="api_key"?"API Key":t("Public")} tone={service.authMode==="api_key"?"safe":"neutral"}/>
  {cidr>0?<Tag value={String(cidr)+" CIDR"} tone="safe"/>:null}
  {pending?<Tag value={t("Yayın bekliyor")} tone="warning"/>:null}
 </div>;
}
function Tile({label,value,desc}:{label:string;value:ReactNode;desc:string}){
 return <div className="rounded-xl border border-slate-200 bg-white p-4">
  <p className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500">{label}</p>
  <div className="text-sm font-semibold text-slate-800">{value}</div>
  <p className="mt-2 text-xs leading-5 text-slate-500">{desc}</p>
 </div>;
}
function accessTitle(policy:ServiceSecurityPolicy,tr:(s:string)=>string){
 const access=policy.authMode==="api_key"?"API Key":tr("Public");
 const count=policy.ipAllowlist.length+policy.ipDenylist.length;
 return access+(count>0?" · "+String(count)+" CIDR":"");
}
export function ServiceSecurityOverview({service,report,loading,error}:{
 service:Service;report:ServiceSecurityStatus|null;loading:boolean;error:string;
}){
 const {t}=useLanguage();
 const p=report?.saved;
 const waf=p?.wafMode??service.wafMode??"off";
 const auth=p?.authMode??service.authMode??"public";
 const rate=p?.rateLimitRps??service.rateLimitRps??0;
 const allow=p?.ipAllowlist?.length??service.ipAllowlist?.length??0;
 const deny=p?.ipDenylist?.length??service.ipDenylist?.length??0;
 const changed=report?.pending??(service.status==="active"&&service.version!==service.publishedVersion);
 const custom=service.routes.some(r=>(r.rateLimitRps??0)>0);
 const routeQuota=(policy:ServiceSecurityPolicy)=>{
   const values=new Set(policy.routes.map(r=>r.rateLimitRps));
   return values.size>1 ? t("Endpoint bazlı") : quotaTitle(policy.rateLimitRps,t);
 };
 return <section id="security-overview" className="space-y-4 scroll-mt-8" aria-labelledby="security-overview-title">
  <div className="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h2 id="security-overview-title" className="text-xl font-semibold">{t("Güvenlik ve Trafik")}</h2>
      <p className="mt-1 text-xs text-slate-500">{t("Kaydedilen politikalar ve düğümlerde uygulanan gerçek değerler.")}</p>
    </div>
    <a className="btn-primary" href="#security-settings">{t("Politikaları düzenle")}</a>
  </div>
  {changed?<div role="status" className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
    <strong>{t("Yayın bekleyen değişiklikler var.")}</strong> {t("Kaydedilen politika henüz canlı sürümle aynı değil. Düğüm tablosunu kontrol edin.")}
  </div>:null}
  <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
    <Tile label="OWASP CRS / WAF" value={<Tag value={wafTitle(waf,t)} tone={waf==="block"?"safe":waf==="detect"?"warning":"neutral"}/>}
      desc={t("Servis genelinde WAF politikası.")}/>
    <Tile label={t("Rate limit")} value={rate===0&&custom?t("Endpoint bazlı"):quotaTitle(rate,t)}
      desc={custom?t("Endpoint bazlı istisnalar var."):t("Düğüm başına; global kota değil.")}/>
    <Tile label={t("İstemci erişimi")} value={<Tag value={auth==="api_key"?"API Key":t("Public")} tone={auth==="api_key"?"safe":"neutral"}/>}
      desc={t("Allowlist")+": "+allow+" · "+t("Denylist")+": "+deny}/>
    <Tile label={t("Backend kimlik doğrulaması")} value={p?.upstreamAuthMode??service.upstreamAuthMode??"none"}
      desc={(p?.requiredClientHeader??service.requiredClientHeader)?t("Zorunlu header")+": "+(p?.requiredClientHeader??service.requiredClientHeader):t("Ek istemci header koşulu yok.")}/>
  </div>
  {report?.saved.routes.length ? <details className="rounded-xl border border-slate-200 bg-white">
    <summary className="cursor-pointer p-4 text-sm font-semibold">{t("Endpoint politikalarını karşılaştır")} ({report.saved.routes.length})</summary>
    <div className="overflow-x-auto"><table className="w-full min-w-[580px] text-left text-xs">
      <thead className="bg-slate-50 text-slate-500"><tr>
        <th className="px-4 py-3">{t("Endpoint")}</th>
        <th className="px-4 py-3">{t("Kayıtlı kota")}</th>
        {report.nodes.map(n=><th key={n.name} className="px-4 py-3">{n.name}</th>)}
      </tr></thead>
      <tbody>{report.saved.routes.map((route,index)=><tr className="border-t border-slate-100" key={route.path+route.matchType+index}>
        <td className="px-4 py-3 font-mono">{route.methods.join(",")||"ANY"} {route.path}</td>
        <td className="px-4 py-3">{quotaTitle(route.rateLimitRps,t)}</td>
        {report.nodes.map(n=>{
          const applied=n.applied?.routes.find(x=>x.path===route.path&&x.matchType===route.matchType&&x.methods.join(",")===route.methods.join(","));
          return <td className="px-4 py-3" key={n.name}>{applied?quotaTitle(applied.rateLimitRps,t):"—"}</td>;
        })}
      </tr>)}</tbody>
    </table></div>
    <p className="border-t border-slate-100 p-3 text-xs text-slate-500">{t("0 olan endpoint ayarı servis kotasını miras alır. Burada etkin kotalar gösterilir.")}</p>
  </details> : null}
  <div className="overflow-hidden rounded-xl border border-slate-200 bg-white">
    <div className="border-b border-slate-100 p-4">
      <h3 className="text-sm font-semibold">{t("Düğüm bazında uygulanan politika")}</h3>
      <p className="mt-1 text-xs text-slate-500">{t("Envoy ACK ile doğrulanan revizyondan okunur; bağlantısı olmayan düğüm aktif sayılmaz.")}</p>
    </div>
    {loading?<p className="p-4 text-xs text-slate-500">{t("Politika durumu yükleniyor…")}</p>:
    error?<p role="alert" className="p-4 text-xs text-red-700">{error}</p>:
    !report?<p className="p-4 text-xs text-slate-500">{t("Politika doğrulanamadı.")}</p>:
    <div className="overflow-x-auto"><table className="w-full min-w-[740px] text-left text-xs">
      <thead className="bg-slate-50 text-slate-500"><tr>
        {[t("Düğüm"),t("Durum"),"WAF",t("Rate limit"),t("Erişim"),t("Revizyon")].map(x=><th key={x} className="border-b border-slate-100 px-4 py-3 font-semibold">{x}</th>)}
      </tr></thead>
      <tbody>{report.nodes.map(n=><tr key={n.name} className="border-b border-slate-100">
        <td className="px-4 py-3"><strong>{n.name}</strong><p className="mt-1 max-w-[160px] break-words text-[10px] text-slate-400">{n.location}</p></td>
        <td className="px-4 py-3"><Tag value={n.state==="acknowledged"?t("ACK onaylı"):n.state==="offline"?t("Çevrimdışı"):n.state==="outdated"?t("Eski sürüm"):t("Kurulmadı")}
          tone={n.state==="acknowledged"?"safe":n.state==="not_deployed"?"neutral":"warning"}/></td>
        <td className="px-4 py-3">{n.applied?wafTitle(n.applied.wafMode,t):"—"}</td>
        <td className="px-4 py-3">{n.applied?routeQuota(n.applied):"—"}</td>
        <td className="px-4 py-3">{n.applied?accessTitle(n.applied,t):"—"}</td>
        <td className="px-4 py-3 font-mono">{n.ackedRevision!=null?"r"+n.ackedRevision:"—"}
          {n.desiredRevision!==n.ackedRevision?<span className="block text-[10px] text-amber-700">{t("İstenen")}: {"r"+(n.desiredRevision??"—")}</span>:null}</td>
      </tr>)}</tbody>
    </table></div>}
    <div className="border-t border-slate-100 p-4 text-xs text-slate-500">{t("HTTP ve WAF olay sayaçları şu anda düğüm toplamıdır, bu servise özel değildir.")}
      {" "}<Link href="/nodes" className="font-semibold text-teal-700 underline">{t("Düğüm metriklerini aç")}</Link>
    </div>
  </div>
 </section>;
}
