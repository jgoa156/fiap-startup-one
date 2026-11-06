"use client";

import Link from "next/link";
import { Card, Col, Row, Tag } from "antd";
import { ApiOutlined, AppstoreOutlined, CloudServerOutlined, DashboardOutlined, LineChartOutlined } from "@ant-design/icons";

// A entrada da documentação. "Viva" quer dizer: cada página nasce do que roda, e um teste quebra a
// suíte quando ela fica para trás. A coluna "o que a mantém viva" diz qual.
const paginas = [
  {
    href: "/docs/api", icone: <ApiOutlined />, titulo: "API (backend)",
    texto: "Todas as rotas de /api, com corpo, respostas e erros, e o botão para chamar cada uma daqui mesmo. Desenhada pelo Scalar a partir do openapi.yaml que o servidor serve.",
    viva: "TestOpenAPICobreTodasAsRotas: rota sem documentação, ou documentação de rota que não existe, quebra a suíte.",
  },
  {
    href: "/design", icone: <AppstoreOutlined />, titulo: "Componentes (front)", estatico: true,
    texto: "O sistema de design no Storybook: cores, tipografia, raios e sombras, os componentes do Ant Design como o tema os veste, e cada componente do portal em todos os estados, com controles para mexer.",
    viva: "As histórias importam os componentes reais; o build do Storybook quebra se um deles mudar de assinatura.",
  },
  {
    href: "/docs/infra", icone: <CloudServerOutlined />, titulo: "Infraestrutura",
    texto: "O caminho de um pedido, cada peça nesta máquina e na AWS, e o Terraform lido do repositório: recursos, quem usa quem, variáveis e saídas.",
    viva: "TestInfraDocAtual: o JSON da página é regerado do Terraform; se ficou para trás de um .tf, a suíte quebra.",
  },
  {
    href: "/operacao", icone: <DashboardOutlined />, titulo: "Operação e ensaios",
    texto: "Dispara os ensaios de carga (último dia, estresse até quebrar, pico do resultado), muda o nível de degradação e mostra o sistema em tempo real.",
    viva: "Os testes do executor (internal/ensaio) e a auditoria de cada patamar.",
  },
  {
    href: "http://127.0.0.1:3000/d/lisura/lisura", icone: <LineChartOutlined />, titulo: "Grafana", estatico: true,
    texto: "O painel completo: instâncias, tráfego por rota, fila, banco, processo e o k6, anotado com cada ensaio e cada mudança de nível.",
    viva: "O painel é um arquivo do repositório (observabilidade/grafana/paineis), provisionado e só leitura.",
  },
];

export function Entrada() {
  return (
    <>
      <h1 style={{ marginTop: 0, fontSize: 30, fontWeight: 800, color: "var(--brand-800)" }}>Documentação viva da Lisura</h1>
      <p style={{ color: "var(--ink-600)", maxWidth: 820, fontSize: 15 }}>
        Tudo aqui sai do que está rodando ou do código, não de um texto escrito à parte. Quando o código muda e a documentação
        não acompanha, um teste falha.
      </p>
      <Row gutter={[16, 16]}>
        {paginas.map((p) => {
          const corpo = (
            <Card hoverable style={{ height: "100%" }}>
              <div style={{ fontSize: 26, color: "var(--brand-700)" }}>{p.icone}</div>
              <h2 style={{ fontSize: 18, fontWeight: 800, margin: "8px 0", color: "var(--brand-800)" }}>{p.titulo}</h2>
              <p style={{ color: "var(--ink-900)" }}>{p.texto}</p>
              <Tag color="success" style={{ whiteSpace: "normal", lineHeight: 1.5, padding: "4px 8px" }}>O que a mantém viva: {p.viva}</Tag>
            </Card>
          );
          return (
            <Col key={p.href} xs={24} md={12} xl={8}>
              {p.estatico ? <a href={p.href} style={{ display: "block", height: "100%" }}>{corpo}</a> : <Link href={p.href} style={{ display: "block", height: "100%" }}>{corpo}</Link>}
            </Col>
          );
        })}
      </Row>
    </>
  );
}
