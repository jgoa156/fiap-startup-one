import type { Decorator, Preview } from "@storybook/nextjs-vite";
import { Provedores } from "../src/app/provedores";
import "../src/app/globals.css";

// Cada história pode simular a API: parameters.api = { "GET /eu": {...}, "/inscricoes/1/avisos": [...] }.
// O valor é o corpo JSON (status 200), ou { status, corpo } para outro status. O que não está na
// lista responde 404. Assim os componentes que buscam dados aparecem em todos os estados, sem servidor.
type Simulada = unknown | { status: number; corpo?: unknown };

const original = typeof window !== "undefined" ? window.fetch.bind(window) : undefined;

const comApi: Decorator = (Story, ctx) => {
  const rotas = (ctx.parameters.api ?? {}) as Record<string, Simulada>;
  window.fetch = async (entrada, init) => {
    const url = new URL(typeof entrada === "string" ? entrada : entrada instanceof URL ? entrada.href : entrada.url, location.href);
    if (!url.pathname.startsWith("/api/")) return original!(entrada, init);
    const caminho = url.pathname.slice(4);
    const metodo = (init?.method ?? "GET").toUpperCase();
    const r = rotas[`${metodo} ${caminho}`] ?? rotas[caminho];
    if (r === undefined) return new Response(JSON.stringify({ erro: `sem simulação para ${metodo} ${caminho}` }), { status: 404 });
    const { status, corpo } = r !== null && typeof r === "object" && "status" in (r as object) ? (r as { status: number; corpo?: unknown }) : { status: 200, corpo: r };
    return new Response(status === 204 || status === 202 ? null : JSON.stringify(corpo ?? null), { status, headers: { "Content-Type": "application/json" } });
  };
  return <Story />;
};

const preview: Preview = {
  decorators: [
    comApi,
    (Story) => (
      <Provedores>
        <div style={{ fontFamily: "var(--font-sans)" }}>
          <Story />
        </div>
      </Provedores>
    ),
  ],
  parameters: {
    layout: "padded",
    nextjs: { appDirectory: true },
    backgrounds: { options: { fundo: { name: "Fundo da moldura", value: "#F4F5F8" }, branco: { name: "Cartão", value: "#FFFFFF" } } },
    options: { storySort: { order: ["Introdução", "Fundamentos", "Ant Design com o tema", "Componentes", "Telas"] } },
  },
  initialGlobals: { backgrounds: { value: "fundo" } },
  tags: ["autodocs"],
};

export default preview;
