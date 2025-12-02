import "./globals.css";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Dataset Explorer",
  description: "Search geospatial datasets via RAG backend",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
