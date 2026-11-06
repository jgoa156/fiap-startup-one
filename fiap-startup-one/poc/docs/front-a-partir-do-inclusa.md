# Front da Lisura a partir do Inclusa

**Decisão do usuário (30/10/2026):** o front da Lisura usa **o mesmo layout do Inclusa, com cores
diferentes**, em **React + Next.js**. **Paleta: 5 · Grafite** — tokens em
[`paletas/tokens-lisura.css`](./paletas/tokens-lisura.css). Fonte: `C:\projects\inclusa` (só documentação e mockups — o
Inclusa ainda não tem código de produto).

Este documento é o inventário do que dá para extrair do Inclusa **no estado em que está hoje**, em
quatro níveis: código pronto para copiar, desenho a reconstruir, regras a adotar e o que não serve.

---

## 1. Código pronto para copiar (existe e roda)

| Item | Onde está no Inclusa | Como entra na Lisura |
|---|---|---|
| **Tema do Ant Design** — tokens (cor primária, texto, bordas, fundo, Nunito, raios 8/12/14, `controlHeight: 44`, sombras) e componentes (`Menu` 44 px com raio 12 e cores de seleção, `Button` e `Input` em pílula, `Card` raio 14, `Tooltip` raio 8) | `mockups/antd-heroui-sidenav/index.html`, objeto `tema` | Arquivo `tema.ts` no Next, **igual na estrutura, com as cores da Lisura** |
| **Sombras e raios do HeroUI** como variáveis CSS (`--hero-shadow-sm/md/lg`, `--hero-radius-sm/md/lg`) | mesmo arquivo, bloco `:root` | Copiar sem mudança para o CSS global |
| **Moldura da aplicação**: lateral em cartão flutuante (248 px, recolhe para 72 px só com ícones e lembra a escolha no navegador), seções com título que viram traço quando recolhida, item ativo com fundo claro + **barra de marcador à esquerda**, cartão de perfil no rodapé com "Trocar perfil", barra superior flutuante com título, **busca `Ctrl+K`**, ajuda **`?`** e **sino com contador**; cartões com sombra que cresce no hover | mesmo arquivo, componente `App` + CSS `.sider`, `.topo`, `.perfil` | Componente `Moldura` do Next (componente de cliente), trocando a lista de módulos pela da Lisura |
| **Estrutura de tokens** (rampa da marca 50→800, cor de marcador, tinta, linha, estados, raios, altura de toque, anel de foco) | `docs/v4/design/tokens.css` | Mesmos nomes, **valores da Lisura** |

**Não aproveitável:** `docs/v4/design/mantine-theme.ts` (era para o Mantine, que perdeu para o Ant
Design na Q-16).

## 2. Desenho a reconstruir (mockups em SVG, sem código)

Cinco telas em `mockups/sistema-direcao-ab/`, todas na mesma moldura (exceto o login). São o padrão
de composição *dentro* da moldura. Cada uma vira componentes do Ant Design:

| Tela do Inclusa | Padrões que ela ensina | Peças do Ant Design |
|---|---|---|
| **Login** | Painel de marca à esquerda (faixa colorida + frase de valor + selos) e formulário à direita; **segmentado** para escolher o tipo de conta; `Sim`/`Não` em pílulas no lugar de checkbox; "Primeiro acesso — tenho um código"; aviso de ajuda com barra amarela; controles de **tamanho de fonte** e **alto contraste** no topo | `Segmented`, `Input`, `Button`, `Alert` |
| **Área de Trabalho** | Saudação + data/contexto; **Ações de hoje** em cartões acionáveis (o urgente com fundo de aviso); **indicadores** com ▲▼ do período e "cada número abre a lista filtrada"; **módulos** em tiles com contador; coluna direita com **agenda do dia** e **pendências**; cartão "Ajuda desta tela" | `Card`, `Statistic`, `Badge`, `List` |
| **Cadastro em passos** | Barra de passos com a linha de progresso em cor de marcador; selo **"Rascunho salvo há 12 s"**; aviso explicando que o passo é opcional; **chips** com contador ("2 selecionadas"); par de botões no lugar de checkbox; busca que vira **cartão do resultado**; segmentado com ícone + texto; rodapé `Voltar · Pular — preencher depois` + **um** botão primário | `Steps`, `Tag.CheckableTag`, `Segmented`, `CheckCard` (ProComponents), `Alert` |
| **Agendamento** | Barra de progresso "2 de 3"; **cartão de contexto** da pessoa com selos; **faixa de dias** com vagas em vez de calendário; segmentado com contagem; **blocos de horário**; aviso de bloqueio com **o que falta + botão para resolver**; `Voltar`, `Salvar rascunho` e um primário | `Progress`, `Card`, `Segmented`, `Button`, `Alert` |
| **Prontuário** | Cabeçalho fixo com avatar, **selos de alerta** e situação; abas; **trilha da jornada** (passos concluídos/atual/futuros); "Próximo compromisso" em destaque; coluna de **pendências** com ação; cartões de estado; lista com selo de situação; **linha do tempo** | `Descriptions`, `Tabs`, `Steps`, `Timeline`, `Tag` |

Também valem: `mockups/identidade-visual/05-alto-contraste.svg` (o modo alto contraste derivado dos
mesmos tokens) e `mockups/sidenav-opcoes/` (ícones de traço, nunca emoji).

## 3. Regras a adotar (documentação)

| Documento do Inclusa | O que vira regra na Lisura |
|---|---|
| `docs/v4/01-principios-ux.md` | **Escolher em vez de digitar** (tabela de substituição: chips, segmentados, cartões de escolha no lugar de checkbox, radio e select; busca com cartões acima de 12 opções); **uma ação primária por tela**; **rascunho automático**; **bloqueio sempre explica**; ajuda `?` em toda tela; estados vazios que ensinam; **WCAG 2.2 AA**, alvos ≥ 44 px, cor nunca sozinha, fonte ajustável; clique duplo desabilita o botão até o servidor responder |
| `docs/v4/09-casos-de-borda-transversais.md` §15 | As perguntas de revisão de toda tela: dois cliques, duas abas, internet caindo, celular, teclado, leitor de tela, quem não deve ver, o que muda no histórico, fuso |
| `docs/v4/README.md` — Definição de Pronto | Critérios de aceite, estados vazio/erro/carregando, acessibilidade, permissões, idempotência, rascunho, texto simples, testes |
| `docs/v4/08` — requisitos não funcionais | Estrutura da lista (desempenho em 4G, acessibilidade, responsividade, observabilidade, testabilidade, datas em UTC) |

## 4. O que não serve, e o que o próprio Inclusa ainda não resolveu

**Não serve à Lisura:** logotipo, nome e cores do Inclusa; todo o backlog de domínio (educação
especial, normas do MEC, regras de IA, concorrentes).

**Pendências do Inclusa que a Lisura não deve herdar:**

| Pendência no Inclusa | Na Lisura |
|---|---|
| Os mockups de tela ainda usam **emoji** como ícone (o próprio Inclusa decidiu trocar) | Ícones de traço do conjunto do Ant Design desde o início |
| Só há telas **desktop**; celular nunca foi desenhado | O candidato é **celular primeiro**: a moldura recolhe a lateral num menu deslizante (`Drawer`) e o conteúdo vira uma coluna |
| Contraste **não verificado** por ferramenta (o doc pede axe) | Rodar axe nas telas em código |
| O protótipo usa **antd 5.21.6**; o README já registra o **v6** | Usar a versão estável atual, conferida na instalação |
| Os docs 10 e 11 ainda citam Mantine; quem vale é o doc 17 | — |
| O tokens.css diz "fundo branco predominante"; o protótipo aprovado usa fundo cinza-claro `#F4F5F8` com cartões brancos | Seguir o protótipo (é o layout aprovado) |

## 5. Tela do Inclusa → tela da Lisura

| Inclusa | Lisura (candidato) | Lisura (banca, fora da POC) |
|---|---|---|
| Login (profissional / família) | Entrar (**candidato** / **banca**) | idem |
| Área de Trabalho | **Início do candidato**: ações de hoje ("Pague a taxa até 30/11", "Cartão de confirmação liberado"), minhas inscrições | Painel da banca: inscrições hoje, pagas, isenções a analisar, carga em relação ao pico |
| Cadastro em passos | **Inscrição em 7 passos** (cargo e cidade em cartões de escolha, cotas em chips, atendimento em chips + laudo, declarações em `Sim`/`Não`) | — |
| Agendamento | **Pagamento** (Pix/boleto em cartões de escolha, aviso de bloqueio) e **pedido de isenção** (hipótese em cartões) | — |
| Prontuário | **Página da inscrição**: cabeçalho com selos (isento, PcD, situação), abas (Resumo, Dados, Documentos, Pagamento, Histórico), trilha `Rascunho → Enviada → Paga → Em análise → Homologada`, próximo prazo, pendências e a trilha de auditoria como linha do tempo | Ficha do candidato para o analista |
| Lateral com 11 módulos | Poucos itens: Início, Certames, Minhas inscrições, Documentos, Minha conta | Módulos da banca |

## 6. Para começar o código

1. Projeto Next (App Router, TypeScript) com `antd`, `@ant-design/icons`, `@ant-design/pro-components`
   e **`@ant-design/nextjs-registry`** (sem ele os estilos do Ant Design piscam no primeiro
   carregamento, porque são gerados no navegador).
2. `tema.ts` + variáveis do HeroUI copiados do protótipo, **com a paleta da Lisura**.
3. `Moldura` portada do protótipo, com a versão de celular em `Drawer`.
4. Telas na ordem do fluxo do candidato, consumindo a API Go em `/api`.
