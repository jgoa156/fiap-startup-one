"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ApiOutlined, AppstoreOutlined, CloudServerOutlined, DashboardOutlined, ReadOutlined } from "@ant-design/icons";

// A barra das páginas de documentação. /design é um <a> comum: é o Storybook estático, fora do Next.
const itens = [
  { href: "/docs", rotulo: "Visão geral", icone: <ReadOutlined /> },
  { href: "/docs/api", rotulo: "API", icone: <ApiOutlined /> },
  { href: "/design", rotulo: "Componentes", icone: <AppstoreOutlined />, estatico: true },
  { href: "/docs/infra", rotulo: "Infraestrutura", icone: <CloudServerOutlined /> },
  { href: "/operacao", rotulo: "Operação e ensaios", icone: <DashboardOutlined /> },
];

export function NavDocs() {
  const atual = usePathname();
  return (
    <header style={{ background: "var(--surface)", borderBottom: "1px solid var(--line-200)", position: "sticky", top: 0, zIndex: 10 }}>
      <nav aria-label="Documentação" style={{ maxWidth: 1280, margin: "0 auto", padding: "10px 16px", display: "flex", gap: 6, alignItems: "center", flexWrap: "wrap" }}>
        <Link href="/docs" style={{ fontWeight: 800, fontSize: 16, color: "var(--brand-800)", marginRight: 12 }}>Lisura · documentação viva</Link>
        {itens.map((i) => {
          const ativo = i.href === "/docs" ? atual === "/docs" : atual?.startsWith(i.href);
          const estilo: React.CSSProperties = {
            display: "inline-flex", alignItems: "center", gap: 6, padding: "8px 14px", borderRadius: "var(--radius-pill)", fontWeight: 700,
            color: ativo ? "var(--brand-700)" : "var(--ink-600)", background: ativo ? "var(--brand-100)" : "transparent",
          };
          return i.estatico ? (
            <a key={i.href} href={i.href} style={estilo}>{i.icone}{i.rotulo}</a>
          ) : (
            <Link key={i.href} href={i.href} style={estilo} aria-current={ativo ? "page" : undefined}>{i.icone}{i.rotulo}</Link>
          );
        })}
      </nav>
    </header>
  );
}
