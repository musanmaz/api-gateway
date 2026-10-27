"use client";

import { useEffect } from "react";
import { useParams, useRouter } from "next/navigation";

export default function EditServicePage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();

  useEffect(() => {
    router.replace(`/services/new?service=${encodeURIComponent(params.id)}`);
  }, [params.id, router]);

  return <p>Yönlendiriliyor…</p>;
}
