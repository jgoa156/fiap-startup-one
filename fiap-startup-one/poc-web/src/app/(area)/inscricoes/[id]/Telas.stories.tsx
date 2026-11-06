import type { Decorator, Meta, StoryObj } from "@storybook/nextjs-vite";
import { Moldura } from "@/componentes/Moldura";
import { euDeExemplo } from "@/componentes/Moldura.stories";
import type { Aviso, Certame, Cobranca, Devolucao, EventoTrilha, Inscricao } from "@/lib/api";
import { Avisos } from "./Avisos";
import { Devolucoes } from "./Devolucoes";
import { Situacao } from "./Situacao";

// Os blocos da página da inscrição, em cada estado que o candidato pode ver. A API é simulada por
// história (parameters.api), então o estado raro aparece sem precisar provocá-lo.

const ID = 1123248;
const agora = "2026-11-06T17:00:00-03:00";

const certame: Certame = {
  id: 3, nome: "PSS 2027 — Prefeitura de Exemplo", banca: "Instituto Exemplo de Seleção", edital_versao: 1,
  cargos: [{ id: 1, nome: "Agente Administrativo", vagas: 120, taxa_centavos: 8000 }],
  cidades_prova: [{ id: 1, nome: "Campinas", uf: "SP" }],
  cotas: [{ codigo: "PPP", nome: "Pessoas pretas e pardas", exige_laudo: false }],
  cronograma: [{ fase: "pagamento", abre_em: "2026-11-01T00:00:00-03:00", fecha_em: "2026-12-01T01:54:00-03:00" }],
};
const inscricao = (estado: Inscricao["estado"]): Inscricao => ({
  id: ID, certame_id: 3, estado, cargo_id: 1, cidade_prova_id: 1, cotas: [], necessidades: [], laudo_documento_id: null, declaracoes_aceitas_em: agora,
});
const trilha: EventoTrilha[] = [
  { evento: "CRIAR", para: "RASCUNHO", em: "2026-11-06T16:59:00-03:00" },
  { evento: "FINALIZAR_COM_PAGAMENTO", para: "AGUARDANDO_PAGAMENTO", em: "2026-11-06T16:59:30-03:00" },
  { evento: "CONFIRMAR_PAGAMENTO", para: "PAGA", em: agora },
];
const pix: Cobranca = { id: 10, metodo: "pix", valor_centavos: 8000, estado: "PAGA", vence_em: "2026-11-07T16:59:00-03:00", paga_em: agora, reembolso_pendente: false };
const devolucao = (estado: Devolucao["estado"], extra: Partial<Devolucao> = {}): Devolucao => ({
  id: 25792, valor_centavos: 8000, motivo: "PAGO_EM_DUPLICIDADE", estado, criada_em: agora, chave_informada: false, ...extra,
});
const aviso = (extra: Partial<Aviso>): Aviso => ({
  evento: "CONFIRMAR_PAGAMENTO", assunto: `Pagamento confirmado: inscrição nº ${ID}`, enviada_em: agora, entregue: true,
  resumo: "6ecc1ecdefae5f1b8a0d3c9e2f7a4b6d1c8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3", devolvido: false, para: euDeExemplo.dados.email, ...extra,
});

const naMoldura: Decorator = (Story) => <Moldura><Story /></Moldura>;
const base = { "GET /eu": euDeExemplo };

const meta: Meta = {
  title: "Telas/Inscrição",
  decorators: [naMoldura],
  parameters: { layout: "fullscreen", nextjs: { navigation: { pathname: `/inscricoes/${ID}` } } },
};
export default meta;

export const DevolucaoEmAndamento: StoryObj = {
  name: "Devolução em andamento",
  render: () => <Devolucoes inscricao={ID} />,
  parameters: { api: { ...base, [`/inscricoes/${ID}/devolucoes`]: [devolucao("PENDENTE")] } },
};

export const FaltaChavePix: StoryObj = {
  name: "Devolução: falta a chave Pix",
  render: () => <Devolucoes inscricao={ID} />,
  parameters: { api: { ...base, [`/inscricoes/${ID}/devolucoes`]: [devolucao("AGUARDANDO_CHAVE", { recusa: "CHAVE_INEXISTENTE" })] } },
};

export const ChaveRecusada: StoryObj = {
  name: "Devolução: a chave informada foi recusada",
  render: () => <Devolucoes inscricao={ID} />,
  parameters: { api: { ...base, [`/inscricoes/${ID}/devolucoes`]: [devolucao("AGUARDANDO_CHAVE", { recusa: "TITULAR_DIFERENTE", chave_informada: true })] } },
};

export const Devolvida: StoryObj = {
  name: "Devolução concluída, e uma do certame cancelado",
  render: () => <Devolucoes inscricao={ID} />,
  parameters: { api: { ...base, [`/inscricoes/${ID}/devolucoes`]: [
    devolucao("CONCLUIDA", { concluida_em: agora, chave_informada: true }),
    devolucao("RECUSADA", { id: 25793, motivo: "CERTAME_CANCELADO" }),
  ] } },
};

export const AvisosEntregues: StoryObj = {
  name: "Avisos entregues",
  render: () => <Avisos inscricao={ID} />,
  parameters: { api: { ...base, [`/inscricoes/${ID}/avisos`]: [
    aviso({ evento: "FINALIZAR_COM_PAGAMENTO", assunto: `Inscrição nº ${ID} enviada: falta pagar até 01/12/2026 às 01:54` }),
    aviso({}),
  ] } },
};

export const AvisoQueVoltou: StoryObj = {
  name: "Aviso que voltou (pede e-mail novo)",
  render: () => <Avisos inscricao={ID} />,
  parameters: { api: { ...base, "PUT /conta": { status: 204 }, [`/inscricoes/${ID}/avisos`]: [
    aviso({ evento: "FINALIZAR_COM_PAGAMENTO", assunto: `Inscrição nº ${ID} enviada: falta pagar até 01/12/2026 às 01:54` }),
    aviso({ devolvido: true, falha: "Permanent/General: smtp; 550 5.1.1 caixa inexistente" }),
  ] } },
};

const paga = {
  ...base,
  [`/inscricoes/${ID}/trilha`]: trilha,
  [`/inscricoes/${ID}/pagamento`]: [pix],
  [`/inscricoes/${ID}/cartao`]: { disponivel: true, url: "#" },
  [`/inscricoes/${ID}/devolucoes`]: [],
  [`/inscricoes/${ID}/avisos`]: [aviso({})],
};

export const SituacaoPaga: StoryObj = {
  name: "Situação: paga",
  render: () => <Situacao inscricao={inscricao("PAGA")} certame={certame} aoMudar={() => {}} />,
  parameters: { api: paga },
};

export const SituacaoAguardandoPix: StoryObj = {
  name: "Situação: aguardando o pagamento",
  render: () => <Situacao inscricao={inscricao("AGUARDANDO_PAGAMENTO")} certame={certame} aoMudar={() => {}} />,
  parameters: { api: { ...paga, [`/inscricoes/${ID}/trilha`]: trilha.slice(0, 2), [`/inscricoes/${ID}/avisos`]: [],
    [`/inscricoes/${ID}/pagamento`]: [{ ...pix, estado: "PENDENTE", paga_em: null, pix_copia_cola: "00020126580014br.gov.bcb.pix0136lisura-sandbox-cob_exemplo5204000053039865406" }] } },
};

export const SituacaoCancelada: StoryObj = {
  name: "Situação: certame cancelado",
  render: () => <Situacao inscricao={inscricao("CANCELADA")} certame={certame} aoMudar={() => {}} />,
  parameters: { api: { ...paga, [`/inscricoes/${ID}/devolucoes`]: [devolucao("PENDENTE", { motivo: "CERTAME_CANCELADO" })] } },
};
