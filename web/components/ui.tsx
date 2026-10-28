import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Icon, IconName } from "@/components/icon";
import { useLanguage } from "@/lib/i18n";
export function PageTitle({ title, text }: { title: string; text?: string }) {
  return <header><h1 className="page-title">{title}</h1>{text && <p className="page-subtitle max-w-3xl">{text}</p>}</header>;
}
export function Notice({ children }: { children: ReactNode }) { return <div className="gc-notice"><Icon name="info" className="mt-0.5 shrink-0" size={16} /><div>{children}</div></div>; }
export function Success({ children }: { children: ReactNode }) { return <div role="status" className="gc-notice gc-success"><Icon name="check" className="mt-0.5 shrink-0" size={16} /><div>{children}</div></div>; }
export function ErrorText({ children }: { children: ReactNode }) { return <div role="alert" className="gc-notice gc-error"><Icon name="info" className="mt-0.5 shrink-0" size={16} /><div>{children}</div></div>; }
export function Panel({ children, className = "" }: { children: ReactNode; className?: string }) { return <div className={`gc-panel p-6 ${className}`}>{children}</div>; }
export function Badge({ value }: { value: string }) {
  const {t}=useLanguage();
  const tone = ({fresh:"ok",stale:"bad",unknown:"off",draft:"wait",active:"ok"} as Record<string,string>)[value] ?? "off";
  const label = ({fresh:"Bağlı",stale:"Güncel değil",unknown:"Bilinmiyor",draft:"Taslak",active:"Yayımlandı"} as Record<string,string>)[value] ?? value;
  return <span className={`status-pill status-${tone}`}>{t(label)}</span>;
}
export function Dot({ tone }: { tone: "ok" | "wait" | "bad" | "off" }) { const color = {ok:"bg-teal-500",wait:"bg-amber-500",bad:"bg-red-500",off:"bg-slate-400"}[tone]; return <span aria-hidden className={`mr-2 inline-block h-1.5 w-1.5 rounded-full ${color}`} />; }
export function PrimaryButton({ children, className = "", ...props }: ButtonHTMLAttributes<HTMLButtonElement>) { return <button {...props} className={`btn-primary ${className}`}>{children}</button>; }
export function SecondaryButton({ children, className = "", ...props }: ButtonHTMLAttributes<HTMLButtonElement>) { return <button {...props} className={`btn-secondary ${className}`}>{children}</button>; }
export function MetricCard({ label, value, detail, icon }: { label: string; value: ReactNode; detail: string; icon: IconName }) {
  return <div className="gc-panel metric-card"><div className="flex items-center justify-between gap-2"><span className="text-xs text-slate-500">{label}</span><span className="metric-icon"><Icon name={icon} size={17} /></span></div><div className="metric-value">{value ?? <span className="skeleton inline-block h-8 w-12" />}</div><p className="mt-3 text-[10px] text-slate-400">{detail}</p></div>;
}
export function EmptyState({ children }: { children: ReactNode }) { return <div className="empty-state"><Icon name="layers" className="mx-auto mb-3 text-slate-300" size={28} />{children}</div>; }
