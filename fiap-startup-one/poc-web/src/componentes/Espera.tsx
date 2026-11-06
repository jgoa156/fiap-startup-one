"use client";

import { useEffect, useState } from "react";
import { Button, Card, Typography } from "antd";
import { ClockCircleOutlined, ExportOutlined, IdcardOutlined, SafetyCertificateFilled } from "@ant-design/icons";
import { cartoesGuardados, type CartaoGuardado, type Fila } from "@/lib/api";
import "./espera.css";

// Telas da degradação graciosa: em vez de um erro, a sala de espera conta a posição e o tempo, e o
// modo só consulta mostra o que continua no ar (o cartão, que é estático).

function tempo(s: number) {
  if (s < 60) return "menos de 1 minuto";
  const m = Math.round(s / 60);
  return m === 1 ? "cerca de 1 minuto" : m < 120 ? `cerca de ${m} minutos` : "mais de 2 horas";
}

function Marca() {
  return (
    <div className="espera-marca">
      <SafetyCertificateFilled style={{ color: "var(--marker-400)", fontSize: 28 }} /> Lisura
    </div>
  );
}

// SalaDeEspera chama `tentar` sozinha, no ritmo que a fila anda. A vez está num cookie: recarregar
// a página ou fechar e abrir de novo não perde o lugar.
export function SalaDeEspera({ fila, tentar }: { fila: Fila; tentar: () => void }) {
  useEffect(() => {
    const t = setTimeout(tentar, Math.min(Math.max(fila.espera_segundos, 3), 15) * 1000);
    return () => clearTimeout(t);
  }, [fila, tentar]);

  return (
    <div className="espera">
      <Card className="espera-cartao">
        <Marca />
        <Typography.Title level={2} style={{ color: "var(--brand-800)", marginTop: 8 }}>
          Muita gente acessando agora
        </Typography.Title>
        <Typography.Paragraph type="secondary">
          Para ninguém perder a inscrição no meio do caminho, a entrada está sendo feita em ordem de chegada.
        </Typography.Paragraph>

        <div className="espera-numeros" role="status" aria-live="polite">
          <div>
            <span className="espera-rotulo">Sua posição</span>
            <span className="espera-valor">{fila.posicao.toLocaleString("pt-BR")}º</span>
          </div>
          <div>
            <span className="espera-rotulo">Tempo estimado</span>
            <span className="espera-valor espera-valor-menor">
              <ClockCircleOutlined /> {tempo(fila.espera_segundos)}
            </span>
          </div>
        </div>

        <ul className="espera-lista">
          <li>Quando chegar a sua vez, esta página continua sozinha — não precisa clicar em nada.</li>
          <li>Recarregar ou fechar e abrir de novo não faz você perder o lugar.</li>
          <li>A fila anda em ordem de chegada: ninguém passa na sua frente.</li>
        </ul>
      </Card>
    </div>
  );
}

// ModoConsulta é o nível 3: a API está fechada para proteger o que já foi feito, mas o cartão de
// confirmação é estático e abre mesmo assim. Pergunta a situação a cada 15 s e volta sozinho.
export function ModoConsulta({ aoVoltar }: { aoVoltar: () => void }) {
  const [cartoes] = useState<CartaoGuardado[]>(cartoesGuardados);
  useEffect(() => {
    const t = setInterval(async () => {
      const s = await fetch("/api/situacao").then((r) => r.json()).catch(() => null);
      if (s && s.nivel < 3) aoVoltar();
    }, 15000);
    return () => clearInterval(t);
  }, [aoVoltar]);

  return (
    <div className="espera">
      <Card className="espera-cartao">
        <Marca />
        <Typography.Title level={2} style={{ color: "var(--brand-800)", marginTop: 8 }}>
          Modo somente consulta
        </Typography.Title>
        <Typography.Paragraph type="secondary">
          Por alguns minutos, abrir ou alterar inscrições está pausado. Nada do que você já fez se perde: inscrições
          enviadas continuam registradas, e um Pix ou boleto que você pagar agora é confirmado normalmente.
        </Typography.Paragraph>

        {cartoes.length > 0 ? (
          <>
            <div className="rotulo-secao" style={{ marginLeft: 0 }}>Seus cartões de confirmação</div>
            {cartoes.map((c) => (
              <div key={c.inscricao} className="espera-item">
                <span>
                  <IdcardOutlined /> {c.certame} <span style={{ color: "var(--ink-600)" }}>· inscrição {c.inscricao}</span>
                </span>
                <Button href={c.url} target="_blank" icon={<ExportOutlined />} iconPlacement="end">
                  Abrir cartão
                </Button>
              </div>
            ))}
          </>
        ) : (
          <Typography.Paragraph>
            Os cartões que você abriu neste aparelho aparecem aqui. O seu cartão continua no endereço de sempre.
          </Typography.Paragraph>
        )}
        <Typography.Paragraph type="secondary" style={{ fontSize: 13, marginTop: 16, marginBottom: 0 }}>
          Esta página volta ao normal sozinha.
        </Typography.Paragraph>
      </Card>
    </div>
  );
}
