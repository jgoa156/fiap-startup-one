# 4. A arquitetura alvo

*(Atende ao item "Apresentação macro da nova arquitetura, em texto ou diagrama".)*

> **Diagrama:** [`diagramas/arquitetura-alvo.png`](diagramas/arquitetura-alvo.png) · fonte:
> [`diagramas/render-arquitetura.ps1`](diagramas/render-arquitetura.ps1)

> **Esta arquitetura já está paga.** Os R$ 900 mil/ano deste programa estão dentro das linhas de
> infraestrutura e de P&D do [plano de negócios](../fiap-startup-one/03-plano-de-negocios.md), e
> não são um orçamento à parte esperando aprovação. É por isso que a decisão de multicloud em
> [`03-defesa-multicloud.md`](03-defesa-multicloud.md) é tomada **na margem**, sobre o que já
> existe.

---

## 4.1 O princípio que organiza tudo

> **O calendário do edital é dado de entrada da infraestrutura.**

Esta é a frase que separa esta arquitetura de qualquer outra. Não escalamos reagindo a métrica.
**Escalamos por agenda**, porque a agenda está publicada com meses de antecedência e porque um
pico que sobe em 90 segundos não espera o autoescalonamento reativo perceber.

Dela derivam três consequências:

1. **Capacidade é provisionada antes, não descoberta durante.**
2. **A prova de que a capacidade existe é um ensaio de carga, não uma estimativa.**
3. **Quando ainda assim faltar, a degradação é uma fila honesta, não um erro 502.**

---

## 4.2 As quatro frentes

### Frente 0 · Isolamento, células por porte de certame

O que mais derruba o sistema hoje não é falta de capacidade global. É **um certame grande
consumindo o que era dos outros**.

| Tipo de célula | Quem fica nela | Quantas |
|---|---|---|
| **Compartilhada** | Bancas pequenas e certames de até 20 mil candidatos | 3 |
| **Dedicada** | Certames acima de 100 mil candidatos, alocada por evento e devolvida depois | sob demanda |

Cada célula é uma instalação completa: **cluster Amazon EKS próprio, Aurora próprio, cache
próprio**. Um certame que quebra sua célula não alcança as outras.

**A célula dedicada é o produto vendável:** a banca que vai rodar um certame de 300 mil candidatos
paga por isolamento garantido, e isso vira cláusula contratual, não favor operacional.

### Frente 1 · Escalonamento dirigido pelo calendário

```
Cronograma do edital  →  Serviço de agenda  →  provisiona 6 h antes do pico
      (dado do produto)                          devolve 6 h depois
```

- **Amazon EKS com Karpenter**, com *node pools* pré-aquecidos por agenda
- **Aurora PostgreSQL Serverless v2** com **capacidade mínima elevada por janela**, porque
  Serverless sozinho também sobe tarde demais
- **Réplicas de leitura criadas por agenda** antes dos picos 3, 4 e 5
- Ensaio de carga automático contra a mesma configuração, na véspera

### Frente 2 · Tirar a leitura do banco

Os três maiores picos em número de acessos, que são cartão de confirmação, gabarito e resultado,
são **leitura de dado que não muda**.

| Antes | Depois |
|---|---|
| Resultado consultado no Postgres, por candidato | **Resultado publicado como objeto estático no S3, servido por CDN** |
| Cartão de confirmação gerado no request | **Gerado em lote e publicado antes da liberação** |
| Gabarito consultado | **Arquivo estático** |

> **O efeito é desproporcional:** o pico 5, que é o maior em requisições, deixa de tocar o banco
> de dados. **Se todo o backend cair no dia do resultado, o resultado continua no ar.**

**É por isso que o número que dimensiona o banco é o pico de escrita**, de cerca de 16,7 mil
inscrições por hora num certame de 300 mil candidatos, e não o pico de leitura, que é maior em
requisições mas sai inteiro do CDN.

### Frente 3 · Degradação graciosa

- **Fila de espera na borda (Cloudflare Waiting Room)**, com admissão controlada, posição e tempo
  estimado visíveis. Uma fila honesta preserva o certame; um erro intermitente gera ação judicial
- **Geração assíncrona** de boleto, cartão e PDF, com **Amazon SQS e AWS Lambda**, nunca no request
- **Congelamento de implantação por calendário.** Nenhuma implantação enquanto houver janela
  crítica aberta em qualquer certame ativo

---

## 4.3 O que fica fora da AWS: o multicloud que adotamos, só para sobreviver

| Componente | Onde fica | Por quê |
|---|---|---|
| **DNS, WAF e fila de espera** | **Cloudflare** | Se a nuvem cai e o DNS está dentro dela, não dá nem para apontar para uma página de status. E o WAF é a defesa contra ataque, a dívida D9, que derrubou a FGV em fevereiro de 2026 |
| **Página de status** | **Provedor separado** (Statuspage ou equivalente) | Precisa sobreviver à queda que está anunciando |
| **Backup off-site** | **Google Cloud, São Paulo** (`southamerica-east1`), armazenamento de objetos com trava de imutabilidade | Backup no provedor que você quer sobreviver não é backup, é cópia |
| **Restauração a frio** | **Google Cloud, São Paulo**: GKE e Cloud SQL para PostgreSQL, criados só no ensaio ou na emergência | Se a AWS cair, é onde o sistema volta, em cerca de 8 h. Alternativa equivalente: Azure Brazil South |
| **Cópia da publicação estática** | **Google Cloud, São Paulo**, servida pelo Cloudflare se a origem na AWS cair | Sem ela, gabarito e resultado dependem do S3 de São Paulo, que cai junto com a região |
| **Repositório e CI** | **GitHub Actions** | Já é externo |
| **IaC** | **Terraform** | Portabilidade por código, não por réplica |

**Custo: R$ 170 mil/ano, 5,0% do EBITDA.** É a porção de multicloud que se paga.

> **Como funciona a restauração a frio.** No Google nada fica ligado: há o backup, a
> infraestrutura como código pronta para subir e a cota de máquinas pedida antecipadamente. Na
> emergência, o sistema sobe em **modo de recuperação**: inscrição, pagamento e consulta
> funcionam; boleto e PDF são gerados por um processo no próprio Kubernetes, lendo uma fila no
> PostgreSQL, no lugar de SQS e Lambda. É mais lento e basta. O mesmo procedimento é o ensaio
> trimestral de restauração, então o plantão o executa quatro vezes por ano. A escolha entre este
> destino e uma região AWS fora do Brasil está em
> [`03-defesa-multicloud.md` §3.10](03-defesa-multicloud.md).

---

## 4.4 O desenho

```
                              CANDIDATO
                                  │
        ┌─────────────────────────▼──────────────────────────┐
        │  CLOUDFLARE · DNS · WAF · Waiting Room             │   ← fora da AWS
        │  plano de controle neutro                          │
        └─────────────────────────┬──────────────────────────┘
                                  │
     ┌────────────────────────────┼────────────────────────────┐
     │                            ▼                            │
     │   AWS sa-east-1 · 3 AZs                                 │
     │                                                          │
     │   ┌──────────────────────────────────────────────────┐  │
     │   │  CloudFront + S3 · PUBLICAÇÃO ESTÁTICA           │  │
     │   │  gabarito · cartão de confirmação · resultado    │  │
     │   │  sobrevive à queda de todo o backend            │  │
     │   └──────────────────────────────────────────────────┘  │
     │                                                          │
     │   ┌────────────┐ ┌────────────┐ ┌────────────┐          │
     │   │ Célula 1   │ │ Célula 2   │ │ Célula 3   │          │
     │   │ EKS+Aurora │ │ EKS+Aurora │ │ EKS+Aurora │          │
     │   └────────────┘ └────────────┘ └────────────┘          │
     │   ┌──────────────────────────────────────────┐          │
     │   │ CÉLULA DEDICADA · certame > 100 mil      │          │
     │   │ alocada por evento, devolvida depois     │          │
     │   └──────────────────────────────────────────┘          │
     │                                                          │
     │   ┌──────────────────────────────────────────────────┐  │
     │   │  SQS + Lambda · geração assíncrona               │  │
     │   │  boleto · cartão · PDF · e-mail                  │  │
     │   └──────────────────────────────────────────────────┘  │
     │                                                          │
     │   ┌──────────────────────────────────────────────────┐  │
     │   │  SERVIÇO DE AGENDA                                │  │
     │   │  lê o cronograma do edital → provisiona antes    │  │
     │   └──────────────────────────────────────────────────┘  │
     └──────────────────────────┬───────────────────────────────┘
                                │
        ┌───────────────────────▼──────────────────────────┐
        │  BACKUP E RESTAURAÇÃO · Google Cloud, São Paulo  │   ← fora da AWS
        │  a frio: volta em ~8 h, ensaiado todo trimestre  │
        └──────────────────────────────────────────────────┘
        ┌──────────────────────────────────────────────────┐
        │  PÁGINA DE STATUS · provedor separado            │   ← fora da AWS
        └──────────────────────────────────────────────────┘
```

---

## 4.5 Modo degradado, o que acontece quando falta capacidade

Em ordem, automático:

| Nível | O que acontece | O candidato vê |
|---|---|---|
| **1** | Fila de espera ativada na borda | Posição e tempo estimado |
| **2** | Funções não críticas desligadas: painel da banca, relatórios | Nada; só a banca sente |
| **3** | Só escrita essencial: inscrição, pagamento, recurso | Resto em modo leitura |
| **4** | **Somente conteúdo estático.** Gabarito, cartão e resultado seguem no CDN | Consulta funciona; envio, não |
| **5** | Página de status fora da AWS, com aviso formal para prorrogação | Comunicado, não erro |

> **O nível 4 é o que torna a decisão de rejeitar a segunda nuvem sempre ligada defensável.** Mesmo com a
> região inteira fora do ar, o resultado e o gabarito continuam publicados, porque são arquivos,
> não consultas, e porque a cópia deles no Google Cloud é servida pelo Cloudflare quando a origem
> na AWS cai. **Se a região ficar fora por horas**, a restauração a frio no Google Cloud (§4.3) não
> é um nível automático: é decisão do plantão, e a página de status passa a mostrar a previsão de
> volta.

---

## 4.6 Migração em três ondas

| Onda | Quando | O quê | Resolve | R$/ano |
|---|---|---|---|---:|
| **1** | M1 a M4 | Fila de espera, geração assíncrona, publicação estática, congelamento por calendário | D3, D4, parcial D6 | 210 mil |
| **2** | M4 a M9 | Células, escalonamento por agenda, réplicas por calendário | D2, D7, D8 | 340 mil |
| **3** | M9 a M12 | Ensaio de carga contratual, restauração testada no Google Cloud, controle e backup fora da AWS | D1, D5, D9 | 350 mil |
| | | | **Total** | **R$ 900 mil** |

**A onda 1 entrega a maior parte do ganho, e é a mais barata.** Isso não é coincidência: as
falhas mais frequentes são as mais baratas de corrigir, o que é exatamente o argumento contra
gastar mais do que isso numa segunda nuvem sempre ligada.

> **Conferência da conta:** onda 1 = 120 (fila de espera) + 40 (geração assíncrona) + 50
> (publicação estática) + 0 (congelamento) = **210 mil**. Onda 2 = 280 (células) + 60
> (escalonamento por agenda) = **340 mil**. Onda 3 = 120 (ensaio de carga) + 60 (restauração
> testada) + 170 (controle, backup e restauração fora da AWS) = **350 mil**. Soma: **900 mil**, o total de
> [§3.10](03-defesa-multicloud.md).

---

## 4.7 Como se prova que funciona

| Prova | Frequência | Evidência |
|---|---|---|
| **Ensaio de carga a 40× a média do certame** | Antes de **cada** certame acima de 50 mil candidatos | Relatório assinado, anexado ao contrato |
| **Restauração completa no Google Cloud**, em modo de recuperação | Trimestral | Tempo até a primeira inscrição aceita. **Acima de 8 horas, dispara o gatilho 3** |
| **Exercício de degradação** | Trimestral, em produção, fora de janela crítica | Fila acionada e níveis 1 a 4 verificados |
| **Queda de célula** | Semestral | Nenhuma outra célula afetada |

> **O ensaio de carga contratualizado é a peça comercial mais forte do produto.** Nenhum
> concorrente anuncia isso, e é a única coisa que responde à pergunta que a banca realmente faz:
> *"você aguenta o meu último dia?"*

> **E ele é o que permite ser honesto sobre o contra C2.** O gerador de carga sintética existe e
> exercita o caminho. O que ele não produz é memória operacional num time de cinco pessoas, e é
> por isso que o argumento contra a segunda nuvem sempre ligada é de **operação**, não de impossibilidade de
> teste. Ver [`03-defesa-multicloud.md` §3.8](03-defesa-multicloud.md).
