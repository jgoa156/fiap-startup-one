import type { Metadata } from "next";
import { NavDocs } from "./NavDocs";

export const metadata: Metadata = {
  title: { default: "Documentação", template: "%s · Documentação · Lisura" },
};

// As páginas de documentação viva e de operação: sem login (não mostram dado de candidato), com a
// mesma barra no topo. /design (os componentes) é o Storybook estático, servido de public/design.
export default function LayoutDocs({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ minHeight: "100vh", background: "var(--bg)" }}>
      <NavDocs />
      <main style={{ maxWidth: 1280, margin: "0 auto", padding: "24px 16px 64px" }}>{children}</main>
    </div>
  );
}
