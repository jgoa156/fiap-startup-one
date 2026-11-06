import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { Space } from "antd";
import { Selo } from "./Selo";
import { estados } from "@/lib/rotulos";

const meta: Meta<typeof Selo> = {
  title: "Componentes/Selo",
  component: Selo,
  parameters: { docs: { description: { component: "Situação em uma palavra, sempre com ícone e texto: a cor nunca é o único sinal." } } },
  argTypes: { tom: { control: "inline-radio", options: ["ok", "aviso", "erro", "neutro"] } },
  args: { tom: "ok", children: "Paga" },
};
export default meta;
type Historia = StoryObj<typeof Selo>;

export const Interativo: Historia = {};

export const TodasAsSituacoes: Historia = {
  name: "Todas as situações da inscrição",
  render: () => (
    <Space wrap>
      {Object.entries(estados).map(([k, e]) => <Selo key={k} tom={e.tom}>{e.rotulo}</Selo>)}
    </Space>
  ),
};
