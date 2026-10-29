# FIAP Startup One: Certamis

**Certamis** é a plataforma que conduz um processo seletivo público de ponta a ponta, do edital à
homologação do resultado. Vendida para **bancas organizadoras**.

> *O nome vem de "certame", como a administração pública chama o processo seletivo público. É o
> objeto inteiro do produto.*

---

## Para apresentar

| Arquivo | O que é |
|---|---|
| **[`apresentacao/Certamis-pitch.pdf`](apresentacao/Certamis-pitch.pdf)** | **O pitch para investidores**, 10 slides, 16:9 (960×540 pt) |
| [`apresentacao/previews/`](apresentacao/previews/) | Os 10 slides em PNG, para revisar sem abrir o PDF |

**A tese em uma frase:** até 2028 as regras do concurso mudam para quase todo o setor público, e
a prova on-line já começou a ser aplicada. Quem for trocar de sistema vai escolher pelo que
quase ninguém resolve, que é **aguentar o último dia.**

---

## Documentos

| Arquivo | Conteúdo |
|---|---|
| [01-ideacao.md](01-ideacao.md) | A rodada de ideação que selecionou esta ideia, com os **seis motivos legítimos** para uma nuvem ser necessária, e as nove ideias verificadas contra eles |
| [02-produto-features-e-fluxos.md](02-produto-features-e-fluxos.md) | **O produto:** 9 módulos, fluxogramas em ASCII, máquina de estados da inscrição, recurso em escala sem IA, e as três decisões fechadas |
| [03-plano-de-negocios.md](03-plano-de-negocios.md) | **O plano:** mercado, preço com sensibilidade, unit economics, projeções de 5 anos, concorrência verificada, riscos |
| [04-escopo-poc.md](04-escopo-poc.md) | **A POC:** 1 pessoa, 8 semanas, ~US$ 100. O entregável é um relatório de ensaio de carga, não uma tela |

---

## O essencial

**O problema** é uma manchete anual: *"site do concurso fora do ar no último dia de inscrição"*.
Em 2026 foram pelo menos três casos: a Alece teve falha no sistema da banca na isenção e no
recurso, a prova on-line da AgSUS foi anulada e reaplicada, e a FGV ficou dias fora do ar após um
ataque. Nenhum dos três foi queda do provedor de nuvem.

**Por que exige nuvem:** um certame tem **cinco picos**, não um. E os três maiores, que são cartão
de confirmação, gabarito e resultado, são acessados pela **base inteira ao mesmo tempo**. A carga
é 40× a média **daquele certame**, cerca de 16,7 mil inscrições por hora num certame de 300 mil
candidatos, e a data está publicada em edital com meses de antecedência.

**Por que agora:** a **Lei nº 14.965/2024** estabelece normas gerais para os concursos de todos os
entes a partir de **1º/01/2028** (estados e municípios podem, em alternativa, editar normas
próprias) e autoriza a prova a distância, que já está sendo praticada em 2026. A lei não obriga
ninguém a trocar de sistema, mas muda o que o sistema precisa fazer.

**Como ganha dinheiro:** assinatura anual pequena mais **valor por inscrição processada**. Alinha
com o modelo da própria banca e concentra a receita nos mesmos dias em que o custo se concentra.
**Vender para a banca evita licitação:** o órgão licita a banca; a banca contrata fornecedor como
empresa privada.

**A variável que decide tudo:** o preço por inscrição. **Abaixo de R$ 2,29 o modelo não fecha.**
Está marcada como não verificada no plano, de propósito.

**O que não escondemos:** o TAM da categoria é modesto (~R$ 90 mi), e **o mercado já está
ocupado**, com Gestor Editais, ProSeleta, Fábrica de Provas e ADSR. O argumento não é "ninguém faz
isso". É *"todos fazem a inscrição, quase ninguém sobrevive ao dia do pico e ao recurso em
massa"*.

---

## O trabalho de multicloud

A defesa de arquitetura está em [`../fiap-multicloud/`](../fiap-multicloud/) e conclui **uma
nuvem para operar, duas para sobreviver**. Operar em duas não resolve 7 das 8 falhas que derrubam
este produto, que são dívidas nossas, e a segunda nuvem sempre ligada custaria R$ 1,22 milhão por
hora de indisponibilidade evitada. Para a queda do provedor, basta uma segunda nuvem desligada no
Google Cloud, em São Paulo, por R$ 80 mil por ano.
