-- 0008: fila de tarefas assíncronas no próprio Postgres (dívida D3).
--
-- Na AWS a fila é SQS; localmente é esta tabela, consumida com SKIP LOCKED. O
-- worker fala com a fila por uma interface, então a troca não mexe no domínio.
--
-- Entrega "pelo menos uma vez": tarefa travada por worker que morreu volta para a
-- fila quando a trava vence. Por isso toda tarefa precisa ser idempotente, e a
-- chave única (tipo, chave) impede enfileirar a mesma tarefa duas vezes.

CREATE TABLE tarefa (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tipo            text NOT NULL,
    chave           text NOT NULL,
    dados           jsonb NOT NULL DEFAULT '{}',
    estado          text NOT NULL DEFAULT 'PENDENTE'
                    CHECK (estado IN ('PENDENTE', 'EXECUTANDO', 'FEITA', 'FALHOU')),
    tentativas      integer NOT NULL DEFAULT 0,
    max_tentativas  integer NOT NULL DEFAULT 8,
    disponivel_em   timestamptz NOT NULL DEFAULT now(),
    travada_ate     timestamptz,
    ultimo_erro     text,
    criada_em       timestamptz NOT NULL DEFAULT now(),
    feita_em        timestamptz,
    UNIQUE (tipo, chave)
);

CREATE INDEX tarefa_pronta ON tarefa (disponivel_em) WHERE estado = 'PENDENTE';
CREATE INDEX tarefa_travada ON tarefa (travada_ate) WHERE estado = 'EXECUTANDO';

-- Pega até p_n tarefas prontas e as trava por p_trava_s segundos.
CREATE FUNCTION pegar_tarefas(p_tipos text[], p_n integer, p_trava_s integer)
RETURNS SETOF tarefa
LANGUAGE sql AS $$
    UPDATE tarefa t
       SET estado = 'EXECUTANDO',
           tentativas = t.tentativas + 1,
           travada_ate = now() + make_interval(secs => p_trava_s)
     WHERE t.id IN (
           SELECT id FROM tarefa
            WHERE tipo = ANY (p_tipos)
              AND ((estado = 'PENDENTE' AND disponivel_em <= now())
                   OR (estado = 'EXECUTANDO' AND travada_ate < now()))
            ORDER BY disponivel_em
            LIMIT p_n
            FOR UPDATE SKIP LOCKED)
    RETURNING t.*
$$;

CREATE FUNCTION concluir_tarefa(p_id bigint) RETURNS void
LANGUAGE sql AS $$
    UPDATE tarefa SET estado = 'FEITA', feita_em = now(), travada_ate = NULL
     WHERE id = p_id AND estado = 'EXECUTANDO'
$$;

-- Falha com recuo exponencial (2, 4, 8... segundos, até 5 min). Esgotadas as
-- tentativas, a tarefa fica FALHOU e aparece para alguém olhar.
CREATE FUNCTION falhar_tarefa(p_id bigint, p_erro text) RETURNS void
LANGUAGE sql AS $$
    UPDATE tarefa
       SET estado = CASE WHEN tentativas >= max_tentativas THEN 'FALHOU' ELSE 'PENDENTE' END,
           disponivel_em = now() + make_interval(secs => least(300, power(2, tentativas))::int),
           travada_ate = NULL,
           ultimo_erro = left(p_erro, 2000)
     WHERE id = p_id AND estado = 'EXECUTANDO'
$$;

CREATE VIEW fila_resumo AS
SELECT tipo, estado, count(*) AS quantidade,
       min(criada_em) FILTER (WHERE estado = 'PENDENTE') AS mais_antiga_pendente
  FROM tarefa GROUP BY tipo, estado;

-- O que o provedor devolve ao registrar a cobrança (QR do Pix, linha digitável).
-- Preenchido pelo worker; o candidato consulta depois, sem esperar no request.
ALTER TABLE cobranca
    ADD COLUMN instrucoes jsonb,
    ADD COLUMN registrada_em timestamptz;
