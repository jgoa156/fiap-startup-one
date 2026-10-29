# Handoff: implementar a POC do Startup One (Certamis)

*Escrito em 29/10/2026 para quem pega a implementação a frio. Todos os caminhos são relativos à
raiz do repositório [jgoa156/fiap-startup-one](https://github.com/jgoa156/fiap-startup-one). Complementa, não substitui,
[`CONTINUAR-AQUI.md`](../CONTINUAR-AQUI.md), que é a fonte das decisões e dos números.*

---

## 1. Onde estamos

| Item | Estado |
|---|---|
| Ideação, produto, plano de negócios, pitch | **Prontos** desde 23/09/2026 |
| Trabalho de multicloud | **Pronto**, pendente só o ensaio com cronômetro |
| **POC** | **Não começou.** Não existe código nem pasta para ela |
| Repositório | [jgoa156/fiap-startup-one](https://github.com/jgoa156/fiap-startup-one), privado, com os dois trabalhos e este handoff desde 29/10/2026 |
| Pasta local | Na máquina original, `D:\projects\fiap\FIAP-Lisura`; a renomeação para `FIAP-Certamis` continua pendente |

O trabalho do Startup One **exige a startup implementada como POC**. É a maior pendência do
projeto, e o escopo já está fechado em `fiap-startup-one/04-escopo-poc.md`: 1 pessoa, 8
semanas, ~US$ 100.

---

## 2. O que a POC precisa provar (não relitigar)

> **H1: o sistema absorve 16,7 mil inscrições por hora, sustentadas por 3 horas, sem perder
> inscrição, sem cobrar duas vezes e sem estourar o custo previsto de infraestrutura.**

- **O entregável é um relatório de ensaio de carga**, não uma tela.
- **Subir até quebrar, sempre.** O ponto de quebra é a métrica mais importante.
- **Resultado negativo também é entregável**, desde que diga onde e por que quebrou.
- **Se faltar tempo, cai o painel da banca, nunca o ensaio.**
- Escopo: um certame pequeno inteiro, só prova objetiva, com o caminho crítico endurecido.
- Dado de candidato **sempre sintético**. Nunca dado real.
- Pagamento: **sandbox**, orquestrado, nunca processado por nós.
- Sem IA no produto.

### Métricas de aceite

| Métrica | Alvo |
|---|---|
| Inscrições perdidas | zero |
| Cobranças duplicadas sob repetição e reenvio | zero |
| p95 no pico | < 2 s |
| Conclusão do fluxo completo | > 98% |
| Ponto de quebra | medido |
| Custo por hora de pico | medido, em US$ e R$ |

---

## 3. Uma observação crítica antes de começar

**16,7 mil inscrições por hora são 4,6 por segundo em média. Isso é pouco para um Postgres.**
Se o gerador de carga distribuir as inscrições de forma uniforme, a POC passa no primeiro dia e
não prova nada, que é exatamente o risco "passou no alvo sem chegar ao limite" da §8 do escopo.

O que torna o ensaio honesto, e já está previsto no escopo, mas precisa virar número:

1. **Perfil de chegada concentrada.** Dentro da hora de pico, rajadas de 90 segundos bem acima
   da média (é a dívida D8). Sugestão: medir também o pico por **segundo**, não só a média por
   hora, e declarar no relatório qual rajada foi usada.
2. **Requisições por inscrição, não inscrições.** Uma inscrição completa são várias chamadas
   (login, cadastro, cargo, cota, upload, pagamento, consulta de status), mais recarregamento e
   abandono. O relatório deve dar as duas unidades.
3. **Repetição e reenvio deliberados**, para testar a idempotência do pagamento sob carga.

Isso **não muda H1**. Muda como o gerador é desenhado na semana 6. Confirmar com o usuário antes
de fixar o fator de rajada, porque é um número que vai para o relatório e para o pitch.

---

## 4. Arquitetura da POC

É a arquitetura da fase startup (`fiap-multicloud/02-arquitetura-fase-startup.md` §2.2) **já com
as correções que a POC precisa demonstrar**. A POC é a primeira instância da entrega, não um
protótipo descartável.

| Camada | Produto | Dívida que a POC endereça |
|---|---|---|
| Borda | CloudFront | |
| Fila de espera | Admissão controlada na frente da aplicação | **D4** |
| Computação | ECS Fargate, monólito modular, escala a zero fora dos ensaios | |
| Escalonamento | **Por agenda**, subir antes do pico e devolver depois | **D8** |
| Banco | RDS PostgreSQL, instância pequena, pool de conexões explícito | **D2** (medir) |
| Assíncrono | SQS + worker para boleto e cartão | **D3** |
| Publicação estática | Resultado gerado como arquivo no S3, servido pelo CloudFront | pico de leitura fora do banco |
| IaC | Terraform | ambiente reproduzível |
| Carga | k6 (open source) | **D1** |
| Observabilidade | CloudWatch | |

Modo degradado na POC: **níveis 1, 4 e 5** de `fiap-multicloud/04-arquitetura-alvo.md` §4.5
(fila de espera, só estático, página de status).

### Decisão de linguagem, a confirmar na primeira sessão

Nenhum documento fixa a linguagem. **Recomendação: Python 3.13 com FastAPI**, porque Python já
está instalado nesta máquina e Node não está. Alternativas aceitáveis se o usuário preferir:
.NET ou Go. **Perguntar antes de criar o projeto.**

---

## 5. Plano de execução

Mantém o cronograma de 8 semanas do escopo. Cada semana tem um **critério de pronto verificável**.

| Sem. | Entrega | Pronto quando |
|---|---|---|
| **1** | Repositório, esqueleto do monólito, modelo de dados do certame, **máquina de estados da inscrição** (§5 do produto), trilha de auditoria | Migrações sobem do zero; teste prova que transição inválida é recusada e que toda transição grava quem, quando, motivo e versão do edital |
| **2** | Fluxo de inscrição ponta a ponta: cadastro por CPF, cargo, cota, upload, isenção por regra | Um candidato sintético vai de RASCUNHO a HOMOLOGADA por API, localmente |
| **3** | Pagamento em sandbox com **chave de idempotência** e conciliação | Teste dispara o mesmo pagamento N vezes em paralelo e gera uma única cobrança |
| **4** | Geração assíncrona de boleto e cartão (SQS + worker) e **publicação estática** do resultado | Nenhum PDF é gerado dentro do request; resultado consultável sem tocar no banco |
| **5** | **Fila de espera** com admissão controlada e modo degradado níveis 1, 4 e 5 | Demo: acima da capacidade, o candidato vê posição e tempo, não erro |
| **6** | Terraform do ambiente inteiro e **gerador k6 com perfil de último dia** (§3 acima) | `terraform apply` sobe tudo e `destroy` devolve tudo; k6 roda contra o ambiente |
| **7** | Ensaios a 4, 8 e 16,7 mil/h, depois **subir até quebrar**; custo por patamar | Curva de quebra com gargalo identificado |
| **8** | Correção do gargalo, repetição do ensaio, **relatório final** | Relatório com antes e depois, métricas da §2 e custo real |

### Estrutura sugerida

```
fiap-startup-one/poc/
  README.md            como rodar local e na AWS
  app/                 monólito modular (inscricao, isencao, pagamento, resultado, fila)
  worker/              consumidor SQS: boleto, cartão, publicação estática
  infra/               Terraform
  carga/               scripts k6 e perfis de carga
  relatorio/           resultados brutos e o relatório final
  docker-compose.yml   Postgres e fila locais para desenvolvimento
```

O código da POC entra neste mesmo repositório, em `fiap-startup-one/poc/`.

---

## 6. Primeira sessão: por onde começar

1. Ler [`CONTINUAR-AQUI.md`](../CONTINUAR-AQUI.md) §6 (decisões fechadas) e §11 (como o
   usuário trabalha).
2. Ler `fiap-startup-one/04-escopo-poc.md` inteiro e
   `fiap-startup-one/02-produto-features-e-fluxos.md` §3, §4, §5 e §8.
3. Perguntar ao usuário, em lista curta e numerada:
   1. Qual é o **prazo real de entrega** do Startup One? (O plano de 8 semanas foi escrito em
      setembro; se o prazo for menor, o corte segue a regra: cai o painel, nunca o ensaio.)
   2. Linguagem: Python/FastAPI (recomendado) ou outra?
   3. A conta AWS citada como "já existe" está acessível e com crédito? Verificar com
      `aws sts get-caller-identity` antes de contar com ela.
   4. Em que máquina a POC vai ser feita? As armadilhas da §7 valem para a máquina original;
      em outra, conferir Python, Docker, k6, Terraform e AWS CLI antes.
4. Com as respostas, executar a semana 1.

---

## 7. Armadilhas desta máquina e deste projeto

- **PowerShell, não bash.** Scripts `.ps1` com acento em **UTF-8 com BOM**.
- **Não usar `2>&1` em executável nativo** no PowerShell 5.1.
- **Não usar `Start-Process -Wait` com Chrome headless** (ver CONTINUAR-AQUI §8).
- Docker e k6 **não foram verificados** nesta máquina. Conferir na primeira sessão.
- **Ensaio de carga só com escala devolvida no fim.** Todo ensaio termina com `terraform destroy`
  ou escala a zero, conferido no console. O orçamento é ~US$ 100.
- **Sem travessões** em documento, slide ou conversa.
- **Toda conta do relatório conferida por script**, não a olho. O usuário cobra.
- **Não alterar os números canônicos** do plano de negócios a partir da POC sem perguntar. Se o
  custo medido divergir do previsto, isso é achado do relatório, não correção silenciosa.

---

## 8. O que a POC devolve para os outros entregáveis

- **Pitch:** um slide novo ou atualizado com o resultado do ensaio (passou, ou quebrou em X e
  por quê). Fonte em `fiap-startup-one/apresentacao/pitch.html`, regerar com `build-pdf.ps1`.
- **Plano de negócios:** o custo por hora de pico medido alimenta o preço por inscrição, que é a
  variável não verificada que decide o negócio (piso de R$ 2,29).
- **Multicloud:** o gatilho 3 (restauração em até 8 h) continua estimativa; a POC **não** testa
  a restauração no Google Cloud, e o relatório não deve sugerir que testou.
