"use client";

import { App, ConfigProvider } from "antd";
import ptBR from "antd/locale/pt_BR";
import { tema } from "@/tema/tema";

// O tema e o idioma do Ant Design valem para todas as telas.
export function Provedores({ children }: { children: React.ReactNode }) {
  return (
    <ConfigProvider theme={tema} locale={ptBR}>
      <App>{children}</App>
    </ConfigProvider>
  );
}
