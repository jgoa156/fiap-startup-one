import type { Estado } from "./api";

// Textos que o candidato lê. Tom: "ok" verde, "aviso" com o marcador, "erro" vermelho.
export type Tom = "ok" | "aviso" | "erro" | "neutro";

export const estados: Record<Estado, { rotulo: string; tom: Tom }> = {
  RASCUNHO: { rotulo: "Em preenchimento", tom: "neutro" },
  ISENCAO_ATIVA: { rotulo: "Enviada com isenção", tom: "ok" },
  AGUARDANDO_PAGAMENTO: { rotulo: "Aguardando pagamento", tom: "aviso" },
  PAGA: { rotulo: "Paga", tom: "ok" },
  EXPIRADA: { rotulo: "Expirada — pagamento não confirmado", tom: "erro" },
  CANCELADA: { rotulo: "Cancelada", tom: "erro" },
  EM_ANALISE: { rotulo: "Em análise pela banca", tom: "neutro" },
  DEFERIDA: { rotulo: "Deferida", tom: "ok" },
  INDEFERIDA: { rotulo: "Indeferida", tom: "erro" },
  HOMOLOGADA: { rotulo: "Homologada", tom: "ok" },
};

export const eventos: Record<string, string> = {
  CRIAR: "Inscrição iniciada",
  FINALIZAR_COM_PAGAMENTO: "Inscrição enviada — aguardando pagamento",
  FINALIZAR_ISENTO: "Inscrição enviada com isenção de taxa",
  CONFIRMAR_PAGAMENTO: "Pagamento confirmado",
  EXPIRAR: "Prazo de pagamento vencido",
  CANCELAR: "Inscrição cancelada",
  INICIAR_ANALISE: "Enviada para análise da banca",
  DEFERIR: "Inscrição deferida",
  INDEFERIR: "Inscrição indeferida",
  PROVER_RECURSO: "Recurso aceito",
  HOMOLOGAR: "Inscrição homologada",
  CANCELAR_CERTAME: "Certame cancelado pela banca",
};

// Por que um valor volta ao candidato.
export const motivosDevolucao: Record<string, string> = {
  PAGO_EM_DUPLICIDADE: "Pagamento em duplicidade",
  PAGO_APOS_EXPIRACAO: "Pago depois do prazo",
  CERTAME_CANCELADO: "Certame cancelado",
};

// Por que o banco não aceitou a chave Pix informada.
export const recusasChave: Record<string, string> = {
  TITULAR_DIFERENTE: "Essa chave Pix está em nome de outra pessoa. Informe uma chave no seu CPF.",
  CHAVE_INEXISTENTE: "Essa chave Pix não existe. Confira e informe de novo.",
  CHAVE_OBRIGATORIA: "Falta informar uma chave Pix sua.",
};

export const fases: Record<string, string> = {
  isencao: "Pedido de isenção",
  inscricao: "Inscrição",
  pagamento: "Pagamento",
  homologacao: "Homologação",
  ensalamento: "Local de prova",
  aplicacao: "Prova",
  gabarito: "Gabarito",
  resultado: "Resultado",
};

export const pendencias: Record<string, string> = {
  dados_pessoais: "Dados pessoais e endereço",
  cidade_prova: "Cidade de prova",
  laudo: "Laudo médico",
  declaracoes: "Declarações",
};

export const necessidades: Record<string, string> = {
  ledor: "Ledor — alguém lê a prova para você",
  transcritor: "Transcritor — alguém marca as respostas",
  prova_ampliada: "Prova com letra ampliada",
  sala_acessivel: "Sala de fácil acesso",
  tempo_adicional: "Tempo adicional de prova",
  lactante: "Espaço para amamentação",
};

export const ufs = ["AC", "AL", "AM", "AP", "BA", "CE", "DF", "ES", "GO", "MA", "MG", "MS", "MT", "PA", "PB", "PE", "PI", "PR", "RJ", "RN", "RO", "RR", "RS", "SC", "SE", "SP", "TO"];

// A trilha da inscrição como o candidato a vê (a máquina de estados, em 5 marcos).
export const jornada = ["Rascunho", "Enviada", "Paga", "Em análise", "Homologada"];

export function passoDaJornada(e: Estado): number {
  switch (e) {
    case "RASCUNHO":
      return 0;
    case "AGUARDANDO_PAGAMENTO":
    case "EXPIRADA":
    case "CANCELADA":
      return 1;
    case "PAGA":
    case "ISENCAO_ATIVA":
      return 2;
    case "EM_ANALISE":
    case "DEFERIDA":
    case "INDEFERIDA":
      return 3;
    case "HOMOLOGADA":
      return 4;
  }
}
