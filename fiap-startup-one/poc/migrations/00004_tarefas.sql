-- +goose Up

-- Fila de tarefas assíncronas. Na arquitetura alvo é SQS + Lambda; na POC, uma tabela com
-- FOR UPDATE SKIP LOCKED, que dá a mesma garantia: cada tarefa é pega por um trabalhador só.
-- Enfileirar na mesma transação da mudança de estado é o que impede tarefa perdida ("outbox").
CREATE TABLE tarefa (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tipo          text NOT NULL,
    -- Chave de deduplicação: a mesma tarefa enfileirada duas vezes vira uma só.
    chave         text NOT NULL,
    carga         jsonb NOT NULL DEFAULT '{}',
    estado        text NOT NULL DEFAULT 'PENDENTE' CHECK (estado IN ('PENDENTE', 'CONCLUIDA', 'FALHOU')),
    tentativas    int NOT NULL DEFAULT 0,
    -- Sobe a cada reenfileiramento. O trabalhador só conclui a versão que pegou: se a tarefa foi
    -- reenfileirada enquanto ele trabalhava (o cartão mudou de novo), ela roda outra vez.
    versao        int NOT NULL DEFAULT 1,
    executar_em   timestamptz NOT NULL DEFAULT now(),
    ultimo_erro   text,
    criada_em     timestamptz NOT NULL DEFAULT now(),
    concluida_em  timestamptz,
    UNIQUE (tipo, chave)
);
CREATE INDEX tarefa_a_executar ON tarefa (executar_em) WHERE estado = 'PENDENTE';

-- +goose Down
DROP TABLE tarefa;
