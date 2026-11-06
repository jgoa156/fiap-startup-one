"use client";

import { Suspense, useCallback, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Alert, Button, Divider, Form, Input, Segmented, Typography } from "antd";
import { CheckOutlined, IdcardOutlined, SafetyCertificateFilled, TeamOutlined, UserAddOutlined } from "@ant-design/icons";
import { api, ErroApi, type Fila } from "@/lib/api";
import { ModoConsulta, SalaDeEspera } from "@/componentes/Espera";
import "./entrar.css";

// Login do Inclusa (mockups/sistema-direcao-ab/01-login): marca à esquerda, formulário à direita,
// tipo de conta escolhido num segmentado, ação secundária para quem ainda não tem conta.

const soDigitos = (v: string) => v.replace(/\D/g, "").slice(0, 11);
const mascara = (v: string) =>
  soDigitos(v)
    .replace(/^(\d{3})(\d)/, "$1.$2")
    .replace(/^(\d{3})\.(\d{3})(\d)/, "$1.$2.$3")
    .replace(/\.(\d{3})(\d)/, ".$1-$2");

function destinoSeguro(v: string | null) {
  return v && v.startsWith("/") && !v.startsWith("//") && !v.startsWith("/\\") ? v : "/";
}

function Formulario() {
  const router = useRouter();
  const volta = destinoSeguro(useSearchParams().get("volta"));
  const [perfil, setPerfil] = useState<"candidato" | "banca">("candidato");
  const [modo, setModo] = useState<"entrar" | "criar">("entrar");
  const [erro, setErro] = useState<string>();
  const [enviando, setEnviando] = useState(false);
  const [fila, setFila] = useState<Fila>();
  const [consulta, setConsulta] = useState(false);
  const [vez, setVez] = useState(false);

  const enviar = async (v: { cpf: string; nome?: string; email?: string }) => {
    setEnviando(true);
    setErro(undefined);
    try {
      if (modo === "entrar") await api("/sessoes", { json: { cpf: soDigitos(v.cpf) } });
      else await api("/contas", { json: { cpf: soDigitos(v.cpf), nome: v.nome, email: v.email } });
      router.replace(volta);
    } catch (e) {
      const status = e instanceof ErroApi ? e.status : 0;
      setEnviando(false);
      if (e instanceof ErroApi && e.fila) return setFila(e.fila);
      if (e instanceof ErroApi && e.degradado === 3) return setConsulta(true);
      setErro(
        status === 404
          ? "Não encontramos conta com este CPF. Crie a sua abaixo."
          : status === 409
            ? "Este CPF já tem conta. Use Entrar."
            : status === 422
              ? "Confira os dados: o CPF precisa ser válido e o e-mail, completo."
              : "Não foi possível falar com o servidor. Tente de novo em instantes.",
      );
    }
  };

  // Na fila, pergunta de novo no ritmo dela; um 401 do /eu quer dizer que passou da sala de espera
  // (só falta entrar), e um 200, que já havia sessão.
  const tentar = useCallback(async () => {
    try {
      await api("/eu");
      router.replace(volta);
    } catch (e) {
      if (e instanceof ErroApi && e.fila) return setFila(e.fila);
      setFila(undefined);
      setVez(true);
    }
  }, [router, volta]);

  if (fila) return <SalaDeEspera fila={fila} tentar={tentar} />;
  if (consulta) return <ModoConsulta aoVoltar={() => setConsulta(false)} />;

  return (
    <div className="entrar">
      <aside className="entrar-marca">
        <div className="entrar-logo">
          <SafetyCertificateFilled style={{ color: "var(--marker-400)", fontSize: 40 }} /> Lisura
        </div>
        <h1>
          Do edital ao resultado,
          <br />
          <span>sem fila e sem susto.</span>
        </h1>
        <p>Uma conta só, com o seu CPF, para se inscrever em concursos e processos seletivos de todas as bancas.</p>
        <div className="entrar-selos">
          <span>
            <CheckOutlined /> Inscrição salva a cada passo
          </span>
          <span>
            <CheckOutlined /> Pix ou boleto
          </span>
          <span>
            <CheckOutlined /> Cada etapa registrada
          </span>
        </div>
      </aside>

      <main className="entrar-form">
        <div className="entrar-caixa">
          <Typography.Title level={2} style={{ color: "var(--brand-800)", marginBottom: 4 }}>
            {modo === "entrar" ? "Entrar na Lisura" : "Criar minha conta"}
          </Typography.Title>
          <Typography.Paragraph type="secondary">Candidatos e bancas têm acessos separados — escolha o seu.</Typography.Paragraph>

          <Segmented
            block
            value={perfil}
            onChange={(v) => setPerfil(v as typeof perfil)}
            options={[
              { value: "candidato", label: <span><IdcardOutlined /> Sou candidato</span> },
              { value: "banca", label: <span><TeamOutlined /> Sou da banca</span> },
            ]}
            style={{ marginBottom: 24 }}
          />

          {perfil === "banca" ? (
            <Alert
              type="info"
              showIcon
              title="O painel da banca ainda não faz parte desta prova de conceito."
              description="A POC cobre o caminho do candidato: inscrição, pagamento e acompanhamento."
            />
          ) : (
            <Form layout="vertical" onFinish={enviar} requiredMark={false} key={modo}>
              <Form.Item
                label="CPF"
                name="cpf"
                normalize={mascara}
                rules={[{ required: true, message: "Informe o CPF." }, { len: 14, message: "O CPF tem 11 dígitos." }]}
              >
                <Input inputMode="numeric" autoComplete="username" placeholder="000.000.000-00" autoFocus />
              </Form.Item>
              {modo === "criar" && (
                <>
                  <Form.Item label="Nome completo" name="nome" rules={[{ required: true, message: "Informe o nome." }]}>
                    <Input autoComplete="name" />
                  </Form.Item>
                  <Form.Item
                    label="E-mail"
                    name="email"
                    rules={[
                      { required: true, message: "Informe o e-mail." },
                      { type: "email", message: "E-mail incompleto." },
                    ]}
                  >
                    <Input autoComplete="email" inputMode="email" />
                  </Form.Item>
                </>
              )}
              {vez && !erro && (
                <Alert type="success" showIcon title="Chegou a sua vez. Entre agora." style={{ marginBottom: 16 }} />
              )}
              {erro && <Alert type="error" showIcon title={erro} style={{ marginBottom: 16 }} />}
              <Button type="primary" htmlType="submit" block size="large" loading={enviando}>
                {modo === "entrar" ? "Entrar" : "Criar conta e entrar"}
              </Button>
              {modo === "entrar" && (
                <Typography.Paragraph type="secondary" style={{ fontSize: 13, marginTop: 8 }}>
                  Nesta prova de conceito você entra só com o CPF. Em produção, o acesso é pelo gov.br.
                </Typography.Paragraph>
              )}
            </Form>
          )}

          {perfil === "candidato" && (
            <>
              <Divider plain>ou</Divider>
              <Button
                block
                size="large"
                icon={modo === "entrar" ? <UserAddOutlined /> : <IdcardOutlined />}
                onClick={() => {
                  setModo(modo === "entrar" ? "criar" : "entrar");
                  setErro(undefined);
                }}
              >
                {modo === "entrar" ? "Primeiro acesso — criar minha conta" : "Já tenho conta — entrar"}
              </Button>
            </>
          )}
        </div>
      </main>
    </div>
  );
}

export default function Entrar() {
  return (
    <Suspense>
      <Formulario />
    </Suspense>
  );
}
