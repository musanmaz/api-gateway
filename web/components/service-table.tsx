import Link from "next/link";
import { Badge, EmptyState } from "@/components/ui";
import { Icon } from "@/components/icon";
import type { Service } from "@/lib/types";
import {useLanguage} from "@/lib/i18n";
import {ServiceSecurityTags} from "@/components/service-security";

export function ServiceTable({ items, compact = false }: { items: Service[] | null; compact?: boolean }) {
  const {t}=useLanguage();
  if (items === null) {
    return <div className="space-y-5 p-6" aria-label="Servisler yükleniyor">
      {[1,2,3].map(i => <div key={i} className="skeleton h-9"/>)}
    </div>;
  }
  if (!items.length) {
    return <EmptyState>Henüz gösterilecek servis yok.
      <Link className="mt-3 block text-teal-700" href="/services/new">İlk servisinizi ekleyin →</Link>
    </EmptyState>;
  }
  return <div className="table-scroll">
    <table className="data-table">
      <thead><tr>
        <th>{t("Servis / Gateway erişim linki")}</th>
        <th>{t("Hedef")}</th>
        <th>{t("Yayın durumu")}</th>
        {!compact && <th>{t("Güvenlik politikası")}</th>}
        {!compact && <th>{t("Sorumlu")}</th>}
        <th><span className="sr-only">Ayrıntılar</span></th>
      </tr></thead>
      <tbody>{items.map(service => {
        const endpoint = service.routes[0]?.upstream.endpoints[0];
        const url = service.gatewayBaseUrl && service.routes[0] ? service.gatewayBaseUrl + service.routes[0].path : null;
        return <tr key={service.id}>
          <td><div className="flex items-start gap-3">
            <span className="service-symbol"><Icon name="globe" size={17}/></span>
            <div className="min-w-0">
              <Link className="service-link" href={`/services/${service.id}`}>{service.name}</Link>
              <p className="mt-1 text-[10px] text-slate-400">{service.hostnames[0] ?? "Özel alan adı yok"} · {service.routes.length} rota · {service.authMode === "api_key" ? "API Key korumalı" : "Public"}</p>
              {url ? service.status === "active" ?
                <a href={url} target="_blank" rel="noreferrer" title={url}
                  className="mt-2 inline-flex max-w-full items-center gap-1 break-all text-[11px] font-medium text-teal-700 hover:underline">
                  <Icon name="external" size={12}/>{url}
                </a> :
                <p className="mt-2 break-all text-[11px] text-slate-500">Yayın sonrası: {url}</p>
                : <p className="mt-2 text-[10px] text-slate-400">HTTP gateway linki yok</p>}
            </div>
          </div></td>
          <td><span className="mono whitespace-nowrap text-[11px] text-slate-500">{endpoint ? `${endpoint.address}:${endpoint.port}` : "Tanımlanmadı"}</span>
            <p className="mt-1 text-[10px] text-slate-400">{service.routes[0]?.path ?? "—"}</p>
          </td>
          <td><Badge value={service.status}/></td>
          {!compact && <td><ServiceSecurityTags service={service}/></td>}
          {!compact && <td className="text-slate-500">{service.owner}</td>}
          <td><Link aria-label={`${service.name} ayrıntıları`} href={`/services/${service.id}`}
            className="inline-flex rounded p-2 text-slate-400 hover:bg-slate-100 hover:text-teal-700">
            <Icon name="chevron" size={15}/></Link></td>
        </tr>;
      })}</tbody>
    </table>
  </div>;
}
