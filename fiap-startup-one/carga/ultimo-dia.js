// Ensaio de carga do H1 — o último dia de inscrição (pico 2: escrita pesada).
//
// Cada iteração é UM candidato fazendo o caminho inteiro, como uma pessoa faria: cria a conta,
// preenche a inscrição em passos com tempo de pensar, gera o Pix (às vezes com clique duplo),
// paga no banco e abre o cartão. Do perfil do último dia (04-escopo-poc §8):
//   - chegada concentrada: sobe de 10% a 100% da taxa em 90 s (o edital marca a hora);
//   - rajadas: a cada 30 min, 2 min com o dobro (a notícia "prazo acaba hoje" circula);
//   - recarregamento: resposta lenta (> 2 s) ou erro → metade recarrega e pede de novo;
//   - abandono e retorno: 12% param no meio; 60% desses voltam minutos depois e continuam o rascunho;
//   - sala de espera: 503 com fila → espera o Retry-After e tenta de novo, mantendo a senha.
//
// Variáveis:
//   BASE_URL   onde o candidato entra (nuvem: o CloudFront; local: a API em http://localhost:8082)
//   FRONT_URL  a página do front, se for outro endereço (local: http://localhost:8080)
//   CERTAME    id do certame (o cmd/seed imprime)
//   MULT       múltiplo da carga média (1, 10, 20, 40, ...)
//   DURACAO    tempo no patamar (padrão 3h)
//   PENSAR     fator do tempo de pensar (1 = gente de verdade; 0.05 = validação rápida)
//   RAJADAS    "0" desliga as rajadas
//   RODADA     marca os candidatos desta rodada (ensaio.<RODADA>.<n>@lisura.test) para a auditoria
//   INSTANCIA  0, 1, 2... quando vários k6 disparam juntos (separa os CPFs)

import http from "k6/http";
import { check, sleep } from "k6";
import exec from "k6/execution";
import { Counter, Rate, Trend } from "k6/metrics";

const BASE = (__ENV.BASE_URL || "http://localhost:8080").replace(/\/$/, "");
// Onde fica a página do front. Na nuvem é o mesmo endereço; na máquina local o k6 fala direto com o
// Go (8082) e só a página vem do Next (8080), para não medir o proxy de desenvolvimento do Next.
const FRONT = (__ENV.FRONT_URL || BASE).replace(/\/$/, "");
const CERTAME = Number(__ENV.CERTAME || 0);
const MULT = Number(__ENV.MULT || 1);
const DURACAO = __ENV.DURACAO || "3h";
const PENSAR = Number(__ENV.PENSAR ?? 1);
const RAJADAS = (__ENV.RAJADAS ?? "1") !== "0";
const RODADA = __ENV.RODADA || String(Date.now() % 1e6);
const INSTANCIA = Number(__ENV.INSTANCIA || 0);
const BLOCO = Math.max(1, Number(RODADA) % 100); // faixa de CPFs da rodada (1 a 99)
if (!/^\d+$/.test(RODADA)) throw new Error("RODADA precisa ser numérica (os 2 últimos dígitos escolhem a faixa de CPFs)");
if (INSTANCIA > 4) throw new Error("no máximo 5 instâncias do k6 por rodada (faixa de CPFs)");

// ---- a carga média e o alvo ----
// Ano 5 do plano: 7,7 milhões de inscrições por ano. Média = 7.700.000 / 8.760 h ≈ 879 por hora.
// 40× = 35.160 inscrições por hora (9,8 por segundo); 3 h a 40× ≈ 105 mil — o último dia de um
// certame grande inteiro.
export const MEDIA_POR_HORA = 7_700_000 / 8760;
const porMinuto = (MEDIA_POR_HORA * MULT) / 60;
const alvo = Math.max(1, Math.round(porMinuto));

function minutos(d) {
  const m = /^(\d+(?:\.\d+)?)(h|m|s)$/.exec(d);
  if (!m) throw new Error(`DURACAO inválida: ${d}`);
  return Number(m[1]) * { h: 60, m: 1, s: 1 / 60 }[m[2]];
}

// Candidatos simultâneos = chegadas por minuto × duração da jornada em minutos. A jornada tem ~3 min
// de tempo de pensar (mais ~0,4 min médio de quem abandona e volta), escalados por PENSAR, e ~0,15
// min fixos de espera (Pix, confirmação, cartão). Antes a conta ignorava o PENSAR: a 640× com
// PENSAR=0,05 pré-alocava 28 mil VUs para ~3 mil simultâneos, e o disparo de todos juntos estourava
// a fila de conexões do Windows (medido em 05/11).
const jornadaMin = 3.4 * PENSAR + 0.15;
const simultaneos = alvo * jornadaMin;
const vusMax = Math.ceil(simultaneos * 2 * (RAJADAS ? 2 : 1)) + 50;
const vusIniciais = Math.ceil(simultaneos * 1.2) + 20;

// Rajada a cada 30 min: sobe em 30 s, fica 90 s, cai em 10 s. O k6 faz rampa linear entre estágios,
// então a queda precisa de um estágio curto próprio (senão a rajada "escorreria" pelos 28 min seguintes).
function estagiosDeRajada() {
  const total = (minutos(DURACAO) + 1.5) * 60; // segundos do cenário principal
  const estagios = [];
  let agora = 0;
  for (let inicio = 28 * 60; inicio + 130 <= total; inicio += 30 * 60) {
    estagios.push({ duration: `${inicio - agora}s`, target: 0 });
    estagios.push({ duration: "30s", target: alvo });
    estagios.push({ duration: "90s", target: alvo });
    estagios.push({ duration: "10s", target: 0 });
    agora = inicio + 130;
  }
  return estagios;
}

const cenarios = {
  inscricao: {
    executor: "ramping-arrival-rate",
    exec: "candidato",
    timeUnit: "1m",
    startRate: Math.max(1, Math.round(alvo * 0.1)),
    preAllocatedVUs: Math.min(vusMax, vusIniciais),
    maxVUs: vusMax,
    stages: [
      { duration: "90s", target: alvo },
      { duration: DURACAO, target: alvo },
      { duration: "30s", target: 0 },
    ],
    gracefulStop: `${Math.ceil(2 + 10 * PENSAR)}m`, // quem está no meio do caminho termina
  },
};
const rajadas = estagiosDeRajada();
if (RAJADAS && rajadas.length > 0) {
  cenarios.rajada = {
    executor: "ramping-arrival-rate",
    exec: "candidato",
    timeUnit: "1m",
    startRate: 0,
    preAllocatedVUs: 10,
    maxVUs: vusMax,
    stages: rajadas,
    gracefulStop: cenarios.inscricao.gracefulStop,
  };
}

export const options = {
  scenarios: cenarios,
  thresholds: {
    // As métricas do H1 (04-escopo-poc §6). Um limite estourado deixa o k6 com código de saída ≠ 0.
    "http_req_duration{critico:sim}": ["p(95)<2000"],
    jornada_concluida: ["rate>0.98"],
    falhas: ["count==0"],
  },
  summaryTrendStats: ["avg", "p(50)", "p(95)", "p(99)", "max"],
  // 409 (conta já existe), 503 (fila / modo degradado) e 404 do cartão ainda não publicado são
  // respostas esperadas, não falhas de rede.
  discardResponseBodies: false,
};

http.setResponseCallback(http.expectedStatuses({ min: 200, max: 299 }, 409, 503));

// ---- métricas ----
const jornadaConcluida = new Rate("jornada_concluida");
const jornada = new Trend("jornada_s");
const tempoAtePix = new Trend("tempo_ate_pix_s");
const tempoAteConfirmar = new Trend("tempo_ate_confirmar_s");
const tempoAteCartao = new Trend("tempo_ate_cartao_s");
const esperaNaFila = new Trend("espera_na_fila_s");
const inscricoesEnviadas = new Counter("inscricoes_enviadas");
const pagamentosVistos = new Counter("pagamentos_vistos");
const cartoesAbertos = new Counter("cartoes_abertos");
const abandonos = new Counter("abandonos");
const retornos = new Counter("retornos");
const recarregamentos = new Counter("recarregamentos");
const cliquesDuplos = new Counter("cliques_duplos");
const naoPagaram = new Counter("nao_pagaram");
const falhas = new Counter("falhas");

// ---- utilidades ----
const sorte = (p) => Math.random() < p;
const entre = (a, b) => a + Math.random() * (b - a);
const pensar = (a, b) => sleep(entre(a, b) * PENSAR);
const escolher = (xs) => xs[Math.floor(Math.random() * xs.length)];

function cpf(n) {
  const base = String(n % 1e9).padStart(9, "0").split("").map(Number);
  for (const tam of [9, 10]) {
    let s = 0;
    for (let i = 0; i < tam; i++) s += base[i] * (tam + 1 - i);
    const r = (s * 10) % 11;
    base.push(r === 10 ? 0 : r);
  }
  return base.join("");
}

class Falha extends Error {}

// Envio de arquivo (multipart): o k6 monta o corpo a partir de um objeto com http.file.
class Formulario {
  constructor(campos) {
    this.campos = campos;
  }
}

// Um PDF mínimo: o servidor confere o formato pelo conteúdo ("%PDF-"), não pelo nome.
const LAUDO = "%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n";

// Se a última chamada a pedir() precisou recarregar. Cada VU roda o seu candidato numa linha só, então
// a variável é dele.
let recarregouUltima = false;

// pedir faz a requisição como o navegador do candidato: espera na sala de espera, recarrega quando
// demora ou falha, e diz se deu certo.
function pedir(metodo, caminho, corpo, nome, { aceitar = [200], critico = true, recarrega = true } = {}) {
  const url = caminho.startsWith("http") ? caminho : `${BASE}${caminho}`;
  const multipart = corpo instanceof Formulario;
  const params = {
    headers: corpo !== undefined && !multipart ? { "Content-Type": "application/json" } : {},
    // `name` agrupa as URLs (/api/inscricoes/123 e /456 viram uma série só): sem ele, cada id vira
    // uma série de métricas e a memória do k6 cresce sem parar num ensaio de 3 h.
    tags: { name: nome, nome, critico: critico ? "sim" : "nao" },
    timeout: "30s",
  };
  const dados = multipart ? corpo.campos : corpo !== undefined ? JSON.stringify(corpo) : null;
  let tentativas = 0;
  const filaDesde = Date.now();
  let naFila = false;
  for (;;) {
    const res = http.request(metodo, url, dados, params);
    if (res.status === 503) {
      let r = null;
      try {
        r = res.json();
      } catch (_) {}
      if (r && r.fila) {
        // Na sala de espera: a página espera sozinha e tenta de novo (a senha vai no cookie).
        if (Date.now() - filaDesde > 45 * 60 * 1000) throw new Falha(`${nome}: 45 min na fila`);
        naFila = true;
        sleep(Math.min(Number(res.headers["Retry-After"] || 5), 15));
        continue;
      }
    }
    if (naFila) {
      esperaNaFila.add((Date.now() - filaDesde) / 1000);
      naFila = false;
    }
    const ruim = res.status === 0 || res.status >= 500 || res.timings.duration > 2000;
    if (ruim && recarrega && tentativas < 2 && sorte(0.5)) {
      // "Travou": o candidato recarrega. Os passos são idempotentes no servidor, inclusive o pagar.
      tentativas++;
      recarregamentos.add(1);
      sleep(entre(1, 4));
      continue;
    }
    if (!aceitar.includes(res.status)) {
      throw new Falha(`${nome}: ${res.status} ${String(res.body).slice(0, 120)}`);
    }
    // Quem chama decide o que um 409 depois de recarregar quer dizer. Fica numa variável do VU, e não
    // na resposta: a resposta do k6 é objeto do Go e recusa propriedade nova (medido em 05/11: a
    // primeira versão derrubou todas as jornadas).
    recarregouUltima = tentativas > 0;
    return res;
  }
}

// ---- preparação: lê o certame uma vez (do arquivo estático, como o front) ----
export function setup() {
  if (!CERTAME) throw new Error("defina CERTAME (o cmd/seed imprime o id)");
  const res = http.get(`${BASE}/publico/certames/${CERTAME}.json`);
  if (res.status !== 200) throw new Error(`certame ${CERTAME} não publicado: ${res.status}`);
  const c = res.json();
  console.log(
    `rodada ${RODADA} · MULT ${MULT}× · alvo ${alvo} inscrições/min (${(alvo / 60).toFixed(2)}/s) · ` +
      `patamar ${DURACAO} · até ${vusMax} VUs · certame ${CERTAME}`,
  );
  return { cargos: c.cargos.map((g) => g.id), cidades: c.cidades_prova.map((d) => d.id) };
}

// ---- o candidato ----
export function candidato(certame) {
  const inicio = Date.now();
  // CPFs sem sobreposição: cada rodada tem uma faixa de 10 milhões (os 2 últimos dígitos da RODADA),
  // dividida entre instâncias do k6 (2 milhões cada) e entre o cenário principal e as rajadas.
  // (Antes era rodada×1000+n e a rodada seguinte reaproveitava as contas da anterior: medido em
  // 05/11, 725 jornadas falharam por cair em inscrições alheias já finalizadas.)
  const n = INSTANCIA * 2e6 + (exec.scenario.name === "rajada" ? 1e6 : 0) + exec.scenario.iterationInTest + 1;
  const meuCpf = cpf(BLOCO * 1e7 + n);
  const email = `ensaio.${RODADA}.${n}@lisura.test`;
  const vaiAbandonar = sorte(0.12);
  const passoDoAbandono = Math.floor(entre(1, 6));
  const volta = vaiAbandonar && sorte(0.6);
  let passo = 0;

  // abandonar() devolve true se o candidato desistiu de vez.
  const abandonar = () => {
    if (!vaiAbandonar || passo++ !== passoDoAbandono) return false;
    abandonos.add(1);
    if (!volta) return true;
    pensar(120, 480); // volta minutos depois, com a sessão ainda aberta
    retornos.add(1);
    pedir("GET", "/api/inscricoes", undefined, "minhas_inscricoes");
    return false;
  };

  try {
    pedir("GET", `${FRONT}/entrar`, undefined, "pagina_entrar", { critico: false });
    pensar(5, 15);
    const criada = pedir("POST", "/api/contas", { cpf: meuCpf, nome: `Candidata Ensaio ${n}`, email }, "criar_conta", {
      aceitar: [201, 409],
    });
    // 409 = o CPF já tem conta. Depois de um recarregamento, é a primeira tentativa que deu certo sem a
    // resposta chegar: o candidato entra na conta que acabou de criar, como faria pela tela de entrar.
    // (Antes isso contava como falha: medido em 05/11, a 1.280× local, 338 jornadas "falharam" assim.)
    // Sem recarregamento, só acontece repetindo a faixa de CPF de outra RODADA: é erro do ensaio, e
    // entrar na conta de outro candidato sujaria a auditoria.
    if (criada.status === 409) {
      if (!recarregouUltima) throw new Falha(`criar_conta: CPF ${meuCpf} já existe (RODADA repetida?)`);
      pedir("POST", "/api/sessoes", { cpf: meuCpf }, "entrar_depois_de_recarregar");
    }
    pedir("GET", "/api/eu", undefined, "eu");
    pedir("GET", `/publico/certames/${CERTAME}.json`, undefined, "certame_estatico", { critico: false });

    pensar(10, 30);
    pedir("PUT", "/api/conta", {
      nome: `Candidata Ensaio ${n}`, email, data_nascimento: "1996-05-20", municipio: "Campinas", uf: "SP",
    }, "dados_pessoais", { aceitar: [200, 204] });
    if (abandonar()) return;

    pensar(5, 15);
    const aberta = pedir("POST", `/api/certames/${CERTAME}/inscricoes`, { cargo_id: escolher(certame.cargos) }, "abrir_inscricao", {
      aceitar: [200, 201],
    });
    const id = aberta.json("id");
    if (abandonar()) return;

    pensar(3, 10);
    pedir("PUT", `/api/inscricoes/${id}/cidade`, { cidade_id: escolher(certame.cidades) }, "passo_cidade", { aceitar: [200, 204] });
    if (abandonar()) return;
    pensar(3, 10);
    pedir("PUT", `/api/inscricoes/${id}/cotas`, { cotas: sorte(0.2) ? ["ppiq"] : [] }, "passo_cotas", { aceitar: [200, 204] });
    pensar(2, 6);
    // 5% pedem atendimento especial, que exige laudo: enviam o arquivo antes (é o upload sob carga).
    let atendimento = { necessidades: [] };
    if (sorte(0.05)) {
      const doc = pedir("POST", "/api/conta/documentos", new Formulario({
        tipo: "laudo", arquivo: http.file(LAUDO, "laudo.pdf", "application/pdf"),
      }), "enviar_laudo", { aceitar: [200, 201] });
      atendimento = { necessidades: ["tempo_adicional"], laudo_documento_id: doc.json("id") };
    }
    pedir("PUT", `/api/inscricoes/${id}/atendimento`, atendimento, "passo_atendimento", { aceitar: [200, 204] });
    if (abandonar()) return;
    pensar(5, 20);
    pedir("PUT", `/api/inscricoes/${id}/declaracoes`, { aceito: true }, "passo_declaracoes", { aceitar: [200, 204] });
    pedir("GET", `/api/inscricoes/${id}`, undefined, "revisao");
    if (abandonar()) return;

    pensar(3, 10);
    pedir("POST", `/api/inscricoes/${id}/finalizar`, undefined, "finalizar");
    inscricoesEnviadas.add(1);

    // Pagar: 85% Pix, 15% boleto; 5% clicam duas vezes no botão.
    pensar(2, 8);
    const metodo = sorte(0.85) ? "pix" : "boleto";
    if (sorte(0.05)) {
      cliquesDuplos.add(1);
      const req = {
        method: "POST", url: `${BASE}/api/inscricoes/${id}/pagamento`, body: JSON.stringify({ metodo }),
        params: { headers: { "Content-Type": "application/json" }, tags: { nome: "pagar_clique_duplo", critico: "sim" } },
      };
      const [a, b] = http.batch([req, req]);
      if (![a.status, b.status].every((s) => s === 200 || s === 201 || s === 202)) {
        throw new Falha(`clique duplo: ${a.status}/${b.status}`);
      }
    } else {
      pedir("POST", `/api/inscricoes/${id}/pagamento`, { metodo }, "pagar", { aceitar: [200, 201, 202] });
    }

    // O Pix chega pela fila: a tela pergunta a cada 1,5 s.
    const pedidoEm = Date.now();
    let cob = null;
    while (!cob) {
      const lista = pedir("GET", `/api/inscricoes/${id}/pagamento`, undefined, "ver_cobranca").json();
      cob = lista.find((x) => x.metodo === metodo && x.estado === "PENDENTE" && (x.pix_copia_cola || x.linha_digitavel));
      if (!cob) {
        if (Date.now() - pedidoEm > 120_000) throw new Falha("Pix/boleto não apareceu em 2 min");
        sleep(1.5);
      }
    }
    tempoAtePix.add((Date.now() - pedidoEm) / 1000);

    if (sorte(0.1)) {
      naoPagaram.add(1); // gerou e não pagou: não é falha do sistema
      return;
    }
    pensar(15, 60); // abre o app do banco e paga
    // Pagar no banco não recarrega: é o app do banco, não a página do Lisura, e quem recarrega a página
    // do Lisura não paga de novo. Antes recarregava como as outras: medido em 05/11, a 1.280× local, 128
    // boletos pagos 2 ou 3 vezes (309 pagamentos), reembolsos que nenhum candidato faria.
    pedir("POST", `/api/inscricoes/${id}/pagamento/${cob.id}/simular`, undefined, "pagar_no_banco", {
      aceitar: [200, 202, 204], recarrega: false,
    });

    const pagoEm = Date.now();
    for (;;) {
      const i = pedir("GET", `/api/inscricoes/${id}`, undefined, "esperar_confirmacao").json();
      if (i.estado === "PAGA") break;
      if (Date.now() - pagoEm > 300_000) throw new Falha(`pagamento não confirmado em 5 min (estado ${i.estado})`);
      sleep(3);
    }
    tempoAteConfirmar.add((Date.now() - pagoEm) / 1000);
    pagamentosVistos.add(1);

    for (;;) {
      const c = pedir("GET", `/api/inscricoes/${id}/cartao`, undefined, "cartao").json();
      if (c.disponivel) {
        tempoAteCartao.add((Date.now() - pagoEm) / 1000);
        const pagina = pedir("GET", c.url, undefined, "cartao_estatico", { critico: false });
        check(pagina, { "cartão com CPF mascarado": (r) => String(r.body).includes("***.") });
        cartoesAbertos.add(1);
        break;
      }
      if (Date.now() - pagoEm > 300_000) throw new Falha("cartão não publicado em 5 min");
      sleep(2);
    }

    jornada.add((Date.now() - inicio) / 1000);
    jornadaConcluida.add(true);
  } catch (e) {
    if (!(e instanceof Falha)) throw e;
    falhas.add(1, { motivo: e.message.split(":")[0] });
    jornadaConcluida.add(false);
    console.warn(`candidato ${n}: ${e.message}`);
  }
}

// ---- resumo: o que o relatório e a auditoria precisam ----
export function handleSummary(d) {
  const m = d.metrics;
  const v = (k, campo = "count") => (m[k] ? m[k].values[campo] : 0);
  const resumo = {
    rodada: RODADA,
    mult: MULT,
    alvo_por_minuto: alvo,
    duracao: DURACAO,
    inscricoes_enviadas: v("inscricoes_enviadas"),
    pagamentos_vistos: v("pagamentos_vistos"),
    cartoes_abertos: v("cartoes_abertos"),
    jornada_concluida: v("jornada_concluida", "rate"),
    falhas: v("falhas"),
    abandonos: v("abandonos"),
    retornos: v("retornos"),
    recarregamentos: v("recarregamentos"),
    cliques_duplos: v("cliques_duplos"),
    nao_pagaram: v("nao_pagaram"),
    p95_critico_ms: m["http_req_duration{critico:sim}"] ? m["http_req_duration{critico:sim}"].values["p(95)"] : null,
    p95_ms: v("http_req_duration", "p(95)"),
    requisicoes: v("http_reqs"),
    tempo_ate_pix_p95_s: v("tempo_ate_pix_s", "p(95)"),
    tempo_ate_confirmar_p95_s: v("tempo_ate_confirmar_s", "p(95)"),
    tempo_ate_cartao_p95_s: v("tempo_ate_cartao_s", "p(95)"),
    espera_na_fila_p95_s: v("espera_na_fila_s", "p(95)"),
    limites: Object.fromEntries(
      Object.entries(m)
        .filter(([, x]) => x.thresholds)
        .map(([k, x]) => [k, Object.values(x.thresholds).every((t) => t.ok)]),
    ),
  };
  const auditar = `lisura-auditoria -rodada ${RODADA} -enviadas ${resumo.inscricoes_enviadas} -pagas ${resumo.pagamentos_vistos}`;
  const saida = { stdout: `\nRESUMO_DO_ENSAIO ${JSON.stringify(resumo)}\nPARA_AUDITAR ${auditar}\n` };
  if (__ENV.SAIDA) saida[__ENV.SAIDA] = JSON.stringify({ resumo, metricas: d.metrics }, null, 2);
  return saida;
}
