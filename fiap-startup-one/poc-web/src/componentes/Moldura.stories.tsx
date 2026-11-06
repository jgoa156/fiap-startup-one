import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { Card, Typography } from "antd";
import { Moldura } from "./Moldura";

export const euDeExemplo = {
  id: 1, cpf: "99499123938",
  dados: { nome: "Carla Devolução Teste", email: "carla@exemplo.test", data_nascimento: "1992-03-04", municipio: "Campinas", uf: "SP" },
};

const meta: Meta<typeof Moldura> = {
  title: "Componentes/Moldura",
  component: Moldura,
  parameters: {
    layout: "fullscreen",
    nextjs: { navigation: { pathname: "/" } },
    api: { "GET /eu": euDeExemplo },
    docs: { description: { component: "A moldura da área logada: lateral em cartão que recolhe para ícones, busca com Ctrl+K e ajuda em toda tela. No celular, a lateral vira gaveta. Sob carga, a moldura inteira vira a sala de espera." } },
  },
  args: {
    children: <Card><Typography.Title level={3} style={{ marginTop: 0 }}>Conteúdo da tela</Typography.Title>Cada tela do portal aparece aqui dentro.</Card>,
  },
};
export default meta;
type Historia = StoryObj<typeof Moldura>;

export const Normal: Historia = {};

export const NoCelular: Historia = {
  name: "No celular",
  globals: { viewport: { value: "mobile1", isRotated: false } },
};

export const NaFila: Historia = {
  name: "No pico (a API devolve a fila)",
  parameters: { api: { "GET /eu": { status: 503, corpo: { erro: "Muita gente acessando agora.", degradado: 1, fila: { posicao: 412, espera_segundos: 95 } } } } },
};
