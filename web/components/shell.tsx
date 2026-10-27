"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { ReactNode } from "react";
import { ZONES, useTimeZone } from "@/lib/format";

const links: { href: string; label: string; also: string[] }[] = [
  { href: "/", label: "Ana sayfa", also: [] },
  { href: "/services", label: "Servisler", also: [] },
  { href: "/changes", label: "Geçmiş", also: ["/audit"] },
  { href: "/nodes", label: "Sistem", also: [] },
  { href: "/certificates", label: "Sertifikalar", also: [] },
];

export function Shell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { timeZone, setTimeZone } = useTimeZone();
  if (pathname === "/login") return <main className="min-h-screen">{children}</main>;

  return (
    <div className="min-h-screen bg-slate-100 text-slate-900">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-x-6 gap-y-3 px-5 py-3">
          <Link href="/" className="text-base font-semibold tracking-tight">
            Geçit
          </Link>
          <nav className="flex flex-wrap gap-1" aria-label="Ana menü">
            {links.map(({ href, label, also }) => {
              const active = href === "/" ? pathname === "/" : [href, ...also].some((item) => pathname.startsWith(item));
              return (
                <Link key={href} href={href} className={`rounded-full px-3 py-1.5 text-sm ${active ? "bg-slate-900 text-white" : "text-slate-600 hover:bg-slate-100"}`}>
                  {label}
                </Link>
              );
            })}
          </nav>
          <div className="ml-auto flex items-center gap-3">
            <label className="text-sm text-slate-500" htmlFor="timezone">
              Saat
              <select id="timezone" className="ml-2 rounded-lg border border-slate-200 bg-white px-2 py-1 text-slate-800" value={timeZone} onChange={(event) => setTimeZone(event.target.value)}>
                {ZONES.map((zone) => (
                  <option key={zone} value={zone}>
                    {zone}
                  </option>
                ))}
              </select>
            </label>
            <button
              type="button"
              className="rounded-full border border-slate-200 px-3 py-1 text-sm"
              onClick={() => {
                sessionStorage.removeItem("gateway.token");
                router.push("/login");
              }}
            >
              Çıkış
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-5 py-8">{children}</main>
    </div>
  );
}
