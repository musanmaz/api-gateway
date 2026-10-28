"use client";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { ReactNode, useEffect, useState } from "react";
import { ZONES, useTimeZone } from "@/lib/format";
import { BrandMark, Icon, IconName } from "@/components/icon";
import { useLanguage } from "@/lib/i18n";

const links: { href: string; label: string; icon: IconName }[] = [
  { href: "/", label: "Genel bakış", icon: "grid" },
  { href: "/services", label: "Servisler", icon: "layers" },
  { href: "/nodes", label: "Gateway düğümleri", icon: "nodes" },
  { href: "/test-lab", label: "Test laboratuvarı", icon: "lab" },
  { href: "/changes", label: "Yayın geçmişi", icon: "history" },
  { href: "/certificates", label: "Sertifikalar", icon: "shield" },
  { href: "/audit", label: "İşlem kayıtları", icon: "terminal" },
];
export function Shell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { timeZone, setTimeZone } = useTimeZone();
  const {language,setLanguage,t}=useLanguage();
  const [open, setOpen] = useState(false);
  useEffect(() => { setOpen(false); }, [pathname]);
  const active = (href: string) => href === "/" ? pathname === "/" : pathname.startsWith(href);
  const current = links.find(item => active(item.href));
  if (pathname === "/login") return <main><div className="fixed right-4 top-4 z-50"><label className="sr-only" htmlFor="login-language">Language / Dil</label><select id="login-language" value={language} onChange={e=>setLanguage(e.target.value as "tr"|"en")} className="rounded-lg border bg-white px-3 py-2 text-xs shadow-sm"><option value="tr">Türkçe</option><option value="en">English</option></select></div>{children}</main>;
  return (
    <div className="min-h-screen">
      <a className="sr-only focus:not-sr-only focus:fixed focus:z-50 focus:bg-white focus:p-4" href="#main-content">İçeriğe geç</a>
      {open && <button className="fixed inset-0 z-30 bg-slate-950/40" aria-label="Menüyü kapat" onClick={() => setOpen(false)} />}
      <aside className={`app-sidebar ${open ? "open" : ""}`} aria-label="Kenar menü">
        <Link href="/" className="brand"><BrandMark /><div><div className="brand-name">gateway<span className="text-teal-400">.</span></div><div className="brand-sub">control plane</div></div></Link>
        <div className="nav-label eyebrow">{t("Çalışma alanı")}</div>
        <nav aria-label="Ana menü">{links.map(({ href, label, icon }, i) => <div key={href}>{i === 4 && <div className="nav-label eyebrow">{t("Yönetim")}</div>}<Link href={href} aria-current={active(href) ? "page" : undefined} className={`nav-link ${active(href) ? "active" : ""}`}><Icon name={icon} size={17} />{t(label)}</Link></div>)}</nav>
        <div className="sidebar-bottom">
          <div className="sidebar-env"><div className="flex items-center justify-between"><span className="font-medium text-slate-200">{t("Test ortamı")}</span><span className="rounded border border-slate-600 px-1.5 py-0.5 text-[9px] tracking-wider">LAB</span></div><p className="mt-2 leading-relaxed text-slate-400">İki gateway düğümü.<br />Tek sunucuda doğrulama.</p></div>
          <div className="sidebar-user"><span className="avatar"><Icon name="user" size={16} /></span><div><p className="text-slate-200">{t("Yönetim oturumu")}</p><p className="mt-0.5 text-[10px] text-slate-500">Gateway Control</p></div><button aria-label="Oturumu kapat" className="ml-auto rounded p-2 hover:bg-slate-700" onClick={() => { sessionStorage.removeItem("gateway.token"); router.push("/login"); }}><Icon name="logout" size={17} /></button></div>
        </div>
      </aside>
      <div className="app-body">
        <header className="app-topbar"><div className="flex min-w-0 items-center gap-3 text-xs"><button className="mobile-toggle rounded p-1" aria-label="Menüyü aç" aria-expanded={open} onClick={() => setOpen(true)}><Icon name="menu" /></button><span className="hidden text-slate-400 sm:inline">Çalışma alanı</span><span className="hidden text-slate-300 sm:inline">/</span><span className="font-medium text-slate-600">{t(current?.label ?? "Gateway Control")}</span></div><div className="flex items-center gap-4"><label className="inline-flex items-center gap-1 text-xs text-slate-500"><span className="sr-only">{t("Dil")}</span><select aria-label="Language / Dil" className="rounded-md border border-slate-200 bg-white px-2 py-1.5 text-xs" value={language} onChange={e=>setLanguage(e.target.value as "tr"|"en")}><option value="tr">TR</option><option value="en">EN</option></select></label><span className="rounded border border-amber-200 bg-amber-50 px-2 py-1 text-[9px] font-semibold tracking-widest text-amber-700">TEST</span><label className="hidden items-center gap-2 text-xs text-slate-400 md:flex"><Icon name="clock" size={14} /><select aria-label="Saat dilimi" className="!min-h-0 border-0 text-[11px] text-slate-500" value={timeZone} onChange={event => setTimeZone(event.target.value)}>{ZONES.map(zone => <option key={zone}>{zone}</option>)}</select></label></div></header>
        <main id="main-content" className="app-main">{children}<footer className="mt-10 flex flex-wrap items-center justify-between gap-2 border-t border-slate-200/70 pt-5 text-[10px] text-slate-400"><span>Gateway Control <span className="mx-2">/</span> Servis operasyonları</span><span>{t("Kontrol sizde. Trafik Envoy’da.")}</span></footer></main>
      </div>
    </div>
  );
}
