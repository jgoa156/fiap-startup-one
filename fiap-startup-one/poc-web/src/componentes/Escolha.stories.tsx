import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { useState } from "react";
import { EnvironmentOutlined } from "@ant-design/icons";
import { Escolha } from "./Escolha";
import { necessidades } from "@/lib/rotulos";

const meta: Meta = {
  title: "Componentes/Escolha",
  parameters: {
    docs: {
      description: {
        component: "Cartões de escolha no lugar de radio, checkbox e select: todas as opções à vista. Selecionado = borda, fundo e ✓, nunca só a cor. Teclado e leitor de tela: botões com aria-pressed num grupo rotulado.",
      },
    },
  },
};
export default meta;

function Cargo() {
  const [v, setV] = useState<number | null>(1);
  return (
    <Escolha rotulo="Cargo" colunas={2} valor={v} aoMudar={setV} opcoes={[
      { valor: 1, titulo: "Agente Administrativo", detalhe: "120 vagas · taxa R$ 80,00" },
      { valor: 2, titulo: "Analista de Sistemas", detalhe: "15 vagas · taxa R$ 120,00" },
      { valor: 3, titulo: "Fiscal (inscrição encerrada)", detalhe: "—", desabilitada: true },
    ]} />
  );
}

function Cidade() {
  const [v, setV] = useState<string | null>(null);
  return (
    <Escolha rotulo="Cidade de prova" valor={v} aoMudar={setV}
      opcoes={["Campinas", "São Paulo", "Santos", "Sorocaba"].map((c) => ({ valor: c, titulo: c, detalhe: "SP", icone: <EnvironmentOutlined /> }))} />
  );
}

function Atendimento() {
  const [v, setV] = useState<string[]>(["ledor"]);
  return <Escolha rotulo="Atendimento especial" multipla compacto valor={v} aoMudar={setV}
    opcoes={Object.entries(necessidades).map(([valor, titulo]) => ({ valor, titulo }))} />;
}

export const UmaSo: StoryObj = { name: "Uma só (cargo)", render: () => <Cargo /> };
export const ComIcone: StoryObj = { name: "Uma só, com ícone (cidade)", render: () => <Cidade /> };
export const VariasCompacta: StoryObj = { name: "Várias, compacta (atendimento)", render: () => <Atendimento /> };
