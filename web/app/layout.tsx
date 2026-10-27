import type { Metadata } from "next";
import { Shell } from "@/components/shell";
import "./globals.css";

export const metadata: Metadata = {
  title: "Gateway Control",
  description: "Servis geçidi operasyon paneli",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="tr">
      <body>
        <Shell>{children}</Shell>
      </body>
    </html>
  );
}
