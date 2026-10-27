import type { ButtonHTMLAttributes, ReactNode } from "react";

export function PageTitle({ title, text }: { title: string; text?: string }) {
  return (
    <header className="mb-6">
      <h1 className="text-3xl font-semibold tracking-tight">{title}</h1>
      {text ? <p className="mt-2 max-w-3xl text-base text-slate-600">{text}</p> : null}
    </header>
  );
}

export function Notice({ children }: { children: ReactNode }) {
  return <p className="rounded-lg border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-950">{children}</p>;
}

export function Success({ children }: { children: ReactNode }) {
  return <p className="rounded-lg border border-teal-300 bg-teal-50 px-3 py-2 text-sm text-teal-950">{children}</p>;
}

export function ErrorText({ children }: { children: ReactNode }) {
  return (
    <p role="alert" className="rounded-lg border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-900">
      {children}
    </p>
  );
}

export function Panel({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`rounded-2xl border border-slate-200 bg-white p-5 shadow-sm ${className}`}>{children}</div>;
}

export function Badge({ value }: { value: string }) {
  const styles: Record<string, string> = {
    fresh: "bg-teal-100 text-teal-950",
    stale: "bg-red-100 text-red-950",
    unknown: "bg-stone-200 text-stone-800",
    draft: "bg-amber-100 text-amber-950",
    active: "bg-teal-100 text-teal-950",
  };
  const labels: Record<string, string> = {
    fresh: "çalışıyor",
    stale: "ulaşılamıyor",
    unknown: "bilinmiyor",
    draft: "taslak, yayında değil",
    active: "yayında",
  };
  return <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${styles[value] ?? styles.unknown}`}>{labels[value] ?? value}</span>;
}

export function Dot({ tone }: { tone: "ok" | "wait" | "bad" | "off" }) {
  const color = { ok: "bg-teal-600", wait: "bg-amber-500", bad: "bg-red-600", off: "bg-stone-400" }[tone];
  return <span aria-hidden className={`mr-2 inline-block h-2.5 w-2.5 rounded-full ${color}`} />;
}

export function PrimaryButton({ children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button {...props} className={`rounded-lg bg-blue-600 px-4 py-2 font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 ${props.className ?? ""}`}>
      {children}
    </button>
  );
}

export function SecondaryButton({ children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button {...props} className={`rounded-lg border border-slate-200 bg-white px-4 py-2 font-medium disabled:opacity-50 ${props.className ?? ""}`}>
      {children}
    </button>
  );
}
