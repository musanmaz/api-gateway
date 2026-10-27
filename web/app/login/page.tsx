"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError("");
    sessionStorage.setItem("gateway.token", token.trim());
    try {
      await api("/api/v1/overview");
      router.push("/");
    } catch {
      sessionStorage.removeItem("gateway.token");
      setError("Bu anahtar kabul edilmedi. Anahtarı eksiksiz kopyaladığınızdan emin olun.");
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={submit} className="mx-auto mt-24 max-w-md rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <h1 className="text-2xl font-semibold">Gateway paneline giriş</h1>
      <ol className="mt-3 list-decimal space-y-1 pl-5 text-sm text-slate-600">
        <li>
          Proje klasöründe terminale <code className="rounded bg-slate-100 px-1">make token</code> yazın.
        </li>
        <li>Çıkan uzun satırı kopyalayın.</li>
        <li>Aşağıdaki kutuya yapıştırın.</li>
      </ol>
      <label className="mt-6 block text-sm font-medium" htmlFor="token">
        Giriş anahtarı
      </label>
      <input
        id="token"
        type="password"
        autoComplete="off"
        required
        value={token}
        onChange={(event) => setToken(event.target.value)}
        className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2"
      />
      {error ? (
        <p role="alert" className="mt-3 text-sm text-red-800">
          {error}
        </p>
      ) : null}
      <button type="submit" disabled={pending} className="mt-4 rounded-lg bg-blue-600 px-4 py-2 font-medium text-white disabled:opacity-60">
        {pending ? "Kontrol ediliyor…" : "Giriş"}
      </button>
    </form>
  );
}
