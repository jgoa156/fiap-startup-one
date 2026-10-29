# Trabalho Final: Soluções Multicloud

**Empresa analisada:** **Certamis** (projeto Startup One), plataforma de inscrição e gestão de
processos seletivos públicos, vendida para bancas organizadoras.
**Apresentação:** 4 a 6 minutos de defesa mais 2 a 4 minutos de discussão · **1 apresentador**.

---

## Tese em uma frase

> **Uma nuvem para operar, duas para sobreviver.** A Certamis não deve operar em duas nuvens. A
> razão não é custo. É que **operar em duas não resolve a falha que de fato nos derruba.**
>
> O que tira um sistema de inscrição do ar no último dia do prazo não é o provedor cair: é a nossa
> própria capacidade acabar. **Das oito falhas que derrubam este produto, sete são dívidas
> nossas**, e nenhuma delas muda de natureza por existir um segundo provedor.
>
> **E deve sobreviver em duas:** DNS, WAF, página de status, backup e a restauração de emergência
> ficam fora da AWS. Se a região inteira cair, o sistema volta a frio no **Google Cloud, em São
> Paulo**, em cerca de 8 horas. Custa R$ 170 mil/ano, 5% do EBITDA.

**A pergunta é formulada na margem, e é aí que ela se resolve.** O endurecimento da nuvem única
custa R$ 900 mil/ano e **já está no orçamento**: ele é o piso, não uma alternativa. Sobre ele, a
segunda nuvem sempre ligada custaria **R$ 1,22 mi/ano a mais** e compraria 4 horas de RTO num evento estimado a
cada 4 anos, ou seja **R$ 1,22 milhão por hora de indisponibilidade evitada**, numa falha que quase
sempre se resolve reabrindo um prazo. Nos três casos de falha em certame documentados em 2026
(Alece, AgSUS, FGV), **nenhum foi queda do provedor de nuvem.**

---

## Para apresentar

| Arquivo | O que é |
|---|---|
| **[`apresentacao/Certamis-multicloud-slides.pdf`](apresentacao/Certamis-multicloud-slides.pdf)** | **Os slides**: capa mais 8 slides, 16:9 (960×540 pt, widescreen do PowerPoint) |
| **[`apresentacao/Certamis-multicloud-roteiro.pdf`](apresentacao/Certamis-multicloud-roteiro.pdf)** | **O roteiro do apresentador**, com miniatura do slide, fala palavra a palavra e marcação |
| [`05-roteiro-apresentacao.md`](05-roteiro-apresentacao.md) | Régua de tempo, ordem de corte e **banco de 30 perguntas** para a discussão |
| [`apresentacao/previews/`](apresentacao/previews/) | Os slides em PNG, para revisar sem abrir o PDF. `slide-01.png` é a capa; o slide N está em `slide-0(N+1).png` |
| [`apresentacao/build-pdf.ps1`](apresentacao/build-pdf.ps1) | Regera os dois PDFs e os previews a partir de `deck.html` |

---

## Mapa dos entregáveis contra as exigências do enunciado

| Exigência | Onde está |
|---|---|
| Breve descrição do projeto/empresa e área de atuação | [01-empresa-e-contexto.md](01-empresa-e-contexto.md) · slide 1 |
| Arquitetura adotada durante a fase startup (texto ou diagrama) | [02-arquitetura-fase-startup.md](02-arquitetura-fase-startup.md) |
| **Defesa da adoção (ou não) de multicloud, com prós e contras baseados nos requisitos de negócio** | [03-defesa-multicloud.md](03-defesa-multicloud.md) §3.7 e §3.8 · slides 3, 4, 5, 6 |
| Apresentação macro da nova arquitetura (texto ou diagrama) | [04-arquitetura-alvo.md](04-arquitetura-alvo.md) · [`diagramas/`](diagramas/) · slide 7 |
| **Referências de institutos de mercado (Forrester e outros) com dados** | [06-referencias.md](06-referencias.md), com Forrester/HashiCorp 2024, Flexera 2025 e Gartner · slide 4 |
| **Exemplos de soluções de mercado nas arquiteturas** | [06-referencias.md §6.6](06-referencias.md), onde cada componente nomeia produto e fornecedor |
| Controle de tempo (4 a 6 min) | Fala medida em **822 palavras, 5 min 28 s a 150 palavras/min**, capa incluída, com ordem de corte |

---

## Documentos

| Arquivo | Conteúdo |
|---|---|
| [01-empresa-e-contexto.md](01-empresa-e-contexto.md) | A empresa, o setor, **os cinco picos de um certame**, a definição exata do "40× o pico", a janela da Lei 14.965/2024 e os números do ano 5 |
| [02-arquitetura-fase-startup.md](02-arquitetura-fase-startup.md) | Arquitetura AWS da fase startup e as **9 dívidas** assumidas, que são a origem das falhas do slide 3 |
| [03-defesa-multicloud.md](03-defesa-multicloud.md) | **O núcleo:** a reformulação da pergunta, as 8 falhas e os casos reais de 2026, prós e contras com veredito, os 3 modelos e a conta marginal |
| [04-arquitetura-alvo.md](04-arquitetura-alvo.md) | Arquitetura alvo: células, escalonamento pelo calendário do edital, publicação estática, modo degradado em 5 níveis, controle, backup e restauração fora da AWS |
| [05-roteiro-apresentacao.md](05-roteiro-apresentacao.md) | Régua de tempo e banco de perguntas |
| [06-referencias.md](06-referencias.md) | Institutos, legislação, entendimento de controle, concorrência, soluções de mercado, e **§6.8: o que o trabalho afirma sem fonte externa e por quê** |
| [`diagramas/`](diagramas/) | `arquitetura-alvo.png` (2880×2040) · `render-arquitetura.ps1` para regerar |

---

## Status

- [x] Conteúdo dos quatro tópicos exigidos
- [x] Diagrama da arquitetura alvo renderizado
- [x] Slides e roteiro em PDF, fala medida em 822 palavras com a capa (5 min 28 s a 150 palavras/min)
- [x] Banco de 30 perguntas para a discussão
- [x] Referências de institutos, mapeadas ao dado que cada uma sustenta
- [x] Concorrência verificada (set/2026)
- [x] Aritmética conferida linha a linha (set/2026)
- [x] **Revisão adversarial dos argumentos** (set/2026), com a comparação refeita na margem
- [ ] **Ensaio com cronômetro, 3×.** O tempo é critério explícito de avaliação
- [x] Reverificação da véspera, com leitura das fontes primárias (22/09/2026, [§6.7](06-referencias.md))
- [x] **Destino da restauração decidido** (23/09/2026): Google Cloud em São Paulo, a frio, R$ 80 mil/ano ([§3.10](03-defesa-multicloud.md))
