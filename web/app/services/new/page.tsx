"use client";

import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { PublishWizard } from "@/components/publish-wizard";

function Wizard() {
  const params = useSearchParams();
  const id = params.get("service") ?? undefined;
  return <PublishWizard key={id ?? "new"} serviceId={id} />;
}

export default function NewServicePage() {
  return (
    <Suspense fallback={<p>Yükleniyor…</p>}>
      <Wizard />
    </Suspense>
  );
}
