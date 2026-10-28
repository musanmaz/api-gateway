import type { Metadata } from "next";
import { Shell } from "@/components/shell";
import { LanguageProvider } from "@/lib/i18n";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: "Gateway Control · Operasyon Merkezi", template: "%s · Gateway Control" },
  robots: { index: false, follow: false },
  description: "Servis geçidi operasyon paneli",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="tr">
      <body>
        <LanguageProvider><Shell>{children}</Shell></LanguageProvider>
      </body>
    </html>
  );
}
