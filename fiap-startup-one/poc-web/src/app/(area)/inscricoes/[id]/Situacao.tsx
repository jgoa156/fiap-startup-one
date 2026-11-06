"use client";

import { useEffect, useRef, useState } from "react";
import { Alert, App, Button, Card, Col, Descriptions, Row, Space, Spin, Timeline, Typography } from "antd";
import { BarcodeOutlined, CheckCircleOutlined, ExperimentOutlined, ExportOutlined, IdcardOutlined, QrcodeOutlined } from "@ant-design/icons";
import { api, dataHora, ErroApi, guardarCartao, janela, reais, useApi, type Certame, type Cobranca, type EventoTrilha, type Inscricao } from "@/lib/api";
import { eventos, necessidades } from "@/lib/rotulos";
import { Avisos } from "./Avisos";
import { Devolucoes } from "./Devolucoes";

// Inscrição já enviada: situação, pagamento (quando falta pagar) e o histórico, que é a trilha de
// auditoria do servidor contada em português.

export function Situacao({ inscricao: i, certame: c, aoMudar }: { inscricao: Inscricao; certame: Certame; aoMudar: () => void }) {
  const trilha = useApi<EventoTrilha[]>(`/inscricoes/${i.id}/trilha`);
  const cobrancas = useApi<Cobranca[]>(`/inscricoes/${i.id}/pagamento`);
  const pagas = (cobrancas.dados ?? []).filter((x) => x.estado === "PAGA");
  const valendo = ["PAGA", "ISENCAO_ATIVA", "EM_ANALISE", "DEFERIDA", "HOMOLOGADA"].includes(i.estado);

  // A cobrança nasce sem o Pix: ele chega pela fila em instantes. Enquanto faltar, pergunta de novo.
  const esperandoPix = (cobrancas.dados ?? []).some((x) => x.estado === "PENDENTE" && !x.pix_copia_cola && !x.linha_digitavel);
  const recarregarCobrancas = cobrancas.recarregar;
  useEffect(() => {
    if (!esperandoPix) return;
    const t = setInterval(recarregarCobrancas, 1500);
    return () => clearInterval(t);
  }, [esperandoPix, recarregarCobrancas]);

  return (
    <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
      <Col xs={24} xl={15}>
        {valendo && <Cartao inscricao={i.id} certame={c.nome} />}
        {i.estado === "AGUARDANDO_PAGAMENTO" && (
          <Pagamento inscricao={i} certame={c} cobrancas={cobrancas.dados ?? []} aoMudar={() => { cobrancas.recarregar(); aoMudar(); trilha.recarregar(); }} />
        )}
        {i.estado === "ISENCAO_ATIVA" && (
          <Alert type="success" showIcon style={{ marginBottom: 16 }} title="Inscrição enviada com isenção — você não paga a taxa." description="A banca vai analisar sua inscrição." />
        )}
        {pagas.map((p) => (
          <Alert key={p.id} type="success" showIcon style={{ marginBottom: 16 }}
            title={`Pagamento de ${reais(p.valor_centavos)} por ${p.metodo === "pix" ? "Pix" : "boleto"} confirmado${p.paga_em ? ` em ${dataHora(p.paga_em)}` : ""}.`} />
        ))}
        {i.estado === "CANCELADA" && (
          <Alert type="error" showIcon style={{ marginBottom: 16 }} title="Esta inscrição foi cancelada."
            description="Se você pagou a taxa, o valor volta: o andamento está em Devoluções." />
        )}
        <Devolucoes inscricao={i.id} />
        <Card title="Sua inscrição">
          <Descriptions column={1} size="middle" items={[
            { key: "c", label: "Cargo", children: c.cargos.find((g) => g.id === i.cargo_id)?.nome },
            { key: "d", label: "Cidade de prova", children: c.cidades_prova.find((d) => d.id === i.cidade_prova_id)?.nome ?? "—" },
            { key: "q", label: "Reserva de vagas", children: i.cotas.length ? i.cotas.map((q) => c.cotas.find((x) => x.codigo === q)?.nome).join("; ") : "Ampla concorrência" },
            { key: "a", label: "Atendimento especial", children: i.necessidades.length ? i.necessidades.map((x) => necessidades[x]).join("; ") : "Nenhum" },
            { key: "l", label: "Laudo compartilhado", children: i.laudo_documento_id ? "Sim, só com esta banca" : "Não" },
          ]} />
        </Card>
      </Col>
      <Col xs={24} xl={9}>
        <Card title="Histórico">
          {trilha.erro?.degradado && (
            <Alert type="info" showIcon style={{ marginBottom: 16 }} title="Histórico temporariamente indisponível."
              description="Para dar prioridade às inscrições e aos pagamentos. Nada foi perdido: ele volta sozinho." />
          )}
          <Timeline items={(trilha.dados ?? []).map((t) => ({ content: <><strong>{eventos[t.evento] ?? t.evento}</strong><div style={{ color: "var(--ink-600)", fontSize: 12 }}>{dataHora(t.em)}</div></> }))} />
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            O sistema só acrescenta registros a este histórico: nenhum registro é editado ou apagado.
          </Typography.Text>
        </Card>
        <Avisos inscricao={i.id} />
      </Col>
    </Row>
  );
}

// Cartão de confirmação: página estática publicada pela fila (CDN na produção). Enquanto não sai,
// a tela pergunta de novo.
function Cartao({ inscricao, certame }: { inscricao: number; certame: string }) {
  const cartao = useApi<{ disponivel: boolean; url: string }>(`/inscricoes/${inscricao}/cartao`);
  const pronto = cartao.dados?.disponivel === true;
  const url = cartao.dados?.url;
  const recarregar = cartao.recarregar;
  // Guardado no aparelho para o modo só consulta, em que a API não responde mas o cartão abre.
  useEffect(() => {
    if (pronto && url) guardarCartao({ inscricao, certame, url });
  }, [pronto, url, inscricao, certame]);
  useEffect(() => {
    if (pronto || !cartao.dados) return;
    const t = setInterval(recarregar, 2000);
    return () => clearInterval(t);
  }, [pronto, cartao.dados, recarregar]);

  return (
    <Card style={{ marginBottom: 16 }}>
      <div style={{ display: "flex", gap: 16, alignItems: "center", justifyContent: "space-between", flexWrap: "wrap" }}>
        <div>
          <div style={{ fontWeight: 800, fontSize: 16 }}><IdcardOutlined /> Cartão de confirmação</div>
          <Typography.Text type="secondary">
            {pronto ? "Pronto para ver e imprimir. O local de prova aparece nele no ensalamento." : "Gerando o seu cartão…"}
          </Typography.Text>
        </div>
        <Button type="primary" icon={<ExportOutlined />} iconPlacement="end" href={pronto ? cartao.dados?.url : undefined}
          target="_blank" disabled={!pronto} loading={!pronto}>
          Abrir cartão
        </Button>
      </div>
    </Card>
  );
}

function Pagamento({ inscricao: i, certame: c, cobrancas, aoMudar }: { inscricao: Inscricao; certame: Certame; cobrancas: Cobranca[]; aoMudar: () => void }) {
  const { message } = App.useApp();
  const [gerando, setGerando] = useState<string | null>(null);
  const [aguardando, setAguardando] = useState(false);
  const espera = useRef<ReturnType<typeof setInterval> | null>(null);
  const pendentes = cobrancas.filter((x) => x.estado === "PENDENTE");
  const prazo = janela(c, "pagamento");
  const taxa = c.cargos.find((g) => g.id === i.cargo_id)?.taxa_centavos ?? 0;

  useEffect(() => () => { if (espera.current) clearInterval(espera.current); }, []);

  const gerar = async (metodo: "pix" | "boleto") => {
    setGerando(metodo);
    try {
      await api(`/inscricoes/${i.id}/pagamento`, { json: { metodo } });
      aoMudar();
    } catch (e) {
      message.error(e instanceof ErroApi && e.status === 409 ? "O prazo de pagamento terminou." : "O sistema de pagamento não respondeu. Tente de novo — você não será cobrado duas vezes.");
    } finally {
      setGerando(null);
    }
  };

  // Sandbox: o "candidato pagando no banco". A confirmação chega pelo aviso do prestador, então a
  // tela pergunta de novo a cada 2 s, por até 30 s.
  const simular = async (cobranca: number) => {
    try {
      await api(`/inscricoes/${i.id}/pagamento/${cobranca}/simular`, { metodo: "POST" });
      setAguardando(true);
      let n = 0;
      espera.current = setInterval(() => {
        aoMudar();
        if (++n >= 15 && espera.current) {
          clearInterval(espera.current);
          setAguardando(false);
        }
      }, 2000);
    } catch (e) {
      message.error(e instanceof ErroApi && e.status === 404 ? "O modo sandbox não está ligado no servidor." : "O banco de teste recusou o pagamento.");
    }
  };

  return (
    <Card className="acao-urgente" style={{ marginBottom: 16 }}>
      <Typography.Title level={4} style={{ marginTop: 0, color: "var(--marker-800)" }}>Falta pagar a taxa: {reais(taxa)}</Typography.Title>
      <Typography.Paragraph>
        Sua inscrição foi recebida e só vale depois do pagamento.
        {prazo && <> Prazo: <strong>{dataHora(prazo.fecha_em)}</strong>.</>}
      </Typography.Paragraph>

      {pendentes.map((x) => (
        <Card key={x.id} size="small" style={{ marginBottom: 12 }}>
          <Typography.Text strong>
            {x.metodo === "pix" ? <><QrcodeOutlined /> Pix copia e cola</> : <><BarcodeOutlined /> Linha digitável do boleto</>}
          </Typography.Text>
          <Typography.Paragraph type="secondary" style={{ fontSize: 12, marginBottom: 6 }}>
            {x.metodo === "pix" ? `Vale até ${dataHora(x.vence_em)}.` : `Vence em ${new Date(x.vence_em).toLocaleDateString("pt-BR")}. Boleto pago leva até 3 dias úteis para ser confirmado — não pague de novo nesse meio tempo.`}
          </Typography.Paragraph>
          {x.pix_copia_cola || x.linha_digitavel ? (
            <>
              <Typography.Paragraph copyable={{ tooltips: ["Copiar", "Copiado"] }} code style={{ wordBreak: "break-all" }}>
                {x.pix_copia_cola ?? x.linha_digitavel}
              </Typography.Paragraph>
              <Button icon={<ExperimentOutlined />} onClick={() => simular(x.id)} disabled={aguardando}>
                Simular pagamento no banco de teste
              </Button>
            </>
          ) : (
            <Typography.Paragraph type="secondary"><Spin size="small" /> Gerando {x.metodo === "pix" ? "o seu Pix" : "o seu boleto"}…</Typography.Paragraph>
          )}
        </Card>
      ))}

      {aguardando && <Alert type="info" showIcon style={{ marginBottom: 12 }} title="Pagamento feito no banco de teste. Aguardando a confirmação…" />}

      <Space wrap>
        <Button type="primary" icon={<QrcodeOutlined />} loading={gerando === "pix"} onClick={() => gerar("pix")}>
          {pendentes.some((x) => x.metodo === "pix") ? "Ver Pix" : "Pagar com Pix"}
        </Button>
        <Button icon={<BarcodeOutlined />} loading={gerando === "boleto"} onClick={() => gerar("boleto")}>
          {pendentes.some((x) => x.metodo === "boleto") ? "Ver boleto" : "Gerar boleto"}
        </Button>
      </Space>
      <Typography.Paragraph type="secondary" style={{ marginTop: 12, marginBottom: 0, fontSize: 12 }}>
        <CheckCircleOutlined /> Tocar duas vezes não gera duas cobranças.
      </Typography.Paragraph>
    </Card>
  );
}
