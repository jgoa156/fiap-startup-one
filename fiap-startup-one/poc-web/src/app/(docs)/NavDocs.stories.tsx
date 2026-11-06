import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { NavDocs } from "./NavDocs";

const meta: Meta<typeof NavDocs> = {
  title: "Componentes/Barra da documentação",
  component: NavDocs,
  parameters: { layout: "fullscreen", docs: { description: { component: "A barra das páginas de documentação e operação; marca a página atual." } } },
};
export default meta;

export const NaApi: StoryObj<typeof NavDocs> = { name: "Na página da API", parameters: { nextjs: { navigation: { pathname: "/docs/api" } } } };
export const NaOperacao: StoryObj<typeof NavDocs> = { name: "Na operação", parameters: { nextjs: { navigation: { pathname: "/operacao" } } } };
