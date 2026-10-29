// Gerador de carga da POC: perfil do último dia de inscrição.
//
// Uso (PowerShell):
//   k6 run carga\ultimo-dia.js                                  # patamar padrão, 3 h
//   k6 run -e MODO=escada carga\ultimo-dia.js                   # sobe até quebrar
//   k6 run -e PATAMAR=8000 -e DURACAO_MIN=30 carga\ultimo-dia.js
//
// O que torna o ensaio honesto (HANDOFF-POC §3):
//   1. Chegada concentrada: rajadas de 90 s acima da média, a cada 10 min. A média
//      da hora continua sendo o PATAMAR; a rajada sai do resto da hora.
//   2. A unidade é a INSCRIÇÃO COMPLETA, com todas as chamadas dela, mais
//      recarregamento e abandono. O relatório dá as duas unidades.
//   3. Repetição e reenvio deliberados, para testar a idempotência sob carga.
//
// FATOR_RAJADA = 3 confirmado pelo dono do projeto em 29/10/2026: nas rajadas de
// 90 s chegam 3× a média, cerca de 10,6 inscrições por segundo a 16,7 mil/h.
// Usar outro fator é outro ensaio, e o resumo registra que não é o confirmado.

import http from "k6/http";
import { check, sleep } from "k6";
import { Counter, Rate, Trend } from "k6/metrics";
import exec from "k6/execution";

const BASE = __ENV.BASE_URL || "http://localhost:8000";
const MODO = __ENV.MODO || "patamar";
const PATAMAR = Number(__ENV.PATAMAR || 16700);          // inscrições por hora
const DURACAO_MIN = Number(__ENV.DURACAO_MIN || 180);
const FATOR_RAJADA = Number(__ENV.FATOR_RAJADA || 3);    // confirmado em 29/10/2026
const FATOR_CONFIRMADO = 3;
const RAJADA_S = Number(__ENV.RAJADA_S || 90);
const CICLO_S = Number(__ENV.CICLO_S || 600);
const CERTAME = Number(__ENV.CERTAME || 1);
const CONTAS_SEMEADAS = Number(__ENV.CONTAS_SEMEADAS || 0); // de app.semear
const SENHA = __ENV.SENHA_SEMEADA || "senha-sintetica-1";

// Comportamento do candidato. Estimativas nossas, declaradas no relatório.
const P = {
  contaExistente: Number(__ENV.P_CONTA_EXISTENTE || 0.5), // já tem conta de outro certame
  abandono: Number(__ENV.P_ABANDONO || 0.08),             // desiste no meio do formulário
  laudo: Number(__ENV.P_LAUDO || 0.1),                    // pede atendimento especial
  recarrega: Number(__ENV.P_RECARREGA || 0.15),           // aperta F5 na tela de status
  repeteCobranca: Number(__ENV.P_REPETE_COBRANCA || 0.1), // clica duas vezes em "gerar boleto"
  outraChave: Number(__ENV.P_OUTRA_CHAVE || 0.03),        // abre outra aba e gera de novo
  reenvioEvento: Number(__ENV.P_REENVIO || 0.05),         // provedor reenvia o aviso
};

// ------------------------------------------------------------------ perfil de chegada

// Taxas em chegadas POR MINUTO: por segundo, o arredondamento para inteiro
// distorcia a média em ~10% (conferido por script). As duas rampas de 5 s contam
// como meia rajada cada, por isso a rajada efetiva é RAJADA_S + 5.
function estagiosPatamar() {
  const rajadaEfetiva = RAJADA_S + 5;
  const fatorMedio = (CICLO_S - rajadaEfetiva + rajadaEfetiva * FATOR_RAJADA) / CICLO_S;
  const basePorMin = PATAMAR / 60 / fatorMedio;
  const picoPorMin = basePorMin * FATOR_RAJADA;
  const b = Math.round(basePorMin), p = Math.round(picoPorMin);
  const estagios = [{ target: b, duration: "60s" }]; // aquecimento
  const ciclos = Math.floor((DURACAO_MIN * 60) / CICLO_S);
  for (let i = 0; i < ciclos; i++) {
    estagios.push({ target: b, duration: `${CICLO_S - RAJADA_S - 10}s` });
    estagios.push({ target: p, duration: "5s" });
    estagios.push({ target: p, duration: `${RAJADA_S}s` });
    estagios.push({ target: b, duration: "5s" });
  }
  return { estagios, inicio: b, basePorS: basePorMin / 60, picoPorS: picoPorMin / 60 };
}

function estagiosEscada() {
  // Sobe até quebrar: patamares de 10 min, sem rajada, para achar a capacidade limpa.
  const degraus = (__ENV.DEGRAUS || "4000,8000,16700,25000,33400,50000,75000,100000")
    .split(",").map(Number);
  const estagios = [];
  for (const d of degraus) {
    estagios.push({ target: Math.round(d / 60), duration: "30s" });
    estagios.push({ target: Math.round(d / 60), duration: `${Number(__ENV.DEGRAU_MIN || 10) * 60 - 30}s` });
  }
  return { estagios, inicio: Math.round(degraus[0] / 60), basePorS: degraus[0] / 3600,
           picoPorS: degraus[degraus.length - 1] / 3600, degraus };
}

const perfil = MODO === "escada" ? estagiosEscada() : estagiosPatamar();

// Quantas inscrições o perfil planeja chegar, integrando a curva (interpolação linear
// entre estágios, como o executor faz). Vai para o resumo: o relatório compara com o real.
function planejadas(p) {
  let total = 0, atual = p.inicio, t = 0;
  for (const e of p.estagios) {
    const d = parseInt(e.duration);
    total += ((atual + e.target) / 2) * (d / 60);
    t += d; atual = e.target;
  }
  return { inscricoes: Math.round(total), horas: t / 3600, media_por_hora: Math.round(total / (t / 3600)) };
}

export const options = {
  scenarios: {
    inscricoes: {
      executor: "ramping-arrival-rate",
      startRate: perfil.inicio,
      timeUnit: "1m",
      preAllocatedVUs: Number(__ENV.VUS || 200),
      maxVUs: Number(__ENV.MAX_VUS || 3000),
      stages: perfil.estagios,
    },
  },
  thresholds: {
    // Os alvos da POC. Não abortam o ensaio: subir até quebrar exige ver o pós-quebra.
    "http_req_duration{fase:critica}": ["p(95)<2000"],
    conclusao: ["rate>0.98"],
  },
  summaryTrendStats: ["avg", "med", "p(90)", "p(95)", "p(99)", "max"],
};

// ------------------------------------------------------------------ métricas

const concluidas = new Counter("inscricoes_concluidas");
const falhas = new Counter("inscricoes_falhas");
const abandonos = new Counter("inscricoes_abandonadas");
const finalizadasOk = new Counter("finalizadas_ok");          // confere com o banco depois
const pagamentosAplicados = new Counter("pagamentos_aplicados");
const conclusao = new Rate("conclusao");                      // concluídas / (concluídas + falhas)
const duracaoFluxo = new Trend("duracao_fluxo_ms", true);
const naFila = new Trend("espera_na_fila_ms", true);

// ------------------------------------------------------------------ fluxo

const JSON_H = { "Content-Type": "application/json" };

function req(metodo, caminho, corpo, cab, nome, critica = true) {
  const params = { headers: Object.assign({}, JSON_H, cab || {}),
                   tags: { name: nome, fase: critica ? "critica" : "apoio" } };
  const b = corpo === null || corpo === undefined ? null
          : typeof corpo === "string" || corpo instanceof ArrayBuffer ? corpo : JSON.stringify(corpo);
  return http.request(metodo, `${BASE}${caminho}`, b, params);
}

// CPF determinístico pelo índice, com dígitos válidos. Contas novas na faixa 9xxxxxxxx;
// contas semeadas por app.semear na faixa 8xxxxxxxx, com a mesma fórmula.
function cpfSintetico(n, prefixo = 9) {
  const base = String(prefixo * 100000000 + (n % 99999999)).padStart(9, "0").split("").map(Number);
  for (const t of [10, 11]) {
    const s = base.reduce((a, d, i) => a + d * (t - i), 0);
    base.push(s % 11 < 2 ? 0 : 11 - (s % 11));
  }
  return base.join("");
}

function uuid() {
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    return (c === "x" ? r : (r & 0x3) | 0x8).toString(16);
  });
}

function falhou(motivo) {
  falhas.add(1, { motivo });
  conclusao.add(false);
}

function entrarNaFila() {
  // Só quando o nível 1 está ativo; o login responde 429 e manda para a fila.
  const t0 = Date.now();
  const s = req("POST", "/fila", null, {}, "fila_entrar", false).json("senha");
  for (let i = 0; i < 600; i++) {
    const r = req("GET", `/fila/${s}`, null, {}, "fila_consultar", false);
    if (r.status === 200 && r.json("admitido")) {
      naFila.add(Date.now() - t0);
      return r.json("passe");
    }
    sleep(Math.min(10, Math.max(1, (r.json("espera_estimada_s") || 2) / 2)));
  }
  return null;
}

function login(cpf) {
  let r = req("POST", "/sessoes", { cpf, senha: SENHA }, {}, "login");
  if (r.status === 429) {
    const passe = entrarNaFila();
    if (!passe) return null;
    r = req("POST", "/sessoes", { cpf, senha: SENHA }, { "X-Passe": passe }, "login");
  }
  return r.status === 200 ? { Authorization: `Bearer ${r.json("token")}` } : null;
}

export default function () {
  const t0 = Date.now();
  const n = exec.scenario.iterationInTest;

  // 1. Conta: reaproveita (conta única por CPF) ou cria.
  // Cada iteração tem índice único no ensaio, então cada conta semeada é usada uma vez.
  let cpf = null;
  if (n < CONTAS_SEMEADAS && Math.random() < P.contaExistente) cpf = cpfSintetico(n, 8);
  if (!cpf) {
    cpf = cpfSintetico(n, 9);
    const r = req("POST", "/candidatos",
      { cpf, nome: `Sintético ${n}`, email: `k6-${n}@exemplo.invalid`, senha: SENHA },
      {}, "cadastro");
    if (r.status !== 201 && r.status !== 409) return falhou(`cadastro_${r.status}`);
  }
  const h = login(cpf);
  if (!h) return falhou("login");

  // 2. Certame e início da inscrição.
  req("GET", `/certames/${CERTAME}`, null, h, "ver_certame");
  let r = req("POST", "/inscricoes", { certame_id: CERTAME }, h, "iniciar");
  if (r.status !== 201) return falhou(`iniciar_${r.status}`);
  const iid = r.json("id");

  // 3. Formulário, salvando a cada passo.
  if (!check(req("PATCH", `/inscricoes/${iid}`, { cargo_id: 1 }, h, "passo_cargo"), { "cargo": (x) => x.status === 200 }))
    return falhou("passo_cargo");
  if (Math.random() < P.abandono) { abandonos.add(1); return; } // abandono não é falha nem conclusão
  const laudo = Math.random() < P.laudo;
  req("PATCH", `/inscricoes/${iid}`, { cidade_id: 1, modalidade_id: 1, atendimento_especial: laudo }, h, "passo_cidade");
  if (laudo) {
    const arquivo = "%PDF-1.4\n" + "x".repeat(200 * 1024); // laudo de ~200 KB
    r = req("PUT", `/inscricoes/${iid}/laudo`, arquivo, Object.assign({}, h, { "Content-Type": "application/pdf" }), "laudo");
    if (r.status !== 201) return falhou(`laudo_${r.status}`);
  }
  req("PATCH", `/inscricoes/${iid}`, { aceitar_declaracoes: true }, h, "passo_declaracoes");

  // 4. Finalizar.
  r = req("POST", `/inscricoes/${iid}/finalizar`, null, h, "finalizar");
  if (r.status !== 200) return falhou(`finalizar_${r.status}`);
  finalizadasOk.add(1);

  // 5. Cobrança, com clique duplo e aba duplicada de propósito.
  const chave = uuid();
  r = req("POST", `/inscricoes/${iid}/cobranca`, { meio: "PIX" }, Object.assign({}, h, { "Idempotency-Key": chave }), "cobranca");
  if (r.status !== 200) return falhou(`cobranca_${r.status}`);
  const ref = r.json("referencia");
  if (Math.random() < P.repeteCobranca)
    req("POST", `/inscricoes/${iid}/cobranca`, { meio: "PIX" }, Object.assign({}, h, { "Idempotency-Key": chave }), "cobranca_repetida");
  if (Math.random() < P.outraChave)
    req("POST", `/inscricoes/${iid}/cobranca`, { meio: "PIX" }, Object.assign({}, h, { "Idempotency-Key": uuid() }), "cobranca_outra_aba");

  // 6. Espera o QR do Pix (registrado pelo worker), como a tela faria.
  for (let i = 0; i < 20; i++) {
    const c = req("GET", `/inscricoes/${iid}/cobranca`, null, h, "aguarda_qr");
    if (c.status === 200 && c.json("instrucoes")) break;
    sleep(1);
  }

  // 7. Pagamento: o provedor avisa; às vezes reenvia o mesmo aviso.
  const evento = `k6-${uuid()}`;
  r = req("POST", `/sandbox/pagar/${ref}?evento_id=${evento}`, null, {}, "webhook");
  if (r.status === 200 && r.json("resultado") === "APLICADO") pagamentosAplicados.add(1);
  if (Math.random() < P.reenvioEvento)
    req("POST", `/sandbox/pagar/${ref}?evento_id=${evento}`, null, {}, "webhook_reenvio");

  // 8. Consulta o status; parte recarrega.
  r = req("GET", `/inscricoes/${iid}`, null, h, "status");
  if (Math.random() < P.recarrega) for (let i = 0; i < 3; i++) req("GET", `/inscricoes/${iid}`, null, h, "recarrega");
  if (!check(r, { "paga": (x) => x.status === 200 && x.json("estado") === "PAGA" })) return falhou("nao_ficou_paga");

  concluidas.add(1);
  conclusao.add(true);
  duracaoFluxo.add(Date.now() - t0);
}

export function handleSummary(dados) {
  const quando = new Date().toISOString().replace(/[:.]/g, "-");
  const resumo = {
    modo: MODO, patamar_por_hora: PATAMAR, duracao_min: DURACAO_MIN,
    fator_rajada: FATOR_RAJADA, fator_rajada_confirmado: FATOR_RAJADA === FATOR_CONFIRMADO,
    rajada_s: RAJADA_S, ciclo_s: CICLO_S,
    chegada_base_por_s: perfil.basePorS, chegada_pico_por_s: perfil.picoPorS,
    probabilidades: P, inicio: __ENV.INICIO || null, planejado: planejadas(perfil),
  };
  return {
    [`relatorio/bruto/k6-${MODO}-${quando}.json`]: JSON.stringify({ perfil: resumo, k6: dados }, null, 2),
    stdout: `\nperfil: ${JSON.stringify(resumo)}\n` +
            `concluídas ${dados.metrics.inscricoes_concluidas?.values.count ?? 0} · ` +
            `falhas ${dados.metrics.inscricoes_falhas?.values.count ?? 0} · ` +
            `p95 crítico ${Math.round(dados.metrics["http_req_duration{fase:critica}"]?.values["p(95)"] ?? 0)} ms\n`,
  };
}
