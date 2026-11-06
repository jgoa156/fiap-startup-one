import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { cor, tema } from "@/tema/tema";

// Os fundamentos lidos do tema e dos tokens de verdade: mudou o tema, mudou aqui.

const meta: Meta = { title: "Fundamentos", parameters: { layout: "padded" } };
export default meta;

function contraste(hex: string, fundo = "#FFFFFF") {
  const lum = (h: string) => {
    const [r, g, b] = [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16) / 255).map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
    return 0.2126 * r + 0.7152 * g + 0.0722 * b;
  };
  const [a, b] = [lum(hex), lum(fundo)].sort((x, y) => y - x);
  return (a + 0.05) / (b + 0.05);
}

const usos: Record<keyof typeof cor, string> = {
  acao: "Ação primária: botões, links, foco",
  titulo: "Títulos",
  secundaria: "Ícones, ação secundária",
  selecao: "Item selecionado",
  superficie: "Superfície de apoio, hover",
  marcador: "Marca e destaque (decorativo)",
  avisoFundo: "Fundo de aviso e pendência",
  avisoTexto: "Texto sobre o fundo de aviso",
  tinta: "Texto",
  tinta2: "Texto secundário",
  divisor: "Divisores e bordas",
  fundo: "Fundo da moldura",
  sucesso: "Sucesso",
  atencao: "Atenção",
  erro: "Erro",
};

export const Cores: StoryObj = {
  render: () => (
    <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(220px, 1fr))", gap: 12 }}>
      {(Object.keys(cor) as (keyof typeof cor)[]).map((k) => {
        const c = contraste(cor[k]);
        return (
          <div key={k} style={{ background: "#fff", borderRadius: 12, boxShadow: "var(--shadow-sm)", overflow: "hidden" }}>
            <div style={{ height: 64, background: cor[k], borderBottom: "1px solid var(--line-200)" }} />
            <div style={{ padding: 10, fontSize: 13 }}>
              <strong>{k}</strong> <code>{cor[k]}</code>
              <div style={{ color: "var(--ink-600)" }}>{usos[k]}</div>
              <div style={{ color: c >= 4.5 ? "var(--state-success)" : "var(--ink-600)" }}>
                {c.toFixed(2)}:1 sobre branco {c >= 7 ? "· AAA" : c >= 4.5 ? "· AA para texto" : "· não serve para texto"}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  ),
};

export const Tipografia: StoryObj = {
  render: () => (
    <div style={{ display: "grid", gap: 10 }}>
      <div style={{ color: "var(--ink-600)", fontSize: 13 }}>Nunito em toda a interface ({String(tema.token?.fontFamily)}); corpo de {tema.token?.fontSize}px.</div>
      {[
        ["Título da tela", 30, 800], ["Título de seção", 20, 800], ["Título de cartão", 16, 700],
        ["Corpo do texto: o candidato lê isto.", 14, 400], ["Texto secundário e legendas", 12, 400],
      ].map(([t, s, w]) => (
        <div key={String(t)} style={{ fontSize: Number(s), fontWeight: Number(w), color: Number(w) >= 700 ? "var(--brand-800)" : "var(--ink-900)" }}>
          {t} <span style={{ fontSize: 12, color: "var(--ink-600)", fontWeight: 400 }}>· {s}px / {w}</span>
        </div>
      ))}
    </div>
  ),
};

export const RaiosSombrasEToque: StoryObj = {
  name: "Raios, sombras e alvo de toque",
  render: () => (
    <div style={{ display: "flex", gap: 24, flexWrap: "wrap", alignItems: "flex-end" }}>
      {[["sm", 8], ["md", 12], ["lg", 14], ["pill", 999]].map(([n, r]) => (
        <div key={String(n)} style={{ textAlign: "center", fontSize: 12 }}>
          <div style={{ width: 96, height: 56, background: "var(--brand-100)", border: "2px solid var(--brand-700)", borderRadius: Number(r) }} />
          --radius-{n}
        </div>
      ))}
      {["sm", "md", "lg"].map((s) => (
        <div key={s} style={{ textAlign: "center", fontSize: 12 }}>
          <div style={{ width: 110, height: 70, background: "#fff", borderRadius: 14, boxShadow: `var(--shadow-${s})` }} />
          --shadow-{s}
        </div>
      ))}
      <div style={{ textAlign: "center", fontSize: 12 }}>
        <div style={{ width: tema.token?.controlHeight, height: tema.token?.controlHeight, border: "2px dashed var(--marker-400)", borderRadius: 999 }} />
        toque: {tema.token?.controlHeight}px
      </div>
    </div>
  ),
};
