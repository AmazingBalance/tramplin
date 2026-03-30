import type { Metadata } from "next";
import { IBM_Plex_Serif, Manrope } from "next/font/google";
import "leaflet/dist/leaflet.css";

import "@/app/globals.css";

const sans = Manrope({
  subsets: ["latin", "cyrillic"],
  variable: "--font-sans",
});

const serif = IBM_Plex_Serif({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "500", "600"],
  variable: "--font-serif",
});

export const metadata: Metadata = {
  title: "Tramplin",
  description:
    "Платформа для стажировок, junior-вакансий, менторских программ и карьерных событий.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ru">
      <body className={`${sans.variable} ${serif.variable}`}>{children}</body>
    </html>
  );
}
