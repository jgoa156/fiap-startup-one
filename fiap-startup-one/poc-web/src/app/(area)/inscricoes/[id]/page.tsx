"use client";

import { useParams } from "next/navigation";
import { Alert, Card, Skeleton, Steps, Typography } from "antd";
import { useAjuda } from "@/componentes/Moldura";
import { Selo } from "@/componentes/Selo";
import { useApi, useCertame, type Inscricao } from "@/lib/api";
import { estados, jornada, passoDaJornada } from "@/lib/rotulos";
import { Formulario } from "./Formulario";
import { Situacao } from "./Situacao";

// Página da inscrição. É o "Prontuário" do Inclusa (mockups/sistema-direcao-ab/05): cabeçalho com
// situação e trilha da jornada. Em rascunho, o corpo é o formulário em passos; depois de
// enviada, é a situação, o pagamento e o histórico.

export default function PaginaInscricao() {
  const { id } = useParams<{ id: string }>();
  const inscricao = useApi<Inscricao>(`/inscricoes/${id}`);
  const certame = useCertame(inscricao.dados ? inscricao.dados.certame_id : null);
  useAjuda(
    "Sua inscrição",
    <>
      <p>Enquanto a inscrição está em preenchimento, cada passo é salvo assim que você muda algo — pode fechar e voltar depois.</p>
      <p>Depois de enviada, ela não pode mais ser alterada. Aqui você paga a taxa e acompanha cada etapa.</p>
    </>,
  );

  if (inscricao.erro) {
    return <Alert type="error" showIcon title={inscricao.erro.status === 404 ? "Inscrição não encontrada." : inscricao.erro.message} />;
  }
  const i = inscricao.dados;
  const c = certame.dados;
  if (!i || !c) return <Card><Skeleton active /></Card>;

  const cargo = c.cargos.find((g) => g.id === i.cargo_id);
  const e = estados[i.estado];
  const recarregar = () => inscricao.recarregar();

  return (
    <>
      <Card style={{ marginTop: 4 }}>
        <div style={{ display: "flex", gap: 16, alignItems: "flex-start", justifyContent: "space-between", flexWrap: "wrap" }}>
          <div>
            <Typography.Title level={3} style={{ color: "var(--brand-800)", margin: 0 }}>{c.nome}</Typography.Title>
            <Typography.Text type="secondary">{c.banca} · {cargo?.nome}</Typography.Text>
          </div>
          <Selo tom={e.tom}>{e.rotulo}</Selo>
        </div>
        <Steps
          style={{ marginTop: 24 }}
          current={passoDaJornada(i.estado)}
          status={e.tom === "erro" ? "error" : undefined}
          items={jornada.map((t) => ({ title: t }))}
          responsive
        />
      </Card>

      {i.estado === "RASCUNHO" ? (
        <Formulario inscricao={i} certame={c} aoSalvar={recarregar} />
      ) : (
        <Situacao inscricao={i} certame={c} aoMudar={recarregar} />
      )}
    </>
  );
}
