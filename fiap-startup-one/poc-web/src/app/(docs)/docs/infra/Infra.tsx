"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Alert, Card, Col, Collapse, Row, Table, Tag, Typography } from "antd";

// A infraestrutura. A parte escrita (o caminho de um pedido) muda pouco; o resto vem de
// /docs/infra.json, que o cmd/infradoc gera do Terraform (TestInfraDocAtual quebra a suíte se o
// JSON ficar para trás de um .tf).

type Bloco = { tipo: string; nome: string; arquivo: string; descricao?: string; referencias?: string[]; padrao?: string; condicional?: boolean };
type Doc = { arquivos: { nome: string; resumo: string }[]; recursos: Bloco[]; dados: Bloco[]; variaveis: Bloco[]; saidas: Bloco[] };

const camadas = [
  {
    nome: "Borda", local: "Next na 8090 repassa /api e /publico", aws: "CloudFront: /publico do S3 (cache), /api e o front pelo balanceador",
    porque: "O pico de leitura (cartão, relação, resultado) sai do banco e vira arquivo estático no CDN.",
  },
  { nome: "Front", local: "Next de produção (standalone) na 8090", aws: "ECS Fargate, tarefas \"web\" atrás do ALB", porque: "O portal do candidato." },
  {
    nome: "API", local: "lisura.exe na 8082, com a fila (4 trabalhadores) no mesmo processo", aws: "ECS Fargate, tarefas \"api\" atrás do ALB",
    porque: "Inscrição, pagamento, avisos; a fila no Postgres faz o papel do SQS + Lambda da arquitetura alvo.",
  },
  { nome: "Banco", local: "PostgreSQL 17 local", aws: "Aurora PostgreSQL Serverless v2 (ACU mínimo e máximo por variável)", porque: "Uma fonte da verdade; a trilha de auditoria é imutável por gatilho." },
  { nome: "Arquivos", local: "pastas em C:\\Users\\rockb\\tools (publico, arquivos, cofre)", aws: "S3: publicação e documentos; um bucket à parte com bloqueio de objeto para as âncoras", porque: "O cofre não deixa reescrever o que foi ancorado." },
  { nome: "Pagamento", local: "PSP simulado na 8081 (diário em arquivo)", aws: "PSP simulado como tarefa do ECS (Cloud Map: psp.lisura.local)", porque: "Repete e perde avisos de propósito, para provar a idempotência." },
  { nome: "E-mail", local: "caixa de saída em arquivo, ou descarte no ensaio", aws: "SES com conjunto de configuração; devoluções e reclamações pelo SNS", porque: "Um aviso por evento; o endereço que devolve é suprimido." },
  {
    nome: "Observação", local: "Prometheus (9090) raspa /metrics; Grafana (3000) com painéis do repositório; o k6 escreve no Prometheus",
    porque: "Ver o sistema mudando enquanto o ensaio roda.",
    aws: "As mesmas métricas em /metrics dentro da VPC; CloudWatch para ECS (tarefas no ar, CPU) e Aurora (ACU). Prometheus e Grafana gerenciados ainda não estão no Terraform.",
  },
];

const fluxo = `flowchart LR
  cand([Candidato]) --> cdn[CloudFront]
  cdn -->|/publico| s3[(S3 publicação)]
  cdn -->|/ e /api| alb[Balanceador]
  alb --> web[Front Next]
  alb --> api[API Go]
  api --> pg[(Aurora Postgres)]
  api -->|fila no Postgres| pg
  api -->|cobrança, devolução| psp[PSP]
  psp -->|aviso assinado| api
  api -->|aviso por e-mail| ses[SES]
  ses -->|devolução, reclamação| sns[SNS] -->|assinado| api
  api -->|âncora da trilha| cofre[(S3 cofre, bloqueio de objeto)]
  api -->|cartão, relação, resultado| s3
  k6[k6] -.->|carga| cdn
  prom[Prometheus] -.->|raspa /metrics| api
  graf[Grafana] -.-> prom`;

function id(s: string) {
  return s.replace(/[^a-zA-Z0-9]/g, "_");
}

function diagramaTerraform(d: Doc) {
  const recursos = new Set(d.recursos.map((r) => `${r.tipo}.${r.nome}`));
  const linhas = ["flowchart LR"];
  for (const a of d.arquivos) {
    const doArquivo = d.recursos.filter((r) => r.arquivo === a.nome);
    if (!doArquivo.length) continue;
    linhas.push(`  subgraph ${id(a.nome)}["${a.nome}"]`);
    for (const r of doArquivo) linhas.push(`    ${id(r.tipo + "." + r.nome)}["${r.tipo.replace(/^aws_/, "")}<br/><b>${r.nome}</b>"]`);
    linhas.push("  end");
  }
  for (const r of d.recursos) {
    for (const ref of r.referencias ?? []) {
      if (recursos.has(ref)) linhas.push(`  ${id(r.tipo + "." + r.nome)} --> ${id(ref)}`);
    }
  }
  return linhas.join("\n");
}

function Mermaid({ codigo, nome }: { codigo: string; nome: string }) {
  const ref = useRef<HTMLDivElement>(null);
  const [erro, setErro] = useState<string>();
  useEffect(() => {
    let vivo = true;
    import("mermaid").then(async ({ default: m }) => {
      m.initialize({ startOnLoad: false, securityLevel: "strict", theme: "base", flowchart: { htmlLabels: true },
        themeVariables: { primaryColor: "#E9EDF2", primaryBorderColor: "#2F3B4C", primaryTextColor: "#14233A", lineColor: "#6B7A90", fontFamily: "Nunito, system-ui" } });
      try {
        const { svg } = await m.render(`m_${nome}`, codigo);
        if (vivo && ref.current) ref.current.innerHTML = svg;
      } catch (e) {
        if (vivo) setErro(String(e));
      }
    });
    return () => { vivo = false; };
  }, [codigo, nome]);
  return erro ? <Alert type="error" showIcon title="O diagrama não desenhou" description={erro} /> : <div ref={ref} style={{ overflowX: "auto" }} aria-label={`Diagrama ${nome}`} />;
}

export function Infra() {
  const [doc, setDoc] = useState<Doc>();
  const [erro, setErro] = useState<string>();
  useEffect(() => {
    fetch("/docs/infra.json", { cache: "no-store" }).then((r) => (r.ok ? r.json() : Promise.reject(r.status))).then(setDoc, () => setErro("Sem /docs/infra.json: rode go run ./cmd/infradoc na pasta poc."));
  }, []);
  const tf = useMemo(() => (doc ? diagramaTerraform(doc) : ""), [doc]);

  return (
    <>
      <Typography.Title level={2} style={{ marginTop: 0 }}>Infraestrutura</Typography.Title>
      <Typography.Paragraph type="secondary" style={{ maxWidth: 860 }}>
        A mesma aplicação roda nesta máquina e na AWS. Abaixo, o caminho de um pedido, o que cada peça é em cada lugar, e o
        Terraform lido do repositório: cada recurso com o comentário que o explica e quem ele usa. O Terraform está
        validado, mas <strong>ainda não foi aplicado</strong> (falta a credencial da AWS).
      </Typography.Paragraph>

      <Card title="O caminho de um pedido (na AWS)" style={{ marginBottom: 16 }}>
        <Mermaid codigo={fluxo} nome="fluxo" />
      </Card>

      <Card title="Cada peça, local e na AWS" style={{ marginBottom: 16 }}>
        <Table size="middle" pagination={false} rowKey="nome" dataSource={camadas} columns={[
          { title: "Peça", dataIndex: "nome", width: 120, render: (v) => <strong>{v}</strong> },
          { title: "Nesta máquina", dataIndex: "local" },
          { title: "Na AWS", dataIndex: "aws" },
          { title: "Por quê", dataIndex: "porque" },
        ]} />
      </Card>

      {erro && <Alert type="warning" showIcon title={erro} />}
      {doc && (
        <>
          <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
            {[["Arquivos", doc.arquivos.length], ["Recursos", doc.recursos.length], ["Variáveis", doc.variaveis.length], ["Saídas", doc.saidas.length]].map(([r, n]) => (
              <Col key={r} xs={12} md={6}><Card><Typography.Text type="secondary">{r}</Typography.Text><div style={{ fontSize: 28, fontWeight: 800 }}>{n}</div></Card></Col>
            ))}
          </Row>
          <Card title="Recursos do Terraform e quem usa quem" style={{ marginBottom: 16 }}>
            <Typography.Paragraph type="secondary">Gerado de infra/terraform/*.tf. Cada caixa é um recurso; a seta aponta para o recurso que ele referencia.</Typography.Paragraph>
            <Mermaid codigo={tf} nome="terraform" />
          </Card>
          <Card title="Por arquivo" style={{ marginBottom: 16 }}>
            <Collapse items={doc.arquivos.map((a) => ({
              key: a.nome,
              label: <><strong>{a.nome}</strong> <Typography.Text type="secondary">{a.resumo.slice(0, 140)}{a.resumo.length > 140 ? "…" : ""}</Typography.Text></>,
              children: (
                <>
                  {a.resumo && <Typography.Paragraph>{a.resumo}</Typography.Paragraph>}
                  <Table size="small" pagination={false} rowKey={(r) => r.tipo + r.nome}
                    dataSource={[...doc.recursos, ...doc.dados].filter((r) => r.arquivo === a.nome)} columns={[
                      { title: "Recurso", render: (_, r) => <><code>{r.tipo}</code> <strong>{r.nome}</strong>{r.condicional && <Tag style={{ marginLeft: 6 }}>condicional</Tag>}</> },
                      { title: "Para quê", dataIndex: "descricao" },
                      { title: "Usa", render: (_, r) => (r.referencias ?? []).filter((x) => !x.startsWith("var.")).map((x) => <Tag key={x}>{x}</Tag>) },
                    ]} />
                </>
              ),
            }))} />
          </Card>
          <Row gutter={[16, 16]}>
            <Col xs={24} lg={14}>
              <Card title="Variáveis (o que se ajusta sem mexer no código)">
                <Table size="small" pagination={false} rowKey="nome" dataSource={doc.variaveis} columns={[
                  { title: "Variável", dataIndex: "nome", render: (v) => <code>{v}</code> },
                  { title: "Padrão", dataIndex: "padrao", render: (v) => (v ? <code>{v}</code> : "—") },
                  { title: "Descrição", dataIndex: "descricao" },
                ]} />
              </Card>
            </Col>
            <Col xs={24} lg={10}>
              <Card title="Saídas">
                <Table size="small" pagination={false} rowKey="nome" dataSource={doc.saidas} columns={[
                  { title: "Saída", dataIndex: "nome", render: (v) => <code>{v}</code> },
                  { title: "Descrição", dataIndex: "descricao" },
                ]} />
              </Card>
            </Col>
          </Row>
        </>
      )}
    </>
  );
}
