# Lisura — portal do candidato

Front da POC. **Layout igual ao do Inclusa** (`C:\projects\inclusa`, protótipo
`mockups/antd-heroui-sidenav`), com a **paleta 5 · Grafite** escolhida em 30/10/2026. Inventário do
que veio do Inclusa: [`../poc/docs/front-a-partir-do-inclusa.md`](../poc/docs/front-a-partir-do-inclusa.md).

| | |
|---|---|
| Framework | **Next.js 16.4.3** (App Router, Turbopack) · **React 19.3** · TypeScript |
| Componentes | **Ant Design 6.6.7** + `@ant-design/icons` + `@ant-design/nextjs-registry` |
| Documentação | **Storybook 10.6.3** (`/design`), **Scalar 0.9.82** (`/docs/api`), **Mermaid 12.2** (diagramas de `/docs/infra`) |
| Tema | [`src/tema/tema.ts`](src/tema/tema.ts) (tokens do Ant Design) · [`src/tema/tokens.css`](src/tema/tokens.css) (variáveis CSS) — raios e sombras do HeroUI, Nunito, controles de 44 px |
| API | A API Go em `../poc`, repassada pelo Next em `/api` (`next.config.ts`) — mesmo endereço, cookie de sessão viaja sozinho |

## Rodar

Com a API Go no ar (`../poc/README.md`, porta 8082):

```powershell
npm install
npm run dev -- -p 8080     # http://localhost:8080
```

`LISURA_API` muda o endereço da API (padrão `http://localhost:8082`); `LISURA_ENSAIADOR`, o do
executor de ensaios (padrão `http://127.0.0.1:8083`); `NEXT_PUBLIC_GRAFANA_URL`, o do Grafana
(padrão `http://127.0.0.1:3000`).

```powershell
npm run storybook          # o sistema de design para editar, em http://localhost:6006
npm run design             # o Storybook estático em public/design, que o Next serve em /design
npm run build              # produção (standalone); depois copiar .next\static e public para .next\standalone
```

**Para medir carga, use o build de produção, nunca o `next dev`**, e suba com
`HOSTNAME=127.0.0.1`: com `localhost`, o Next escuta só em IPv6 e o k6, que disca IPv4, recebe
recusa.

**Dependências com pelo menos 2 semanas:** o front foi reinstalado em 06/11 com
`npm install --before=2026-10-23T20:00:00Z`, que limita também as indiretas. O antd estava em 6.6.8
(26/10) e foi fixado em 6.6.7. O npm 11 não roda scripts de instalação sem aprovação; os de
`esbuild`, `unrs-resolver` e `vue-demi` ficaram sem rodar, e o build funciona sem eles.

`npm run lint`, `npx tsc --noEmit`, `npm run design` e `npm run build` passam limpos (06/11).

## Telas

| Caminho | Padrão do Inclusa | O que faz |
|---|---|---|
| `/entrar` | Login | Painel da marca + formulário; candidato ou banca num segmentado; entrar com CPF ou criar conta |
| `/` | Área de Trabalho | Saudação, **ações de hoje** (pagar, terminar inscrição), minhas inscrições, certames abertos |
| `/certames` | Lista | Busca pelo `Ctrl+K` da barra superior |
| `/certames/[id]` | Agendamento | Cargo em **cartões de escolha**, começar inscrição, pedido de isenção (CadÚnico ou doador, com envio do comprovante), cronograma, cargos e vagas |
| `/inscricoes/[id]` | Cadastro em passos + Prontuário | Em rascunho: **7 passos, cada um salvo ao mudar**, selo "Rascunho salvo", revisão que leva ao que falta. Enviada: situação, **Pix/boleto** (com simulação de pagamento no sandbox), **devoluções** (com a chave Pix quando o CPF não é chave), **avisos por e-mail** com a prova de cada um (o que voltou pede um e-mail novo ali mesmo), histórico |

Documentação e operação (sem login, grupo `(docs)` com barra própria):

| Caminho | O que faz |
|---|---|
| `/docs` | A entrada da documentação viva, com o que mantém cada página viva |
| `/docs/api` | A referência da API (Scalar) a partir de `GET /api/openapi.yaml`; telemetria, agente de IA e MCP do Scalar desligados |
| `/design` | Redireciona para `/design/index.html`: o Storybook estático (os arquivos dele são relativos e não abrem de `/design` sem a barra) |
| `/docs/infra` | O caminho de um pedido, cada peça local e na AWS, e o Terraform lido de `public/docs/infra.json` |
| `/operacao` | Dispara ensaios pelo executor (`/ensaiador`), muda o nível de degradação, embute o Grafana. **Quebrou no navegador em 06/11**; corrigida no código, não conferida de novo |

Histórias do Storybook ficam junto do componente (`*.stories.tsx`) ou em `src/stories`. O
`.storybook/preview.tsx` simula a API por história (`parameters.api`), então os estados raros (aviso
que voltou, chave recusada, fila no pico) aparecem sem servidor.

Moldura ([`src/componentes/Moldura.tsx`](src/componentes/Moldura.tsx)): lateral em cartão flutuante
que recolhe para ícones e lembra a escolha; no celular vira gaveta. Barra superior com busca
`Ctrl+K` e ajuda `?` (cada tela declara a sua com `useAjuda`). Cartões de escolha no lugar de
radio/checkbox/select ([`src/componentes/Escolha.tsx`](src/componentes/Escolha.tsx)), com
`aria-pressed` e ✓ — cor nunca sozinha.

## Verificado no navegador (04/11/2026)

Desktop 1440 × 900 e celular 375 px (sem rolagem lateral, gaveta funcionando), console limpo:
entrar → certame → cargo → 7 passos (cota PcD exige laudo; laudo compartilhado) → enviar → Pix →
simular pagamento → **"Paga"**, com "Pagamento confirmado" no histórico. Sessão vencida volta ao
login e, depois de entrar, ao endereço de antes (busca incluída).

**Não verificado clicando:** o envio de arquivo (o seletor de arquivos do sistema não é automatizável
aqui; o laudo usado foi enviado pela API) e o modo escuro (o tema ainda não tem um).

## Decisões

- **Tudo roda no navegador.** Cada tela busca `/api` depois de abrir, com o cookie de sessão
  `HttpOnly`. Por isso o modo de cache do Next (`cacheComponents`), que vinha ligado no gerador, foi
  **desligado**: ele serve a páginas renderizadas no servidor e só acusava o `Math.random()` interno
  do Ant Design.
- **Ant Design 6 mudou nomes:** `Alert.title` (não `message`), `Steps` item `content`, `Timeline`
  item `content`, `Spin.description`, `Space.orientation`, `Drawer.size`, `Button.iconPlacement`.
- `AGENTS.md` é gerado pelo próprio `next dev`.
