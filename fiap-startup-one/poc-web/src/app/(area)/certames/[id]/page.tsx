"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { Alert, App, Button, Card, Col, Input, Row, Skeleton, Table, Typography, Upload } from "antd";
import { ArrowRightOutlined, HeartOutlined, IdcardOutlined, UploadOutlined } from "@ant-design/icons";
import { useAjuda } from "@/componentes/Moldura";
import { Escolha } from "@/componentes/Escolha";
import { Selo } from "@/componentes/Selo";
import {
  aberta,
  api,
  dataHora,
  enviarDocumento,
  ErroApi,
  janela,
  reais,
  useApi,
  useCertame,
  type Certame,
  type Isencao,
  type Resumo,
} from "@/lib/api";
import { fases } from "@/lib/rotulos";

const motivos: Record<string, string> = {
  NIS_DE_OUTRO_CANDIDATO: "este NIS já foi usado por outro CPF neste certame",
};

export default function PaginaCertame() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { message } = App.useApp();
  const certame = useCertame(id);
  const minhas = useApi<Resumo[]>("/inscricoes");
  const isencao = useApi<Isencao | null>(`/certames/${id}/isencao`);
  const [cargo, setCargo] = useState<number | null>(null);
  const [abrindo, setAbrindo] = useState(false);
  useAjuda(
    "Página do certame",
    <>
      <p>Escolha o cargo e toque em &quot;Começar inscrição&quot;. Você pode trocar o cargo depois, até enviar.</p>
      <p>Se você tem direito à isenção da taxa, peça antes de se inscrever, dentro do prazo do pedido.</p>
    </>,
  );

  if (certame.erro) return <Alert type="error" showIcon title={certame.erro.status === 404 ? "Certame não encontrado." : certame.erro.message} />;
  const c = certame.dados;
  if (!c) return <Card><Skeleton active /></Card>;

  const existente = minhas.dados?.find((i) => i.certame_id === c.id);
  const inscricaoAberta = aberta(c, "inscricao");
  const fimInscricao = janela(c, "inscricao");

  const comecar = async () => {
    if (cargo === null) return;
    setAbrindo(true);
    try {
      const r = await api<{ id: number }>(`/certames/${c.id}/inscricoes`, { json: { cargo_id: cargo } });
      router.push(`/inscricoes/${r.id}`);
    } catch (e) {
      message.error(e instanceof ErroApi && e.status === 409 ? "O prazo de inscrição terminou." : "Não foi possível começar a inscrição.");
      setAbrindo(false);
    }
  };

  return (
    <>
      <Typography.Title level={2} style={{ color: "var(--brand-800)", margin: "8px 4px 2px" }}>{c.nome}</Typography.Title>
      <Typography.Text type="secondary" style={{ margin: "0 4px 16px", display: "block" }}>
        {c.banca}
        {c.edital_versao && ` · edital versão ${c.edital_versao}`}
      </Typography.Text>

      <Row gutter={[16, 16]}>
        <Col xs={24} xl={15}>
          <Card title="Inscrição">
            {existente ? (
              <div style={{ display: "flex", gap: 16, alignItems: "center", justifyContent: "space-between", flexWrap: "wrap" }}>
                <span>Você já tem uma inscrição neste certame.</span>
                <Link href={`/inscricoes/${existente.id}`}>
                  <Button type="primary" icon={<ArrowRightOutlined />} iconPlacement="end">Ver minha inscrição</Button>
                </Link>
              </div>
            ) : !inscricaoAberta ? (
              <Alert type="info" showIcon title="As inscrições não estão abertas agora." />
            ) : (
              <>
                <Typography.Paragraph>
                  Inscrições abertas até <strong>{fimInscricao && dataHora(fimInscricao.fecha_em)}</strong>. Escolha o cargo:
                </Typography.Paragraph>
                <Escolha
                  rotulo="Cargo"
                  colunas={2}
                  valor={cargo}
                  aoMudar={setCargo}
                  opcoes={c.cargos.map((g) => ({
                    valor: g.id,
                    titulo: g.nome,
                    detalhe: `${g.vagas} vagas · taxa ${reais(g.taxa_centavos)}`,
                  }))}
                />
                <div style={{ marginTop: 16, display: "flex", justifyContent: "flex-end" }}>
                  <Button type="primary" size="large" disabled={cargo === null} loading={abrindo} onClick={comecar}
                    icon={<ArrowRightOutlined />} iconPlacement="end">
                    Começar inscrição
                  </Button>
                </div>
              </>
            )}
          </Card>

          {isencao.erro?.degradado ? (
            <Card title="Isenção da taxa" style={{ marginTop: 16 }}>
              <Alert type="info" showIcon title={isencao.erro.message}
                description="Um pedido de isenção que você já fez continua valendo. Esta parte volta sozinha." />
            </Card>
          ) : (
            <PedidoDeIsencao certame={c} isencao={isencao.dados ?? null} aoMudar={isencao.recarregar} />
          )}
        </Col>

        <Col xs={24} xl={9}>
          <Card title="Cronograma">
            <Table
              size="small"
              pagination={false}
              rowKey="fase"
              dataSource={c.cronograma}
              columns={[
                { title: "Etapa", dataIndex: "fase", render: (f: string) => fases[f] ?? f },
                { title: "Abre", dataIndex: "abre_em", render: dataHora },
                { title: "Fecha", dataIndex: "fecha_em", render: dataHora },
              ]}
            />
          </Card>
          <Card title="Cargos e vagas" style={{ marginTop: 16 }}>
            <Table
              size="small"
              pagination={false}
              rowKey="id"
              dataSource={c.cargos}
              columns={[
                { title: "Cargo", dataIndex: "nome" },
                { title: "Vagas", dataIndex: "vagas", align: "right" },
                { title: "Taxa", dataIndex: "taxa_centavos", align: "right", render: reais },
              ]}
            />
            {c.cotas.length > 0 && (
              <Typography.Paragraph type="secondary" style={{ marginTop: 12, marginBottom: 0 }}>
                Reserva de vagas: {c.cotas.map((q) => q.nome).join("; ")}.
              </Typography.Paragraph>
            )}
          </Card>
        </Col>
      </Row>
    </>
  );
}

function PedidoDeIsencao({ certame, isencao, aoMudar }: { certame: Certame; isencao: Isencao | null; aoMudar: () => void }) {
  const { message } = App.useApp();
  const [hipotese, setHipotese] = useState<"cadunico" | "doador_medula" | null>(null);
  const [nis, setNis] = useState("");
  const [comprovante, setComprovante] = useState<number | null>(null);
  const [enviando, setEnviando] = useState(false);
  const prazo = janela(certame, "isencao");

  if (!isencao && !aberta(certame, "isencao")) return null;

  const pedir = async () => {
    setEnviando(true);
    try {
      await api(`/certames/${certame.id}/isencao`, {
        json: hipotese === "cadunico" ? { hipotese, nis: nis.replace(/\D/g, "") } : { hipotese, documento_id: comprovante },
      });
      message.success("Pedido de isenção registrado.");
      aoMudar();
    } catch (e) {
      message.error(e instanceof ErroApi && e.status === 422
        ? hipotese === "cadunico" ? "O NIS não é válido. Confira os 11 dígitos." : "Envie o comprovante antes de pedir."
        : "Não foi possível registrar o pedido.");
    } finally {
      setEnviando(false);
    }
  };

  return (
    <Card title="Isenção da taxa" style={{ marginTop: 16 }}>
      {isencao ? (
        <>
          <Selo tom={isencao.estado === "DEFERIDA" ? "ok" : isencao.estado === "INDEFERIDA" ? "erro" : "aviso"}>
            {isencao.estado === "DEFERIDA" ? "Deferida — você não paga a taxa" : isencao.estado === "INDEFERIDA" ? "Indeferida" : "Em análise pela banca"}
          </Selo>
          {isencao.motivo_codigo && (
            <Typography.Paragraph type="secondary" style={{ marginTop: 8 }}>
              Motivo: {motivos[isencao.motivo_codigo] ?? isencao.motivo_codigo}.
            </Typography.Paragraph>
          )}
          <Typography.Paragraph type="secondary" style={{ marginTop: 8, marginBottom: 0 }}>
            Pedido feito em {dataHora(isencao.solicitada_em)}.
          </Typography.Paragraph>
        </>
      ) : (
        <>
          <Typography.Paragraph>
            Pedidos até <strong>{prazo && dataHora(prazo.fecha_em)}</strong>, antes da inscrição. Em qual situação você se encaixa?
          </Typography.Paragraph>
          <Escolha
            rotulo="Hipótese de isenção"
            colunas={2}
            valor={hipotese}
            aoMudar={setHipotese}
            opcoes={[
              { valor: "cadunico", titulo: "Inscrito no CadÚnico", detalhe: "Informe o seu NIS", icone: <IdcardOutlined /> },
              { valor: "doador_medula", titulo: "Doador de medula óssea", detalhe: "Envie o comprovante", icone: <HeartOutlined /> },
            ]}
          />
          {hipotese === "cadunico" && (
            <div style={{ marginTop: 16, maxWidth: 320 }}>
              <label htmlFor="nis" style={{ fontWeight: 700, display: "block", marginBottom: 6 }}>NIS (11 dígitos)</label>
              <Input id="nis" inputMode="numeric" value={nis} onChange={(e) => setNis(e.target.value)} maxLength={14} />
            </div>
          )}
          {hipotese === "doador_medula" && (
            <div style={{ marginTop: 16 }}>
              <Upload
                accept=".pdf,.jpg,.jpeg,.png"
                maxCount={1}
                customRequest={async ({ file, onSuccess, onError }) => {
                  try {
                    const r = await enviarDocumento("comprovante_isencao", file as File);
                    setComprovante(r.id);
                    onSuccess?.(r);
                  } catch (e) {
                    message.error("Arquivo recusado: envie PDF, JPG ou PNG de até 5 MB.");
                    onError?.(e as Error);
                  }
                }}
              >
                <Button icon={<UploadOutlined />}>Enviar comprovante</Button>
              </Upload>
            </div>
          )}
          <div style={{ marginTop: 16, display: "flex", justifyContent: "flex-end" }}>
            <Button
              onClick={pedir}
              loading={enviando}
              disabled={!hipotese || (hipotese === "cadunico" ? nis.replace(/\D/g, "").length !== 11 : comprovante === null)}
            >
              Pedir isenção
            </Button>
          </div>
        </>
      )}
    </Card>
  );
}
