# Trabalho Final — Soluções Multicloud

**Empresa analisada:** **Lisura** (projeto Startup One) — plataforma de inscrição e gestão de
processos seletivos públicos, vendida para bancas organizadoras.
**Apresentação:** 4 a 6 minutos de defesa + 2 a 4 minutos de discussão · **1 apresentador**.

---

## Tese em uma frase

> **Não. A Lisura não deve adotar multicloud** — e a razão não é custo, é que **multicloud não
> resolve a falha que de fato nos derruba.**
>
> O que tira um sistema de inscrição do ar no último dia do prazo não é o provedor cair: é a nossa
> própria capacidade acabar. **Das sete causas reais de indisponibilidade, multicloud cobre uma** —
> e é a mais rara. Uma segunda nuvem rodando o mesmo software mal dimensionado cai igual.
>
> **Com uma ressalva que adotamos:** DNS, WAF, página de status e backup saem do provedor
> primário. Não é arquitetura multicloud — é não ter ponto único no **controle** e na **última
> linha de defesa**. Custa R$ 162 mil/ano, 3% do EBITDA.

**E o argumento não é econômico:** a segunda nuvem custaria R$ 1,66 mi/ano; o programa que
escolhemos custa R$ 1,69 mi/ano. **Custam o mesmo.** A escolha é sobre o que o dinheiro compra.

---

## Para apresentar

| Arquivo | O que é |
|---|---|
| **[`apresentacao/Lisura-multicloud-slides.pdf`](apresentacao/Lisura-multicloud-slides.pdf)** | **Os slides** — 8 páginas, 16:9 (960×540 pt, widescreen do PowerPoint) |
| **[`apresentacao/Lisura-multicloud-roteiro.pdf`](apresentacao/Lisura-multicloud-roteiro.pdf)** | **O roteiro do apresentador** — miniatura do slide, fala palavra a palavra, marcação |
| [`05-roteiro-apresentacao.md`](05-roteiro-apresentacao.md) | Régua de tempo, ordem de corte e **banco de 17 perguntas** para a discussão |
| [`apresentacao/previews/`](apresentacao/previews/) | Os 8 slides em PNG, para revisar sem abrir o PDF |
| [`apresentacao/deck-lisura.html`](apresentacao/deck-lisura.html) | A **fonte** dos slides e do roteiro — edite aqui |
| [`apresentacao/build-pdf.ps1`](apresentacao/build-pdf.ps1) | Regera os dois PDFs a partir dela (Edge headless — sem Python nem Node) |

---

## Mapa dos entregáveis × exigências do enunciado

| Exigência | Onde está |
|---|---|
| Breve descrição do projeto/empresa e área de atuação | [01-empresa-e-contexto.md](01-empresa-e-contexto.md) · slide 1 |
| Arquitetura adotada durante a fase startup (texto ou diagrama) | [02-arquitetura-fase-startup.md](02-arquitetura-fase-startup.md) |
| **Defesa da adoção (ou não) de multicloud, com prós e contras baseados nos requisitos de negócio** | [03-defesa-multicloud.md](03-defesa-multicloud.md) — §3.7 e §3.8 · slides 3, 4, 5, 6 |
| Apresentação macro da nova arquitetura (texto ou diagrama) | [04-arquitetura-alvo.md](04-arquitetura-alvo.md) · [`diagramas/`](diagramas/) · slide 7 |
| **Referências de institutos de mercado (Forrester e outros) com dados** | [06-referencias.md](06-referencias.md) — Forrester/HashiCorp 2024, Flexera 2025, Gartner · slide 4 |
| **Exemplos de soluções de mercado nas arquiteturas** | [06-referencias.md §6.6](06-referencias.md) — cada componente nomeia produto e fornecedor |
| Controle de tempo (4 a 6 min) | 8 slides cronometrados em **5 min 30 s**, com ordem de corte |

---

## Documentos

| Arquivo | Conteúdo |
|---|---|
| [01-empresa-e-contexto.md](01-empresa-e-contexto.md) | A empresa, o setor, **os cinco picos de um certame**, a janela da Lei 14.965/2024 e os números do ano 5 |
| [02-arquitetura-fase-startup.md](02-arquitetura-fase-startup.md) | Arquitetura AWS da fase startup e as **8 dívidas** assumidas |
| [03-defesa-multicloud.md](03-defesa-multicloud.md) | **O núcleo** — as 7 causas reais, prós e contras com veredito, os 3 modelos, a conta e os gatilhos |
| [04-arquitetura-alvo.md](04-arquitetura-alvo.md) | Arquitetura alvo: células, escalonamento pelo calendário do edital, publicação estática, modo degradado em 5 níveis, plano de controle neutro |
| [05-roteiro-apresentacao.md](05-roteiro-apresentacao.md) | Régua de tempo + banco de perguntas |
| [06-referencias.md](06-referencias.md) | Institutos, legislação, entendimento de controle, concorrência e soluções de mercado |
| [`diagramas/`](diagramas/) | `arquitetura-alvo.png` e `.jpg` (2880×2040) · `render-arquitetura.ps1` para regerar |

---

## Status

- [x] Conteúdo dos quatro tópicos exigidos
- [x] Diagrama da arquitetura alvo, com PNG/JPG renderizados
- [x] Slides e roteiro em PDF, 8 slides cronometrados em 5m30s
- [x] Banco de 17 perguntas para a discussão
- [x] Referências de institutos, mapeadas ao dado que cada uma sustenta
- [x] Concorrência verificada (set/2026)
- [ ] **Ensaio com cronômetro, 3×** — o tempo é critério explícito de avaliação
- [ ] Reverificar os 4 itens da [§6.7](06-referencias.md) na véspera
