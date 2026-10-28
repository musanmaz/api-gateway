"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { Icon } from "@/components/icon";
import { ErrorText, PrimaryButton } from "@/components/ui";
import { useLanguage } from "@/lib/i18n";

export default function LoginPage() {
  const router = useRouter();
  const {t}=useLanguage();
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError("");
    sessionStorage.setItem("gateway.token", token.trim());
    try {
      await api("/api/v1/overview");
      router.push("/");
    } catch {
      sessionStorage.removeItem("gateway.token");
      setError("Giriş doğrulanamadı. Erişim anahtarını kontrol edin.");
    } finally {
      setPending(false);
    }
  }

  return (
    <main className="login-layout">
      <div className="login-form-wrap">
        <form onSubmit={submit} className="login-form">
          <span className="mb-8 inline-flex h-12 w-12 items-center justify-center rounded-xl border border-teal-100 bg-teal-50 text-teal-700">
            <Icon name="lock" size={23} />
          </span>
          <p className="eyebrow text-teal-700">{t("Gateway Yönetim Paneli")}</p>
          <h1 className="mt-3 text-[30px] font-semibold tracking-tight text-slate-800">{t("Giriş yap")}</h1>
          <p className="mt-3 text-sm leading-6 text-slate-500">
            {t("Servislerinizi yönetmek için erişim anahtarınızı girin.")}
          </p>
          <label htmlFor="token" className="mb-2 mt-8 block text-xs font-semibold text-slate-600">
            {t("Erişim anahtarı")}
          </label>
          <input
            id="token"
            type="password"
            autoComplete="off"
            required
            value={token}
            onChange={(event) => setToken(event.target.value)}
            placeholder={t("Erişim anahtarınız")}
            className="login-input"
          />
          {error && <div className="mt-4"><ErrorText>{error}</ErrorText></div>}
          <PrimaryButton type="submit" disabled={pending || !token.trim()} className="mt-5 w-full !py-3">
            {pending ? t("Doğrulanıyor…") : t("Giriş yap")} <Icon name="arrow" size={16} />
          </PrimaryButton>
        </form>
      </div>
    </main>
  );
}
