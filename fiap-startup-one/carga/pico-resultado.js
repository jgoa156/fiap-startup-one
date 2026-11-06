// Pico 5: a base inteira abre o resultado ao mesmo tempo (02-produto §6, "toda a base acessa ao mesmo
// tempo"). Cada chegada é um candidato que já guardou o código do cartão no aparelho: deriva a chave
// do resultado e baixa a fatia estática. Nenhuma chamada à API — é isso que o ensaio confere, com o
// contador de transações do banco antes, durante e depois.
//
//   PUBLICO_URL  quem serve /publico (local: a API em :8082; na nuvem: o CloudFront)
//   CERTAME      o certame calculado
//   ID_MIN/ID_MAX faixa das inscrições homologadas
//   TOKEN_KEY    a LISURA_TOKEN_KEY do servidor. Atalho do disparador: no aparelho o código vem do
//                cartão guardado, aqui é recalculado para não precisar de 200 mil cartões.
//   TAXA         chegadas por segundo no patamar (padrão 1000) · DURACAO do patamar (padrão 2m)
import http from 'k6/http';
import crypto from 'k6/crypto';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const BASE = __ENV.PUBLICO_URL || 'http://localhost:8082';
const CERTAME = __ENV.CERTAME;
const ID_MIN = Number(__ENV.ID_MIN);
const ID_MAX = Number(__ENV.ID_MAX);
const CHAVE = __ENV.TOKEN_KEY;
const TAXA = Number(__ENV.TAXA || 1000);

if (!CERTAME || !ID_MIN || !ID_MAX || !CHAVE) {
  throw new Error('defina CERTAME, ID_MIN, ID_MAX e TOKEN_KEY');
}

const semLinha = new Counter('resultado_sem_linha');

export const options = {
  discardResponseBodies: false,
  scenarios: {
    pico5: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      preAllocatedVUs: Math.ceil(TAXA / 10),
      maxVUs: Math.ceil(TAXA / 2),
      stages: [
        { duration: '30s', target: TAXA },
        { duration: __ENV.DURACAO || '2m', target: TAXA },
        { duration: '10s', target: 0 },
      ],
    },
  },
  thresholds: {
    'http_req_failed': ['rate<0.001'],
    'http_req_duration{name:fatia}': ['p(95)<2000'],
    'resultado_sem_linha': ['count==0'],
  },
};

export default function () {
  const id = ID_MIN + Math.floor(Math.random() * (ID_MAX - ID_MIN + 1));
  // As mesmas contas de publicacao.Token e resultado.ChaveResultado.
  const token = crypto.hmac('sha256', CHAVE, 'cartao:' + id, 'hex').slice(0, 32);
  const chave = crypto.sha256('resultado:' + token, 'hex').slice(0, 24);
  const r = http.get(`${BASE}/publico/certames/${CERTAME}/resultados/${chave.slice(0, 2)}.json`, { tags: { name: 'fatia' } });
  // Procurar a chave no texto em vez de interpretar o JSON: a fatia tem centenas de KB, e interpretar
  // cada uma faria o k6 medir a própria CPU.
  const achou = r.status === 200 && r.body.includes(`"${chave}"`);
  if (r.status === 200 && !achou) semLinha.add(1);
  check(r, { 'fatia 200': (x) => x.status === 200, 'linha do candidato na fatia': () => achou });
}
