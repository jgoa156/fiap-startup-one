# 2. A arquitetura da fase startup

*(Atende ao item "Arquitetura adotada durante a fase de startup, em texto ou diagrama".)*

> **Nota para a apresentação:** este inventário **não vai para o deck.** A plateia nunca viu esta
> empresa, e listar componentes de uma arquitetura que ninguém conhece é tempo perdido em uma
> defesa de 5 minutos. Ele vira **uma linha de delta** no slide de contexto ("com 3 bancas uma
> instância bastava; com 45 e certames de 300 mil, não basta") e permanece completo aqui, que é
> onde o enunciado o exige.

---

## 2.1 O contexto do ano 1

3 bancas clientes. 100 mil inscrições no ano inteiro. Certames de 2 a 8 mil candidatos. Uma
pessoa em infraestrutura, que também escrevia produto.

**A decisão correta naquele momento era a mais simples possível**, e foi.

---

## 2.2 O desenho

```
                    Internet
                       │
                       ▼
            ┌──────────────────────┐
            │  Route 53 (DNS)      │
            └──────────┬───────────┘
                       ▼
            ┌──────────────────────┐
            │  CloudFront          │  só estáticos: CSS, JS, imagem
            └──────────┬───────────┘
                       ▼
            ┌──────────────────────┐
            │  ALB                 │
            └──────────┬───────────┘
                       ▼
     ┌─────────────────────────────────────┐
     │  ECS Fargate · 2 a 6 tarefas        │
     │  monólito modular                   │
     │  · inscrição   · isenção            │
     │  · recurso     · ensalamento        │
     │  · resultado   · painel da banca    │
     │  · geração de PDF  ← SÍNCRONA       │
     └───────┬──────────────────┬──────────┘
             ▼                  ▼
   ┌──────────────────┐  ┌──────────────┐
   │ RDS PostgreSQL   │  │  S3          │
   │ db.r6g.large     │  │  documentos  │
   │ Multi-AZ         │  │  laudos      │
   │ SEM réplica de   │  └──────────────┘
   │ leitura          │
   └──────────────────┘
             │
             ▼
   ┌──────────────────┐
   │ SQS → e-mail     │  única coisa assíncrona
   └──────────────────┘

   Pagamento: API do provedor da própria banca (nunca processamos)
   Observabilidade: CloudWatch
   Implantação: GitHub Actions, janela de ~10 min
```

### Componentes, com produto nomeado

| Camada | Produto | Por que este |
|---|---|---|
| DNS | **Amazon Route 53** | Já estava na conta; zero decisão |
| CDN | **Amazon CloudFront** | Só estáticos, porque o HTML era todo dinâmico |
| Balanceamento | **AWS ALB** | Padrão |
| Computação | **AWS Fargate (ECS)** | Sem servidor para gerenciar, com 1 pessoa de infra |
| Banco | **Amazon RDS PostgreSQL**, Multi-AZ | Relacional porque certame é relacional: candidato, cargo, cota, nota |
| Objetos | **Amazon S3** | Laudos e comprovantes |
| Fila | **Amazon SQS** | Só e-mail |
| Observabilidade | **Amazon CloudWatch** | Suficiente na escala |
| IaC | **Terraform** | Desde o dia 1, e foi a decisão mais barata e mais valiosa |
| Pagamento | **API do PSP da banca** | Decisão de negócio: taxa de inscrição é receita pública, e há entendimento de controle contra delegar sua gestão a empresa privada |

---

## 2.3 As dívidas assumidas, e conhecidas

Nenhuma destas foi acidente. Todas foram trocas conscientes por velocidade, e todas viraram a
pauta do ano 5.

| # | Dívida | O que ela custa quando o pico chega |
|---|---|---|
| **D1** | **Sem ensaio de carga** | A capacidade é descoberta **em produção, no pior dia possível** |
| **D2** | **Banco único, sem réplica de leitura** | O pico de resultado (leitura em massa) compete com a inscrição (escrita) no mesmo Postgres |
| **D3** | **Geração de PDF síncrona no request** | Cartão de confirmação e boleto gerados na hora consomem toda a CPU exatamente no pico 3 |
| **D4** | **Sem fila de espera** | Todos entram, todos degradam, **ninguém completa**. É pior que recusar |
| **D5** | **Backup nunca restaurado** | Existe backup automático do RDS. **Nunca foi testada uma restauração** |
| **D6** | **Implantação com janela** | Impossível corrigir um bug com um certame em andamento |
| **D7** | **Inquilino compartilhado** | Um certame de 150 mil candidatos degrada os outros 20 certames ativos |
| **D8** | **Escalonamento reativo** | O autoescalonamento responde a métrica. Num pico que sobe em 90 segundos, **ele chega tarde** |
| **D9** | **Sem proteção de borda** | Não há WAF nem mitigação de negação de serviço: o desenho vai de Route 53 e CloudFront direto ao ALB. **Um ataque derruba o site como um pico derrubaria**, e o backup, na mesma conta AWS, pode ser alcançado por uma invasão |

> **D9 foi acrescentada em 22/09/2026**, depois do caso FGV, cujo site ficou dias fora do ar em
> fevereiro de 2026 após ataque hacker. Ela estava no desenho desde o início, só não estava na
> lista.

> **D1, D4, D7 e D8 são as que explicam as manchetes.** Nenhuma delas é resolvida trocando ou
> somando provedor de nuvem, e é esse fato que estrutura a defesa em
> [`03-defesa-multicloud.md`](03-defesa-multicloud.md).

---

## 2.4 O que já estava certo

Vale registrar, porque a arquitetura alvo não é uma reescrita:

- **Terraform desde o início.** A migração do ano 5 é incremental, não um recomeço
- **Multi-AZ no banco** desde cedo, então a classe de falha mais comum já estava coberta
- **Não processar pagamento**, decisão de negócio que evitou uma camada inteira de exigência
- **Monólito modular, não microsserviços.** Com 9 pessoas, teria sido suicídio distribuir
- **Postgres relacional.** Certame tem integridade referencial dura; documento livre teria doído

---

## 2.5 O delta que vai para o slide

| | Ano 1 | Ano 5 |
|---|---|---|
| Bancas | 3 | **45** |
| Maior certame | 8 mil candidatos | **300 mil** |
| Pico sobre a média *do certame* | ~10× | **~40×** |
| Certames simultâneos | 1 a 2 | **dezenas** |
| Time de plataforma | 1 | **5** |

**Uma instância bastava. Agora não basta, e a pergunta é o que fazer com isso.**
