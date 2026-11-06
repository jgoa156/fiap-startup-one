import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { ModoConsulta, SalaDeEspera } from "./Espera";

const meta: Meta = {
  title: "Componentes/Degradação",
  parameters: {
    layout: "fullscreen",
    docs: { description: { component: "As telas que tomam o lugar do erro sob carga: a sala de espera (nível 1) e o modo só consulta (nível 3)." } },
  },
};
export default meta;

export const SalaDeEsperaNoPico: StoryObj<typeof SalaDeEspera> = {
  name: "Sala de espera",
  args: { fila: { posicao: 1834, espera_segundos: 310 } },
  argTypes: { tentar: { action: "tentou de novo" } },
  render: (a) => <SalaDeEspera {...a} tentar={a.tentar ?? (() => {})} />,
};

export const QuaseAVez: StoryObj<typeof SalaDeEspera> = {
  name: "Sala de espera, quase a vez",
  args: { fila: { posicao: 3, espera_segundos: 12 } },
  render: (a) => <SalaDeEspera {...a} tentar={() => {}} />,
};

export const SoConsultaComCartoes: StoryObj = {
  name: "Só consulta, com cartões guardados",
  loaders: [async () => {
    localStorage.setItem("lisura.cartoes", JSON.stringify([
      { inscricao: 1123248, certame: "PSS 2027 — Prefeitura de Exemplo", url: "#" },
      { inscricao: 998, certame: "Concurso 2026 — Câmara de Exemplo", url: "#" },
    ]));
    return {};
  }],
  render: () => <ModoConsulta aoVoltar={() => {}} />,
};

export const SoConsultaSemCartoes: StoryObj = {
  name: "Só consulta, sem cartões neste aparelho",
  loaders: [async () => { localStorage.removeItem("lisura.cartoes"); return {}; }],
  render: () => <ModoConsulta aoVoltar={() => {}} />,
};
