"use client";

import { useCallback, useEffect, useState } from "react";
import { Alert, App, Button, Card, Col, Descriptions, Empty, InputNumber, Row, Segmented, Select, Space, Table, Tag, Typography } from "antd";
import { ExportOutlined, PlayCircleOutlined, StopOutlined } from "@ant-design/icons";

// O painel de operação: dispara ensaios de carga pelo executor (cmd/ensaiador, repassado em
// /ensaiador), mostra o andamento e a auditoria de cada patamar, muda o nível de degradação, e
// embute os painéis do Grafana, que mostram o sistema em tempo real (anotados com o início e o fim
// de cada ensaio e cada mudança de nível).

const GRAFANA = process.env.NEXT_PUBLIC_GRAFANA_URL ?? "http://127.0.0.1:3000";

type Cenario = { id: string; nome: string; descricao: string };
type Patamar = {
  mult?: number; taxa?: number; rodada: string; inicio: string; fim?: string; saida_k6?: number;
  enviadas: number; pagas: number; problemas: string[]; passou?: boolean;
};
type Ensaio = {
  id: string; estado: "rodando" | "passou" | "quebrou" | "parado" | "erro"; inicio: string; fim?: string; erro?: string;
  pedido: { cenario: string; mult: number; duracao: string; certame: number; teto: number; taxa: number };
  patamares: Patamar[]; log: string[];
};
type EstadoExecutor = { cenarios: Cenario[]; atual: Ensaio | null; historico: Ensaio[] };

const corEstado: Record<Ensaio["estado"], string> = { rodando: "processing", passou: "success", quebrou: "error", parado: "default", erro: "error" };
const textoEstado: Record<Ensaio["estado"], string> = { rodando: "rodando", passou: "passou em tudo", quebrou: "quebrou", parado: "parado", erro: "erro" };
const niveis = [
  { value: 0, label: "Normal" },
  { value: 1, label: "Sala de espera" },
  { value: 2, label: "Só o essencial" },
  { value: 3, label: "Só consulta" },
];

async function executor<T>(caminho: string, json?: unknown): Promise<T> {
  const r = await fetch(`/ensaiador${caminho}`, json === undefined ? { cache: "no-store" } : {
    method: "POST", headers: { "Content-Type": "application/json", "X-Lisura-Painel": "1" }, body: JSON.stringify(json),
  });
  const corpo = await r.json().catch(() => null);
  if (!r.ok) throw new Error(corpo?.erro ?? `erro ${r.status}`);
  return corpo as T;
}

const hora = (iso?: string) => (iso ? new Date(iso).toLocaleTimeString("pt-BR") : "—");
const duracao = (de: string, ate?: string) => {
  const s = Math.max(0, Math.round(((ate ? Date.parse(ate) : Date.now()) - Date.parse(de)) / 1000));
  return `${Math.floor(s / 60)} min ${s % 60} s`;
};

export function Painel() {
  const { message } = App.useApp();
  const [estado, setEstado] = useState<EstadoExecutor>();
  const [fora, setFora] = useState<string>();
  const [cenario, setCenario] = useState("ultimo-dia");
  const [mult, setMult] = useState(40);
  const [tempo, setTempo] = useState("1m");
  const [certame, setCertame] = useState(5);
  const [teto, setTeto] = useState(1280);
  const [taxa, setTaxa] = useState(500);
  const [enviando, setEnviando] = useState(false);
  const [nivel, setNivel] = useState<number>();

  const ler = useCallback(async () => {
    try {
      const e = await executor<EstadoExecutor>("/estado");
      setEstado({ ...e, cenarios: e.cenarios ?? [], historico: e.historico ?? [] });
      setFora(undefined);
    } catch {
      setFora("O executor de ensaios não respondeu. Suba com observabilidade/subir-observabilidade.ps1.");
    }
    try {
      const s = await (await fetch("/api/situacao", { cache: "no-store" })).json();
      setNivel(s.nivel);
    } catch {}
  }, []);
  useEffect(() => {
    const primeira = setTimeout(ler, 0);
    const t = setInterval(ler, 1500);
    return () => { clearTimeout(primeira); clearInterval(t); };
  }, [ler]);

  const iniciar = async () => {
    setEnviando(true);
    try {
      await executor("/ensaios", { cenario, mult, duracao: tempo, certame, teto, taxa });
      message.success("Ensaio iniciado. Acompanhe abaixo e nos painéis.");
      ler();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "O executor recusou o ensaio.");
    } finally {
      setEnviando(false);
    }
  };
  const parar = async () => {
    try {
      await executor("/parar", {});
      message.info("Parando o k6…");
    } catch (e) {
      message.error(e instanceof Error ? e.message : "Não deu para parar.");
    }
  };
  const mudarNivel = async (n: number) => {
    try {
      await executor("/nivel", { nivel: n, taxa_por_segundo: n === 1 ? 20 : 0, folga: 0 });
      setNivel(n);
      message.success(`Nível: ${niveis[n].label}. A mudança aparece anotada nos painéis.`);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "A API recusou a mudança.");
    }
  };

  const atual = estado?.atual;
  return (
    <>
      <Typography.Title level={2} style={{ marginTop: 0 }}>Operação e ensaios de carga</Typography.Title>
      <Typography.Paragraph type="secondary" style={{ maxWidth: 820 }}>
        Dispara o k6 contra o sistema desta máquina e mostra, em tempo real, o que acontece com ele: instâncias, tráfego,
        fila, banco e processo. Cada patamar termina com a auditoria (nenhuma inscrição perdida, nenhuma cobrança dupla).
        O H1 é 40 vezes a carga média por 3 horas; aqui, para ver acontecer, minutos bastam.
      </Typography.Paragraph>
      {fora && <Alert type="warning" showIcon style={{ marginBottom: 16 }} title={fora} />}

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={10}>
          <Card title="Disparar um ensaio">
            <Space orientation="vertical" size={14} style={{ width: "100%" }}>
              <Segmented block value={cenario} onChange={(v) => { setCenario(String(v)); if (v === "pico-resultado") setCertame(6); else setCertame(5); }}
                options={(estado?.cenarios ?? []).map((c) => ({ value: c.id, label: c.nome }))} />
              <Typography.Text type="secondary">{estado?.cenarios.find((c) => c.id === cenario)?.descricao}</Typography.Text>
              <Descriptions column={1} size="small" items={[
                ...(cenario !== "pico-resultado" ? [{ key: "m", label: cenario === "escada" ? "Começar em" : "Multiplicador", children: (
                  <InputNumber aria-label="Multiplicador" min={1} max={2560} value={mult} onChange={(v) => setMult(v ?? 1)} suffix="×" />) }] : []),
                ...(cenario === "escada" ? [{ key: "t", label: "Teto", children: (
                  <InputNumber aria-label="Teto" min={mult} max={2560} value={teto} onChange={(v) => setTeto(v ?? mult)} suffix="×" />) }] : []),
                ...(cenario === "pico-resultado" ? [{ key: "x", label: "Chegadas por segundo", children: (
                  <InputNumber aria-label="Chegadas por segundo" min={1} max={20000} step={100} value={taxa} onChange={(v) => setTaxa(v ?? 100)} />) }] : []),
                { key: "d", label: cenario === "escada" ? "Por patamar" : "Duração", children: (
                  <Select aria-label="Duração" value={tempo} onChange={setTempo} style={{ width: 140 }}
                    options={["30s", "1m", "2m", "5m", "10m", "30m", "3h"].map((d) => ({ value: d, label: d }))} />) },
                { key: "c", label: "Certame", children: <InputNumber aria-label="Certame" min={1} value={certame} onChange={(v) => setCertame(v ?? 1)} /> },
              ]} />
              <Button type="primary" icon={<PlayCircleOutlined />} onClick={iniciar} loading={enviando} disabled={!!atual || !!fora}>
                {atual ? "Já há um ensaio rodando" : "Iniciar ensaio"}
              </Button>
            </Space>
          </Card>

          <Card title="Nível de degradação" style={{ marginTop: 16 }}>
            <Typography.Paragraph type="secondary">
              Sob carga, o sistema degrada em vez de cair: sala de espera com senha, só o essencial (inscrição e pagamento), ou só
              consulta (o estático continua no ar). Mudar aqui aparece como anotação nos painéis.
            </Typography.Paragraph>
            <Segmented block value={nivel} onChange={(v) => mudarNivel(Number(v))} options={niveis} disabled={!!fora} />
          </Card>
        </Col>

        <Col xs={24} lg={14}>
          <Card title={atual ? <>Ensaio {atual.id} <Tag color={corEstado[atual.estado]}>{textoEstado[atual.estado]}</Tag></> : "Nenhum ensaio rodando"}
            extra={atual && <Button danger icon={<StopOutlined />} onClick={parar}>Parar</Button>}>
            {atual ? (
              <>
                <Typography.Paragraph type="secondary">
                  Começou às {hora(atual.inicio)} · {duracao(atual.inicio)} · {estado?.cenarios.find((c) => c.id === atual.pedido.cenario)?.nome}
                </Typography.Paragraph>
                <Patamares p={atual.patamares} />
                <pre aria-label="Saída do k6" style={{ marginTop: 12, maxHeight: 240, overflow: "auto", background: "#14233A", color: "#E6EBF2",
                  padding: 12, borderRadius: 8, fontSize: 12, whiteSpace: "pre-wrap" }}>
                  {atual.log.slice(-60).join("\n") || "Aguardando o k6…"}
                </pre>
              </>
            ) : (
              <Empty description="Escolha um cenário e inicie. O último resultado fica no histórico abaixo." />
            )}
          </Card>
          {(estado?.historico.length ?? 0) > 0 && (
            <Card title="Histórico desta sessão do executor" style={{ marginTop: 16 }}>
              <Space orientation="vertical" style={{ width: "100%" }} size={12}>
                {estado!.historico.map((h) => (
                  <div key={h.id}>
                    <Typography.Text strong>{h.id}</Typography.Text>{" "}
                    <Tag color={corEstado[h.estado]}>{textoEstado[h.estado]}</Tag>
                    <Typography.Text type="secondary">
                      {estado!.cenarios.find((c) => c.id === h.pedido.cenario)?.nome} · {duracao(h.inicio, h.fim)}
                    </Typography.Text>
                    {h.erro && <Alert type="error" showIcon style={{ marginTop: 6 }} title={h.erro} />}
                    <Patamares p={h.patamares} />
                  </div>
                ))}
              </Space>
            </Card>
          )}
        </Col>
      </Row>

      <Paineis />
    </>
  );
}

function Patamares({ p }: { p: Patamar[] }) {
  if (!p.length) return null;
  return (
    <Table size="small" pagination={false} rowKey="rodada" style={{ marginTop: 8 }} dataSource={p} columns={[
      { title: "Patamar", render: (_, x) => (x.mult ? `${x.mult}×` : `${x.taxa}/s`) },
      { title: "Rodada", dataIndex: "rodada" },
      { title: "Enviadas", dataIndex: "enviadas", align: "right" },
      { title: "Pagas", dataIndex: "pagas", align: "right" },
      { title: "Duração", render: (_, x) => duracao(x.inicio, x.fim) },
      {
        title: "Resultado", render: (_, x) => x.passou === undefined
          ? <Tag color="processing">rodando</Tag>
          : x.passou ? <Tag color="success">passou</Tag>
          : <><Tag color="error">quebrou</Tag><div style={{ fontSize: 12, color: "var(--state-danger)" }}>{x.problemas.join("; ")}</div></>,
      },
    ]} />
  );
}

// Os painéis do Grafana, embutidos (o Grafana local permite, só leitura e sem login).
const paineis = [
  { id: 1, titulo: "Instâncias", h: 140, span: 6 }, { id: 2, titulo: "Nível", h: 140, span: 6 },
  { id: 3, titulo: "Requisições/s", h: 140, span: 6 }, { id: 4, titulo: "p95", h: 140, span: 6 },
  { id: 20, titulo: "VUs e multiplicador", h: 260, span: 12 }, { id: 22, titulo: "p95 visto pelo candidato", h: 260, span: 12 },
  { id: 30, titulo: "Requisições por rota", h: 300, span: 12 }, { id: 40, titulo: "Fila pendente", h: 300, span: 12 },
  { id: 50, titulo: "Conexões do banco", h: 260, span: 12 }, { id: 51, titulo: "CPU da API", h: 260, span: 12 },
];

function Paineis() {
  const [vivo, setVivo] = useState<boolean>();
  useEffect(() => {
    fetch(`${GRAFANA}/api/health`, { mode: "no-cors" }).then(() => setVivo(true), () => setVivo(false));
  }, []);
  return (
    <Card style={{ marginTop: 16 }} title="Em tempo real (Grafana, atualiza a cada 5 s)"
      extra={<Button icon={<ExportOutlined />} iconPlacement="end" href={`${GRAFANA}/d/lisura/lisura`} target="_blank">Painel completo</Button>}>
      {vivo === false ? (
        <Alert type="warning" showIcon title="O Grafana não respondeu em 127.0.0.1:3000."
          description="Suba com observabilidade/subir-observabilidade.ps1. Na AWS, os painéis vêm do CloudWatch (ver Infraestrutura)." />
      ) : (
        <Row gutter={[12, 12]}>
          {paineis.map((p) => (
            <Col key={p.id} xs={24} md={p.span}>
              <iframe title={p.titulo} loading="lazy" style={{ width: "100%", height: p.h, border: 0, borderRadius: 8, background: "#fff" }}
                src={`${GRAFANA}/d-solo/lisura/lisura?orgId=1&panelId=${p.id}&from=now-15m&to=now&refresh=5s&theme=light`} />
            </Col>
          ))}
        </Row>
      )}
    </Card>
  );
}
