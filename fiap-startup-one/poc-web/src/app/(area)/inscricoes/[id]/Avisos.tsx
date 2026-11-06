"use client";

import { useState } from "react";
import { Alert, App, Button, Card, Input, Space, Tag, Tooltip, Typography } from "antd";
import { MailOutlined } from "@ant-design/icons";
import { useSessao } from "@/componentes/Moldura";
import { api, dataHora, useApi, type Aviso } from "@/lib/api";

// Os e-mails que o sistema mandou sobre esta inscrição, com a prova de cada um (o resumo SHA-256 do
// texto que saiu). Quando um aviso volta, o candidato vê aqui, e não só pela falta do e-mail.

export function Avisos({ inscricao }: { inscricao: number }) {
  const avisos = useApi<Aviso[]>(`/inscricoes/${inscricao}/avisos`);
  const { eu } = useSessao();
  if (!avisos.dados?.length) return null;
  // Só pede um e-mail novo se o que falhou ainda é o da conta: depois de trocar, o alerta some.
  const atual = eu.dados.email.trim().toLowerCase();
  const voltou = avisos.dados.some((a) => (a.devolvido || !a.entregue) && a.para.trim().toLowerCase() === atual);
  return (
    <Card title={<><MailOutlined /> Avisos por e-mail</>} style={{ marginTop: 16 }}>
      {voltou && <CorrigirEmail aoCorrigir={avisos.recarregar} />}
      <Space orientation="vertical" size={10} style={{ width: "100%" }}>
        {avisos.dados.map((a) => (
          <div key={a.resumo + a.enviada_em}>
            <div style={{ display: "flex", justifyContent: "space-between", gap: 8 }}>
              <Typography.Text strong style={{ fontSize: 13 }}>{a.assunto}</Typography.Text>
              {a.devolvido ? <Tag color="error">Voltou</Tag> : a.entregue ? <Tag color="success">Enviado</Tag> : <Tag color="error">Não enviado</Tag>}
            </div>
            <div style={{ color: "var(--ink-600)", fontSize: 12 }}>
              {dataHora(a.enviada_em)} ·{" "}
              <Tooltip title={`Prova do aviso: o SHA-256 do assunto e do texto que saíram. ${a.resumo}`}>
                <span style={{ fontFamily: "monospace" }}>{a.resumo.slice(0, 12)}…</span>
              </Tooltip>
            </div>
            {(a.devolvido || !a.entregue) && a.falha && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>{a.falha}</Typography.Text>
            )}
          </div>
        ))}
      </Space>
    </Card>
  );
}

// Aviso que voltou: o endereço não recebe mais (insistir derruba a entrega de todos). Um e-mail novo
// volta a receber os próximos avisos. O que os avisos já disseram está nesta página.
function CorrigirEmail({ aoCorrigir }: { aoCorrigir: () => void }) {
  const { message } = App.useApp();
  const { eu, recarregarEu } = useSessao();
  const [email, setEmail] = useState("");
  const [salvando, setSalvando] = useState(false);
  const valido = /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email.trim());

  const salvar = async () => {
    setSalvando(true);
    try {
      await api("/conta", { metodo: "PUT", json: { ...eu.dados, email: email.trim() } });
      message.success("E-mail atualizado. Os próximos avisos vão para ele.");
      setEmail("");
      recarregarEu();
      aoCorrigir();
    } catch {
      message.error("Não foi possível salvar o e-mail. Confira e tente de novo.");
    } finally {
      setSalvando(false);
    }
  };

  return (
    <Alert type="warning" showIcon style={{ marginBottom: 12 }} title="Um aviso não chegou ao seu e-mail."
      description={<>
        <Typography.Paragraph style={{ marginBottom: 8 }}>
          O e-mail da sua conta é <strong>{eu.dados.email}</strong>. Se ele está errado ou não existe mais, informe outro:
          os próximos avisos vão para ele. Tudo o que os avisos dizem também está nesta página.
        </Typography.Paragraph>
        <Space.Compact style={{ width: "100%", maxWidth: 460 }}>
          <Input aria-label="Novo e-mail" type="email" inputMode="email" autoComplete="email" placeholder="novo@exemplo.com"
            value={email} onChange={(e) => setEmail(e.target.value)} onPressEnter={() => valido && salvar()} />
          <Button type="primary" loading={salvando} disabled={!valido} onClick={salvar}>Atualizar e-mail</Button>
        </Space.Compact>
      </>} />
  );
}
