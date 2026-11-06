-- +goose Up

-- O dinheiro que entrou sem dever volta, um pagamento por vez. Até aqui o sistema só marcava a
-- cobrança (reembolso_pendente) e ninguém devolvia nada.
--
-- Uma linha por PAGAMENTO, não por cobrança: o boleto pago três vezes tem dois pagamentos a devolver.
-- A chave única em psp_pagamento_id é o espelho de "nunca cobrar duas vezes": nenhum pagamento é
-- devolvido duas vezes, venha a ordem de quantos lugares vier.
CREATE TABLE devolucao (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    psp_pagamento_id  text NOT NULL UNIQUE REFERENCES pagamento_recebido (psp_pagamento_id),
    cobranca_id       bigint NOT NULL REFERENCES cobranca (id),
    inscricao_id      bigint NOT NULL REFERENCES inscricao (id),
    valor_centavos    int NOT NULL CHECK (valor_centavos > 0),
    motivo            text NOT NULL CHECK (motivo IN ('PAGO_EM_DUPLICIDADE', 'PAGO_APOS_EXPIRACAO')),
    -- Pix volta para quem pagou, pelo próprio pagamento. Boleto não diz quem pagou: volta por Pix
    -- para a chave CPF do candidato, e o prestador confere que o titular é esse CPF.
    metodo_origem     text NOT NULL CHECK (metodo_origem IN ('pix', 'boleto')),
    chave_pix         text,
    estado            text NOT NULL DEFAULT 'PENDENTE' CHECK (estado IN (
                          'PENDENTE',          -- a fila vai pedir ao prestador
                          'AGUARDANDO_CHAVE',  -- boleto, e o CPF do candidato não é chave Pix
                          'CONCLUIDA',
                          'RECUSADA')),        -- o prestador recusou de vez: alguém precisa olhar
    psp_devolucao_id  text UNIQUE,
    ultimo_erro       text,
    criada_em         timestamptz NOT NULL DEFAULT now(),
    concluida_em      timestamptz,
    CHECK ((estado = 'CONCLUIDA') = (psp_devolucao_id IS NOT NULL AND concluida_em IS NOT NULL))
);
CREATE INDEX devolucao_inscricao ON devolucao (inscricao_id);
CREATE INDEX devolucao_a_retomar ON devolucao (criada_em) WHERE estado = 'PENDENTE';

-- O que já entrou a mais antes desta migração também volta: a rotina de pagamentos acha estas linhas
-- PENDENTE sem tarefa e as enfileira.
INSERT INTO devolucao (psp_pagamento_id, cobranca_id, inscricao_id, valor_centavos, motivo, metodo_origem)
SELECT p.psp_pagamento_id, b.id, b.inscricao_id, p.valor_centavos, p.resultado, b.metodo
FROM pagamento_recebido p JOIN cobranca b ON b.id = p.cobranca_id
WHERE p.resultado IN ('PAGO_EM_DUPLICIDADE', 'PAGO_APOS_EXPIRACAO');

-- +goose Down
DROP TABLE devolucao;
