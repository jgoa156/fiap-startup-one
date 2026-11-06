"use client";

import { Suspense } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Alert, Card, Col, Empty, Row, Skeleton, Typography } from "antd";
import { useAjuda } from "@/componentes/Moldura";
import { Selo } from "@/componentes/Selo";
import { dataHora, jaAbriu, useApi, type CertameItem } from "@/lib/api";

function Lista() {
  const q = (useSearchParams().get("q") ?? "").trim().toLowerCase();
  const { dados, erro, carregando } = useApi<CertameItem[]>("/certames");
  useAjuda("Certames abertos", <p>Todos os certames com inscrição aberta ou por abrir. Toque num deles para ver cargos, prazos e se inscrever.</p>);

  const lista = (dados ?? []).filter((c) => !q || `${c.nome} ${c.banca}`.toLowerCase().includes(q));
  return (
    <>
      <Typography.Title level={2} style={{ color: "var(--brand-800)", margin: "8px 4px 16px" }}>
        Certames abertos
      </Typography.Title>
      {q && (
        <Typography.Paragraph type="secondary" style={{ margin: "-8px 4px 16px" }}>
          Resultado para “{q}” · <Link href="/certames">limpar busca</Link>
        </Typography.Paragraph>
      )}
      {erro?.degradado ? (
        <Alert type="info" showIcon title={erro.message}
          description={<>As inscrições que você já começou continuam no <Link href="/">Início</Link>, e quem tem o link de um certame entra direto nele.</>} />
      ) : erro ? (
        <Alert type="error" showIcon title={erro.message} />
      ) : carregando && !dados ? (
        <Card>
          <Skeleton active />
        </Card>
      ) : lista.length === 0 ? (
        <Card>
          <Empty description={q ? "Nenhum certame com esse nome." : "Nenhum certame com inscrição aberta agora."} />
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {lista.map((c) => {
            const abriu = jaAbriu(c.inscricao_abre_em);
            return (
              <Col xs={24} md={12} key={c.id}>
                <Link href={`/certames/${c.id}`} style={{ display: "block" }}>
                  <Card hoverable>
                    <div style={{ fontWeight: 800, fontSize: 17, color: "var(--brand-800)" }}>{c.nome}</div>
                    <div style={{ color: "var(--ink-600)", marginBottom: 12 }}>{c.banca}</div>
                    {abriu ? (
                      <Selo tom="ok">Inscrições abertas até {dataHora(c.inscricao_fecha_em)}</Selo>
                    ) : (
                      <Selo tom="neutro">Inscrições abrem em {dataHora(c.inscricao_abre_em)}</Selo>
                    )}
                  </Card>
                </Link>
              </Col>
            );
          })}
        </Row>
      )}
    </>
  );
}

export default function Certames() {
  return (
    <Suspense>
      <Lista />
    </Suspense>
  );
}
