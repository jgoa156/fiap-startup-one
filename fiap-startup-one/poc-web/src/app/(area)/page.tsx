"use client";

import Link from "next/link";
import { Alert, Button, Card, Col, Empty, Row, Skeleton, Typography } from "antd";
import { ArrowRightOutlined, ClockCircleOutlined, EditOutlined, FileSearchOutlined } from "@ant-design/icons";
import { useAjuda, useSessao } from "@/componentes/Moldura";
import { Selo } from "@/componentes/Selo";
import { data, dataHora, useApi, type CertameItem, type Resumo } from "@/lib/api";
import { estados } from "@/lib/rotulos";

// Área de Trabalho do Inclusa (mockups/sistema-direcao-ab/02-dashboard) para o candidato:
// saudação, ações de hoje, minhas inscrições e certames abertos.

function saudacao() {
  const h = new Date().getHours();
  return h < 12 ? "Bom dia" : h < 18 ? "Boa tarde" : "Boa noite";
}

export default function Inicio() {
  const { eu } = useSessao();
  const inscricoes = useApi<Resumo[]>("/inscricoes");
  const certames = useApi<CertameItem[]>("/certames");
  useAjuda(
    "Início",
    <>
      <p>Aqui ficam o que você precisa fazer agora e todas as suas inscrições.</p>
      <p>Cada cartão de &quot;Ações de hoje&quot; leva direto para onde a ação acontece.</p>
    </>,
  );

  const lista = inscricoes.dados ?? [];
  const acoes = lista.filter((i) => i.estado === "AGUARDANDO_PAGAMENTO" || i.estado === "RASCUNHO");

  return (
    <>
      <Typography.Title level={2} style={{ color: "var(--brand-800)", margin: "8px 4px 2px" }}>
        {saudacao()}, {eu.dados.nome.split(" ")[0]}
      </Typography.Title>
      <Typography.Text type="secondary" style={{ margin: "0 4px", display: "block" }}>
        {new Date().toLocaleDateString("pt-BR", { weekday: "long", day: "numeric", month: "long", year: "numeric" })}
        {lista.length > 0 && ` · ${lista.length} inscriç${lista.length === 1 ? "ão" : "ões"}`}
      </Typography.Text>

      {inscricoes.erro && <Alert type="error" showIcon title={inscricoes.erro.message} style={{ marginTop: 16 }} />}

      <div className="rotulo-secao">Ações de hoje</div>
      {inscricoes.carregando && !inscricoes.dados ? (
        <Card>
          <Skeleton active paragraph={{ rows: 1 }} />
        </Card>
      ) : acoes.length === 0 ? (
        <Card>
          <div style={{ display: "flex", gap: 16, alignItems: "center", justifyContent: "space-between", flexWrap: "wrap" }}>
            <div>
              <div style={{ fontWeight: 800, fontSize: 17 }}>Nada pendente por aqui.</div>
              <Typography.Text type="secondary">Quando houver prazo correndo, ele aparece neste lugar.</Typography.Text>
            </div>
            <Link href="/certames">
              <Button type="primary" icon={<FileSearchOutlined />}>
                Ver certames abertos
              </Button>
            </Link>
          </div>
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {acoes.map((i) => {
            const pagar = i.estado === "AGUARDANDO_PAGAMENTO";
            return (
              <Col xs={24} lg={12} key={i.id}>
                <Card className={pagar ? "acao-urgente" : undefined}>
                  <div style={{ display: "flex", gap: 16, alignItems: "center", justifyContent: "space-between", flexWrap: "wrap" }}>
                    <div style={{ minWidth: 0 }}>
                      <div style={{ fontWeight: 800, fontSize: 17, color: pagar ? "var(--marker-800)" : undefined }}>
                        {pagar ? <ClockCircleOutlined /> : <EditOutlined />} {pagar ? "Pague a taxa" : "Termine sua inscrição"}
                      </div>
                      <Typography.Text type="secondary">
                        {i.certame_nome} · {i.cargo_nome}
                      </Typography.Text>
                    </div>
                    <Link href={`/inscricoes/${i.id}`}>
                      <Button type={pagar ? "primary" : "default"} icon={<ArrowRightOutlined />} iconPlacement="end">
                        {pagar ? "Pagar" : "Continuar"}
                      </Button>
                    </Link>
                  </div>
                </Card>
              </Col>
            );
          })}
        </Row>
      )}

      <div className="rotulo-secao">Minhas inscrições</div>
      {lista.length === 0 && !inscricoes.carregando ? (
        <Card>
          <Empty description="Você ainda não se inscreveu em nenhum certame.">
            <Link href="/certames">
              <Button type="primary">Escolher um certame</Button>
            </Link>
          </Empty>
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {lista.map((i) => (
            <Col xs={24} md={12} xl={8} key={i.id}>
              <Link href={`/inscricoes/${i.id}`} style={{ display: "block" }}>
                <Card hoverable>
                  <div style={{ fontWeight: 800, fontSize: 16, color: "var(--brand-800)" }}>{i.certame_nome}</div>
                  <div style={{ color: "var(--ink-600)", marginBottom: 12 }}>{i.cargo_nome}</div>
                  <Selo tom={estados[i.estado].tom}>{estados[i.estado].rotulo}</Selo>
                  <div style={{ color: "var(--ink-600)", fontSize: 12, marginTop: 10 }}>Atualizada em {dataHora(i.atualizada_em)}</div>
                </Card>
              </Link>
            </Col>
          ))}
        </Row>
      )}

      <div className="rotulo-secao">Certames com inscrição aberta</div>
      {certames.erro?.degradado && (
        <Alert type="info" showIcon title={certames.erro.message} description="Suas inscrições, acima, continuam funcionando normalmente." />
      )}
      <Row gutter={[16, 16]}>
        {(certames.dados ?? []).slice(0, 3).map((c) => (
          <Col xs={24} md={12} xl={8} key={c.id}>
            <Link href={`/certames/${c.id}`} style={{ display: "block" }}>
              <Card hoverable>
                <div style={{ fontWeight: 800, color: "var(--brand-800)" }}>{c.nome}</div>
                <div style={{ color: "var(--ink-600)" }}>{c.banca}</div>
                <div style={{ marginTop: 8 }}>Inscrições até <strong>{data(c.inscricao_fecha_em)}</strong></div>
              </Card>
            </Link>
          </Col>
        ))}
      </Row>
    </>
  );
}
