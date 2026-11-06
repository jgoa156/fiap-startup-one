"use client";

import { ApiReferenceReact } from "@scalar/api-reference-react";
import "@scalar/api-reference-react/style.css";

// O Scalar lê a especificação do servidor no ar e permite chamar cada rota ("Try it") pelo próprio
// endereço do front: o cookie de sessão vai junto, como no portal.
export function Referencia() {
  return (
    <div style={{ background: "var(--surface)", borderRadius: "var(--radius-lg)", boxShadow: "var(--shadow-sm)", overflow: "hidden" }}>
      <ApiReferenceReact
        configuration={{
          url: "/api/openapi.yaml",
          hideClientButton: false,
          hideDarkModeToggle: true,
          forceDarkModeState: "light",
          defaultHttpClient: { targetKey: "js", clientKey: "fetch" },
          // Nada sai daqui para a Scalar: sem telemetria, sem o agente de IA nem o MCP deles (que
          // mandariam a especificação para fora).
          telemetry: false,
          agent: { disabled: true },
          mcp: { disabled: true },
          customCss: `.scalar-app { --scalar-font: var(--font-sans); --scalar-color-accent: #2F3B4C; --scalar-radius: 12px; }`,
        }}
      />
    </div>
  );
}
