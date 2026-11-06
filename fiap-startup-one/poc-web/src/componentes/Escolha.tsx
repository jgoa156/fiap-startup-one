"use client";

import { CheckCircleFilled } from "@ant-design/icons";
import "./escolha.css";

// Cartões de escolha (Inclusa, doc 01 §2): no lugar de radio, checkbox e select, o candidato vê
// todas as opções como cartões. Selecionado = borda na cor de ação + fundo de seleção + ✓ —
// nunca só a cor. Teclado e leitor de tela: botões com aria-pressed dentro de um grupo rotulado.

export type Opcao<T extends string | number> = {
  valor: T;
  titulo: React.ReactNode;
  detalhe?: React.ReactNode;
  icone?: React.ReactNode;
  desabilitada?: boolean;
};

type Props<T extends string | number> = {
  rotulo: string;
  opcoes: Opcao<T>[];
  colunas?: number;
  compacto?: boolean; // chips em pílula, para listas de várias escolhas
} & (
  | { multipla?: false; valor: T | null; aoMudar: (v: T) => void }
  | { multipla: true; valor: T[]; aoMudar: (v: T[]) => void }
);

export function Escolha<T extends string | number>(props: Props<T>) {
  const { rotulo, opcoes, colunas = 3, compacto } = props;
  const marcada = (v: T) => (props.multipla ? props.valor.includes(v) : props.valor === v);
  const clicar = (v: T) => {
    if (props.multipla) {
      props.aoMudar(props.valor.includes(v) ? props.valor.filter((x) => x !== v) : [...props.valor, v]);
    } else {
      props.aoMudar(v);
    }
  };

  return (
    <div
      role="group"
      aria-label={rotulo}
      className={compacto ? "escolha escolha-chips" : "escolha"}
      style={compacto ? undefined : { gridTemplateColumns: `repeat(auto-fill, minmax(${Math.floor(720 / colunas)}px, 1fr))` }}
    >
      {opcoes.map((o) => {
        const sel = marcada(o.valor);
        return (
          <button
            key={String(o.valor)}
            type="button"
            className={sel ? "escolha-item marcada" : "escolha-item"}
            aria-pressed={sel}
            disabled={o.desabilitada}
            onClick={() => clicar(o.valor)}
          >
            {sel ? <CheckCircleFilled className="escolha-check" /> : o.icone && <span className="escolha-icone">{o.icone}</span>}
            <span className="escolha-texto">
              <span className="escolha-titulo">{o.titulo}</span>
              {o.detalhe && <span className="escolha-detalhe">{o.detalhe}</span>}
            </span>
          </button>
        );
      })}
    </div>
  );
}
