import type { Metadata } from "next";
import { Nunito } from "next/font/google";
import { AntdRegistry } from "@ant-design/nextjs-registry";
import { Provedores } from "./provedores";
import "./globals.css";

const nunito = Nunito({
  variable: "--fonte-nunito",
  subsets: ["latin"],
  weight: ["400", "600", "700", "800"],
});

export const metadata: Metadata = {
  title: { default: "Lisura", template: "%s · Lisura" },
  description: "Inscrição em concursos e processos seletivos",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="pt-BR" className={nunito.variable}>
      <body>
        {/* Sem o registro, os estilos do Ant Design só aparecem depois que a página carrega. */}
        <AntdRegistry>
          <Provedores>{children}</Provedores>
        </AntdRegistry>
      </body>
    </html>
  );
}
