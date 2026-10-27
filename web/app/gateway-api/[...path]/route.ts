import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

const hop = new Set(["host", "connection", "content-length", "transfer-encoding", "keep-alive"]);

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  const base = (process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080").replace(/\/$/, "");
  const incoming = new URL(request.url);
  const target = `${base}/${path.join("/")}${incoming.search}`;
  const headers = new Headers();
  request.headers.forEach((value, key) => {
    if (!hop.has(key.toLowerCase())) headers.set(key, value);
  });
  const method = request.method;
  const body = method === "GET" || method === "HEAD" ? undefined : await request.arrayBuffer();
  let upstream: Response;
  try {
    upstream = await fetch(target, { method, headers, body, redirect: "manual", cache: "no-store" });
  } catch {
    return NextResponse.json({ error: "Yönetim API erişilemiyor" }, { status: 502 });
  }
  const out = new Headers();
  upstream.headers.forEach((value, key) => {
    if (!hop.has(key.toLowerCase())) out.set(key, value);
  });
  return new NextResponse(upstream.body, { status: upstream.status, headers: out });
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
