"use client";

import { useState } from "react";
import { Alert, App, Button, Card, Descriptions, Form, Input, Select, Steps, Tag, Typography, Upload } from "antd";
import { CheckOutlined, CloseOutlined, FileProtectOutlined, SendOutlined, UploadOutlined } from "@ant-design/icons";
import { useSessao } from "@/componentes/Moldura";
import { Escolha } from "@/componentes/Escolha";
import { api, enviarDocumento, ErroApi, reais, useApi, type Certame, type Dados, type Documento, type Inscricao } from "@/lib/api";
import { necessidades, pendencias as rotuloPendencia, ufs } from "@/lib/rotulos";

// Formulário em passos do Inclusa (mockups/sistema-direcao-ab/03-cadastro-estudante): barra de
// passos, "Rascunho salvo", escolhas em cartões e chips, um botão primário por passo.

const PASSOS = ["Dados pessoais", "Cargo", "Cidade de prova", "Reserva de vagas", "Atendimento", "Declarações", "Revisão e envio"];
const PASSO_DA_PENDENCIA: Record<string, number> = { dados_pessoais: 0, cidade_prova: 2, laudo: 4, declaracoes: 5 };

function mensagem(e: unknown) {
  if (e instanceof ErroApi) {
    if (e.status === 409) return "O prazo terminou ou a inscrição já foi enviada.";
    if (e.status === 422) return e.faltando?.length ? "Ainda falta preencher algumas partes." : "Algum dado não confere. Revise e tente de novo.";
  }
  return "Não foi possível salvar. Verifique a conexão e tente de novo.";
}

export function Formulario({ inscricao: i, certame: c, aoSalvar }: { inscricao: Inscricao; certame: Certame; aoSalvar: () => void }) {
  const { message } = App.useApp();
  const { eu, recarregarEu } = useSessao();
  const pend = useApi<{ faltando: string[] }>(`/inscricoes/${i.id}/pendencias`);
  const docs = useApi<Documento[]>("/conta/documentos");
  const [passo, setPasso] = useState(0);
  const [salvoEm, setSalvoEm] = useState<Date | null>(null);
  const [enviando, setEnviando] = useState(false);
  const faltando = pend.dados?.faltando ?? [];
  const laudos = (docs.dados ?? []).filter((d) => d.tipo === "laudo");
  const exigeLaudo = i.necessidades.length > 0 || i.cotas.some((q) => c.cotas.find((x) => x.codigo === q)?.exige_laudo);

  // Cada passo salva na hora; o selo mostra quando foi o último salvamento.
  const salvar = async (caminho: string, json: unknown) => {
    try {
      await api(caminho, { metodo: "PUT", json });
      setSalvoEm(new Date());
      aoSalvar();
      pend.recarregar();
      return true;
    } catch (e) {
      message.error(mensagem(e));
      return false;
    }
  };

  const enviar = async () => {
    setEnviando(true);
    try {
      await api(`/inscricoes/${i.id}/finalizar`, { metodo: "POST" });
      message.success("Inscrição enviada.");
      aoSalvar();
    } catch (e) {
      message.error(mensagem(e));
      pend.recarregar();
      setEnviando(false);
    }
  };

  const conteudo = [
    // 0 — dados pessoais (ficam na conta, valem para todos os certames)
    <Form<Dados>
      key="dados"
      layout="vertical"
      requiredMark={false}
      initialValues={eu.dados}
      onFinish={async (v) => {
        if (await salvar("/conta", v)) {
          recarregarEu();
          setPasso(1);
        }
      }}
    >
      <Typography.Paragraph type="secondary">Estes dados ficam na sua conta e valem para todos os certames.</Typography.Paragraph>
      <Form.Item label="Nome completo" name="nome" rules={[{ required: true, message: "Informe o nome." }]}>
        <Input autoComplete="name" />
      </Form.Item>
      <Form.Item label="E-mail" name="email" rules={[{ required: true, type: "email", message: "E-mail incompleto." }]}>
        <Input autoComplete="email" inputMode="email" />
      </Form.Item>
      <Form.Item label="Data de nascimento" name="data_nascimento" rules={[{ required: true, message: "Informe a data." }]}>
        <Input type="date" autoComplete="bday" style={{ maxWidth: 220 }} />
      </Form.Item>
      <div style={{ display: "flex", gap: 16, flexWrap: "wrap" }}>
        <Form.Item label="Município onde mora" name="municipio" rules={[{ required: true, message: "Informe o município." }]} style={{ flex: "1 1 260px" }}>
          <Input autoComplete="address-level2" />
        </Form.Item>
        <Form.Item label="UF" name="uf" rules={[{ required: true, message: "Escolha a UF." }]} style={{ flex: "0 0 140px" }}>
          <Select showSearch options={ufs.map((u) => ({ value: u, label: u }))} placeholder="UF" />
        </Form.Item>
      </div>
      <Button type="primary" htmlType="submit">Salvar e continuar</Button>
    </Form>,

    // 1 — cargo
    <Escolha
      key="cargo"
      rotulo="Cargo"
      colunas={2}
      valor={i.cargo_id}
      aoMudar={(v) => salvar(`/inscricoes/${i.id}/cargo`, { cargo_id: v })}
      opcoes={c.cargos.map((g) => ({ valor: g.id, titulo: g.nome, detalhe: `${g.vagas} vagas · taxa ${reais(g.taxa_centavos)}` }))}
    />,

    // 2 — cidade de prova
    <Escolha
      key="cidade"
      rotulo="Cidade de prova"
      valor={i.cidade_prova_id}
      aoMudar={(v) => salvar(`/inscricoes/${i.id}/cidade`, { cidade_id: v })}
      opcoes={c.cidades_prova.map((d) => ({ valor: d.id, titulo: d.nome, detalhe: d.uf }))}
    />,

    // 3 — reserva de vagas
    <div key="cotas">
      <Typography.Paragraph type="secondary">Marque se quiser concorrer também pela reserva. Sem marcar, você concorre na ampla concorrência.</Typography.Paragraph>
      <Escolha
        rotulo="Reserva de vagas"
        multipla
        compacto
        valor={i.cotas}
        aoMudar={(v) => salvar(`/inscricoes/${i.id}/cotas`, { cotas: v })}
        opcoes={c.cotas.map((q) => ({ valor: q.codigo, titulo: q.nome, detalhe: q.exige_laudo ? "exige laudo médico" : undefined }))}
      />
    </div>,

    // 4 — atendimento especial e laudo
    <div key="atendimento">
      <Typography.Paragraph type="secondary">Precisa de algum apoio no dia da prova? Deixe em branco se não.</Typography.Paragraph>
      <Escolha
        rotulo="Atendimento especial"
        multipla
        compacto
        valor={i.necessidades}
        aoMudar={(v) => salvar(`/inscricoes/${i.id}/atendimento`, { necessidades: v, laudo_documento_id: i.laudo_documento_id })}
        opcoes={Object.entries(necessidades).map(([valor, titulo]) => ({ valor, titulo }))}
      />
      {exigeLaudo && (
        <div style={{ marginTop: 24 }}>
          <Typography.Title level={5}>Laudo médico</Typography.Title>
          {!i.laudo_documento_id && (
            <Alert type="warning" showIcon style={{ marginBottom: 12 }}
              title="A reserva para pessoa com deficiência e o atendimento especial exigem laudo médico." />
          )}
          <Typography.Paragraph type="secondary">
            Ao escolher, você autoriza o envio deste laudo <strong>só para esta inscrição</strong>. Ele continua guardado na sua conta e não vai para outras bancas.
          </Typography.Paragraph>
          <Escolha
            rotulo="Laudo a compartilhar"
            colunas={2}
            valor={i.laudo_documento_id ?? 0}
            aoMudar={(v) => salvar(`/inscricoes/${i.id}/atendimento`, { necessidades: i.necessidades, laudo_documento_id: v === 0 ? null : v })}
            opcoes={[
              { valor: 0, titulo: "Não compartilhar", icone: <CloseOutlined /> },
              ...laudos.map((d) => ({ valor: d.id, titulo: d.nome_arquivo, detalhe: `enviado em ${new Date(d.enviado_em).toLocaleDateString("pt-BR")}`, icone: <FileProtectOutlined /> })),
            ]}
          />
          <Upload
            accept=".pdf,.jpg,.jpeg,.png"
            showUploadList={false}
            customRequest={async ({ file, onSuccess, onError }) => {
              try {
                const r = await enviarDocumento("laudo", file as File);
                docs.recarregar();
                await salvar(`/inscricoes/${i.id}/atendimento`, { necessidades: i.necessidades, laudo_documento_id: r.id });
                onSuccess?.(r);
              } catch (e) {
                message.error("Arquivo recusado: envie PDF, JPG ou PNG de até 5 MB.");
                onError?.(e as Error);
              }
            }}
          >
            <Button icon={<UploadOutlined />} style={{ marginTop: 12 }}>Enviar um laudo novo</Button>
          </Upload>
        </div>
      )}
    </div>,

    // 5 — declarações
    <div key="declaracoes">
      <Typography.Paragraph>
        Declaro que li o edital, que as informações desta inscrição são verdadeiras e que atendo aos requisitos do cargo.
      </Typography.Paragraph>
      <Escolha
        rotulo="Aceite das declarações"
        colunas={4}
        valor={i.declaracoes_aceitas_em ? "sim" : null}
        aoMudar={(v) => salvar(`/inscricoes/${i.id}/declaracoes`, { aceito: v === "sim" })}
        opcoes={[
          { valor: "sim", titulo: "Sim, declaro", icone: <CheckOutlined /> },
          { valor: "nao", titulo: "Não", icone: <CloseOutlined /> },
        ]}
      />
    </div>,

    // 6 — revisão e envio
    <div key="revisao">
      <Descriptions column={1} bordered size="middle" items={[
        { key: "n", label: "Candidato", children: eu.dados.nome || "—" },
        { key: "c", label: "Cargo", children: c.cargos.find((g) => g.id === i.cargo_id)?.nome },
        { key: "d", label: "Cidade de prova", children: c.cidades_prova.find((d) => d.id === i.cidade_prova_id)?.nome ?? "—" },
        { key: "q", label: "Reserva de vagas", children: i.cotas.length ? i.cotas.map((q) => c.cotas.find((x) => x.codigo === q)?.nome).join("; ") : "Ampla concorrência" },
        { key: "a", label: "Atendimento especial", children: i.necessidades.length ? i.necessidades.map((x) => necessidades[x]).join("; ") : "Nenhum" },
        { key: "l", label: "Laudo compartilhado", children: i.laudo_documento_id ? "Sim, só com esta banca" : "Não" },
      ]} />
      {faltando.length > 0 ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginTop: 16 }}
          title="Ainda falta preencher:"
          description={
            <ul style={{ margin: 0, paddingLeft: 18 }}>
              {faltando.map((f) => (
                <li key={f}>
                  <Button type="link" style={{ padding: 0, height: "auto" }} onClick={() => setPasso(PASSO_DA_PENDENCIA[f] ?? 0)}>
                    {rotuloPendencia[f] ?? f}
                  </Button>
                </li>
              ))}
            </ul>
          }
        />
      ) : (
        <Alert type="success" showIcon style={{ marginTop: 16 }} title="Tudo preenchido. Depois de enviada, a inscrição não pode mais ser alterada." />
      )}
      <div style={{ marginTop: 16, display: "flex", justifyContent: "flex-end" }}>
        <Button type="primary" size="large" icon={<SendOutlined />} loading={enviando} disabled={faltando.length > 0} onClick={enviar}>
          Enviar inscrição
        </Button>
      </div>
    </div>,
  ];

  return (
    <Card
      style={{ marginTop: 16 }}
      title={`Passo ${passo + 1} de ${PASSOS.length} · ${PASSOS[passo]}`}
      extra={salvoEm && <Tag color="success" icon={<CheckOutlined />}>Rascunho salvo às {salvoEm.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" })}</Tag>}
    >
      <Steps
        size="small"
        current={passo}
        onChange={setPasso}
        responsive
        items={PASSOS.map((t, n) => ({
          title: t,
          status: n === passo ? "process" : Object.entries(PASSO_DA_PENDENCIA).some(([f, p]) => p === n && faltando.includes(f)) ? "wait" : "finish",
        }))}
        style={{ marginBottom: 28 }}
      />
      {conteudo[passo]}
      {passo !== 0 && (
        <div style={{ marginTop: 24, display: "flex", justifyContent: "space-between" }}>
          <Button onClick={() => setPasso(passo - 1)}>Voltar</Button>
          {passo !== PASSOS.length - 1 && <Button type="primary" onClick={() => setPasso(passo + 1)}>Próximo</Button>}
        </div>
      )}
    </Card>
  );
}
