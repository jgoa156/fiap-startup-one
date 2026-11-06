"use client";

import { useEffect, useState } from "react";
import { Alert, App, Button, Card, Input, Space, Tag, Typography } from "antd";
import { RollbackOutlined } from "@ant-design/icons";
import { api, dataHora, ErroApi, reais, useApi, type Devolucao } from "@/lib/api";
import { motivosDevolucao, recusasChave } from "@/lib/rotulos";

// O que volta ao candidato: pagamento em duplicidade, pago depois do prazo, certame cancelado. O Pix
// volta sozinho para a conta de onde saiu; o boleto volta por Pix para a chave do CPF. Quando o CPF
// não é chave Pix, a devolução espera o candidato informar outra chave dele aqui.

export function Devolucoes({ inscricao }: { inscricao: number }) {
  const lista = useApi<Devolucao[]>(`/inscricoes/${inscricao}/devolucoes`);
  const andando = (lista.dados ?? []).some((d) => d.estado === "PENDENTE");
  const recarregar = lista.recarregar;
  // A devolução sai pela fila: enquanto alguma anda, pergunta de novo.
  useEffect(() => {
    if (!andando) return;
    const t = setInterval(recarregar, 3000);
    return () => clearInterval(t);
  }, [andando, recarregar]);

  if (!lista.dados?.length) return null;
  return (
    <Card title={<><RollbackOutlined /> Devoluções</>} style={{ marginBottom: 16 }}>
      <Space orientation="vertical" size={12} style={{ width: "100%" }}>
        {lista.dados.map((d) => <Item key={d.id} inscricao={inscricao} d={d} aoMudar={recarregar} />)}
      </Space>
    </Card>
  );
}

function Item({ inscricao, d, aoMudar }: { inscricao: number; d: Devolucao; aoMudar: () => void }) {
  const titulo = (
    <div style={{ display: "flex", justifyContent: "space-between", gap: 8, flexWrap: "wrap" }}>
      <Typography.Text strong>{reais(d.valor_centavos)} — {motivosDevolucao[d.motivo] ?? d.motivo}</Typography.Text>
      <Situacao d={d} />
    </div>
  );
  return (
    <Card size="small">
      {titulo}
      {d.estado === "CONCLUIDA" && d.concluida_em && (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>Devolvido em {dataHora(d.concluida_em)}.</Typography.Text>
      )}
      {d.estado === "PENDENTE" && (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>O banco está processando. Você não precisa fazer nada.</Typography.Text>
      )}
      {d.estado === "RECUSADA" && (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>O banco recusou a devolução. A banca vai entrar em contato com você.</Typography.Text>
      )}
      {d.estado === "AGUARDANDO_CHAVE" && <Chave inscricao={inscricao} d={d} aoMudar={aoMudar} />}
    </Card>
  );
}

function Situacao({ d }: { d: Devolucao }) {
  switch (d.estado) {
    case "CONCLUIDA":
      return <Tag color="success">Devolvido</Tag>;
    case "AGUARDANDO_CHAVE":
      return <Tag color="warning">Falta sua chave Pix</Tag>;
    case "RECUSADA":
      return <Tag color="error">Recusada pelo banco</Tag>;
    default:
      return <Tag>Em andamento</Tag>;
  }
}

function Chave({ inscricao, d, aoMudar }: { inscricao: number; d: Devolucao; aoMudar: () => void }) {
  const { message } = App.useApp();
  const [chave, setChave] = useState("");
  const [enviando, setEnviando] = useState(false);
  const [erro, setErro] = useState<string>();

  const enviar = async () => {
    setEnviando(true);
    setErro(undefined);
    try {
      await api(`/inscricoes/${inscricao}/devolucoes/${d.id}/chave`, { metodo: "PUT", json: { chave } });
      message.success("Chave recebida. O banco confere que ela é sua e faz a devolução.");
      setChave("");
      aoMudar();
    } catch (e) {
      if (e instanceof ErroApi && e.status === 422) setErro("Essa chave não está num formato válido. Use seu e-mail, seu celular com +55 e DDD ou uma chave aleatória.");
      else if (e instanceof ErroApi && e.status === 409) { setErro("Esta devolução já não espera chave."); aoMudar(); }
      else setErro("O sistema não respondeu. Tente de novo: a chave só é usada uma vez.");
    } finally {
      setEnviando(false);
    }
  };

  return (
    <div style={{ marginTop: 8 }}>
      <Typography.Paragraph style={{ marginBottom: 8 }}>
        O boleto não diz de que conta saiu o dinheiro, então ele volta por Pix. O seu CPF não é chave Pix:
        informe outra chave <strong>sua</strong>. O banco confere que a chave está no seu CPF antes de devolver.
      </Typography.Paragraph>
      {d.chave_informada && d.recusa && recusasChave[d.recusa] && (
        <Alert type="error" showIcon style={{ marginBottom: 8 }} title={recusasChave[d.recusa]} />
      )}
      <Space.Compact style={{ width: "100%" }}>
        <Input aria-label="Chave Pix" placeholder="E-mail, celular (+55 DDD número) ou chave aleatória" value={chave}
          onChange={(e) => setChave(e.target.value)} onPressEnter={() => chave.trim() && enviar()} maxLength={77} />
        <Button type="primary" loading={enviando} disabled={!chave.trim()} onClick={enviar}>Enviar chave</Button>
      </Space.Compact>
      {erro && <Typography.Text type="danger" style={{ display: "block", marginTop: 6, fontSize: 12 }}>{erro}</Typography.Text>}
    </div>
  );
}
