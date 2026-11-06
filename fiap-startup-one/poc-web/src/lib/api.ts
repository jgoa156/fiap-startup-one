"use client";

import { useCallback, useEffect, useState } from "react";

// Chamadas à API Go (repassada pelo Next em /api). A sessão vai no cookie HttpOnly, sozinha.

// Sob carga, a API responde 503 com o nível de degradação (1 sala de espera, 2 só o essencial,
// 3 só consulta) e, na sala de espera, a posição na fila.
export type Fila = { posicao: number; espera_segundos: number };
export const EVENTO_DEGRADADO = "lisura:degradado";

export class ErroApi extends Error {
  constructor(
    public status: number,
    mensagem: string,
    public faltando?: string[],
    public degradado?: number,
    public fila?: Fila,
  ) {
    super(mensagem);
  }
}

export async function api<T = void>(
  caminho: string,
  opcoes: { metodo?: string; json?: unknown; corpo?: BodyInit } = {},
): Promise<T> {
  const res = await fetch(`/api${caminho}`, {
    method: opcoes.metodo ?? (opcoes.json !== undefined || opcoes.corpo ? "POST" : "GET"),
    headers: opcoes.json !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: opcoes.json !== undefined ? JSON.stringify(opcoes.json) : opcoes.corpo,
  });
  if (res.status === 204 || res.status === 202) return undefined as T;
  const dados = await res.json().catch(() => null);
  if (!res.ok) {
    const erro = new ErroApi(res.status, dados?.erro ?? `erro ${res.status}`, dados?.faltando, dados?.degradado, dados?.fila);
    // Fila ou modo só consulta valem para a tela inteira, não só para esta chamada: a moldura ouve.
    if (erro.fila || erro.degradado === 3) window.dispatchEvent(new CustomEvent(EVENTO_DEGRADADO, { detail: erro }));
    throw erro;
  }
  return dados as T;
}

export type Documento = { id: number; tipo: string; nome_arquivo: string; tamanho: number; enviado_em: string };

// enviarDocumento guarda um arquivo na carteira da conta. O formato é conferido pelo conteúdo no
// servidor (PDF, JPG ou PNG, até 5 MB).
export async function enviarDocumento(tipo: "laudo" | "comprovante_isencao", arquivo: File) {
  const fd = new FormData();
  fd.append("tipo", tipo);
  fd.append("arquivo", arquivo);
  return api<{ id: number }>("/conta/documentos", { corpo: fd });
}

// useApi busca ao montar e expõe recarregar(); erro 401 sobe para a moldura mandar ao login.
export function useApi<T>(caminho: string | null) {
  const [dados, setDados] = useState<T>();
  const [erro, setErro] = useState<ErroApi>();
  const [carregando, setCarregando] = useState(caminho !== null);

  // buscar só muda estado depois da resposta (nunca durante o efeito) e ignora respostas de uma
  // tela que já foi desmontada ou trocou de caminho.
  const buscar = useCallback(
    (vivo: () => boolean) => {
      if (caminho === null) return Promise.resolve();
      return api<T>(caminho).then(
        (d) => {
          if (!vivo()) return;
          setDados(d);
          setErro(undefined);
          setCarregando(false);
        },
        (e) => {
          if (!vivo()) return;
          setErro(e instanceof ErroApi ? e : new ErroApi(0, "Sem conexão com o servidor."));
          setCarregando(false);
        },
      );
    },
    [caminho],
  );

  useEffect(() => {
    let vivo = true;
    buscar(() => vivo);
    return () => {
      vivo = false;
    };
  }, [buscar]);

  const recarregar = useCallback(() => {
    setCarregando(true);
    return buscar(() => true);
  }, [buscar]);

  return { dados, erro, carregando, recarregar };
}

// useCertame lê os dados públicos do certame do arquivo estático (/publico — CDN na produção), que
// não toca no banco; se ainda não foi publicado, cai para a API.
export function useCertame(id: number | string | null) {
  const [dados, setDados] = useState<Certame>();
  const [erro, setErro] = useState<ErroApi>();
  useEffect(() => {
    if (id === null) return;
    let vivo = true;
    fetch(`/publico/certames/${id}.json`)
      .then((r) => (r.ok ? (r.json() as Promise<Certame>) : api<Certame>(`/certames/${id}`)))
      .then(
        (c) => vivo && setDados(c),
        (e) => vivo && setErro(e instanceof ErroApi ? e : new ErroApi(0, "Sem conexão com o servidor.")),
      );
    return () => {
      vivo = false;
    };
  }, [id]);
  return { dados, erro };
}

// Cartões já gerados ficam guardados no aparelho: no modo só consulta a API não responde, mas o
// cartão é estático e continua abrindo.
export type CartaoGuardado = { inscricao: number; certame: string; url: string };

export function cartoesGuardados(): CartaoGuardado[] {
  try {
    return JSON.parse(localStorage.getItem("lisura.cartoes") ?? "[]");
  } catch {
    return [];
  }
}

export function guardarCartao(c: CartaoGuardado) {
  try {
    const outros = cartoesGuardados().filter((x) => x.inscricao !== c.inscricao);
    localStorage.setItem("lisura.cartoes", JSON.stringify([c, ...outros].slice(0, 20)));
  } catch {}
}

// ---- tipos da API ----

export type Dados = {
  nome: string;
  email: string;
  data_nascimento: string;
  municipio: string;
  uf: string;
};
export type Eu = { id: number; cpf: string; dados: Dados };

export type CertameItem = {
  id: number;
  nome: string;
  banca: string;
  inscricao_abre_em: string;
  inscricao_fecha_em: string;
};

export type Fase = { fase: string; abre_em: string; fecha_em: string };
export type Certame = {
  id: number;
  nome: string;
  banca: string;
  edital_versao: number | null;
  cargos: { id: number; nome: string; vagas: number; taxa_centavos: number }[];
  cidades_prova: { id: number; nome: string; uf: string }[];
  cotas: { codigo: string; nome: string; exige_laudo: boolean }[];
  cronograma: Fase[];
};

export type Estado =
  | "RASCUNHO"
  | "ISENCAO_ATIVA"
  | "AGUARDANDO_PAGAMENTO"
  | "PAGA"
  | "EXPIRADA"
  | "CANCELADA"
  | "EM_ANALISE"
  | "DEFERIDA"
  | "INDEFERIDA"
  | "HOMOLOGADA";

export type Resumo = {
  id: number;
  certame_id: number;
  certame_nome: string;
  cargo_nome: string;
  estado: Estado;
  atualizada_em: string;
};

export type Inscricao = {
  id: number;
  certame_id: number;
  estado: Estado;
  cargo_id: number;
  cidade_prova_id: number | null;
  cotas: string[];
  necessidades: string[];
  laudo_documento_id: number | null;
  declaracoes_aceitas_em: string | null;
};

export type EventoTrilha = { evento: string; para: Estado; em: string };

export type Cobranca = {
  id: number;
  metodo: "pix" | "boleto";
  valor_centavos: number;
  estado: string;
  pix_copia_cola?: string | null;
  linha_digitavel?: string | null;
  vence_em: string;
  paga_em?: string | null;
  reembolso_pendente: boolean;
};

export type Devolucao = {
  id: number;
  valor_centavos: number;
  motivo: string;
  estado: "PENDENTE" | "AGUARDANDO_CHAVE" | "CONCLUIDA" | "RECUSADA";
  criada_em: string;
  concluida_em?: string;
  recusa?: string;
  chave_informada: boolean;
};

// Aviso por e-mail: entregue = o provedor aceitou; devolvido = voltou depois (endereço que não
// existe, caixa recusando) ou o candidato marcou como spam. resumo = SHA-256 do assunto e do texto.
export type Aviso = {
  evento: string;
  assunto: string;
  enviada_em: string;
  entregue: boolean;
  resumo: string;
  devolvido: boolean;
  falha?: string;
  para: string;
};

export type Isencao = {
  id: number;
  estado: "EM_ANALISE" | "DEFERIDA" | "INDEFERIDA";
  motivo_codigo: string | null;
  solicitada_em: string;
  decidida_em: string | null;
};

// ---- utilitários de exibição ----

export const reais = (centavos: number) =>
  (centavos / 100).toLocaleString("pt-BR", { style: "currency", currency: "BRL" });

export const dataHora = (iso: string) =>
  new Date(iso).toLocaleString("pt-BR", { dateStyle: "short", timeStyle: "short" });

export const data = (iso: string) => new Date(iso).toLocaleDateString("pt-BR");

export const formatarCPF = (cpf: string) =>
  cpf.length === 11 ? `${cpf.slice(0, 3)}.${cpf.slice(3, 6)}.${cpf.slice(6, 9)}-${cpf.slice(9)}` : cpf;

export function janela(c: Certame, fase: string) {
  return c.cronograma.find((f) => f.fase === fase);
}

export const jaAbriu = (iso: string) => Date.parse(iso) <= Date.now();

export function aberta(c: Certame, fase: string) {
  const f = janela(c, fase);
  if (!f) return false;
  const agora = Date.now();
  return agora >= Date.parse(f.abre_em) && agora < Date.parse(f.fecha_em);
}
