-- +goose Up

-- Prestação de contas (02-produto §C). O dinheiro não passa por nós: cai na conta da banca, no
-- prestador dela. O que devemos à banca e ao órgão é a conta exata do que entrou e do que saiu, e a
-- garantia de que um dia fechado não muda depois.
--
-- O dia de um pagamento é o dia em que o sistema o registrou (recebido_em), não o dia em que o
-- candidato pagou: o boleto pago na sexta e informado na terça entra na terça, como num extrato
-- bancário pela data de crédito. Assim, um dia fechado só muda por defeito ou adulteração. A
-- devolução entra no dia em que o prestador a concluiu.
CREATE TABLE arrecadacao_fechamento (
    certame_id          bigint NOT NULL REFERENCES certame (id),
    dia                 date NOT NULL,
    pagamentos          int NOT NULL CHECK (pagamentos >= 0),
    entrou_centavos     bigint NOT NULL CHECK (entrou_centavos >= 0),
    devolucoes          int NOT NULL CHECK (devolucoes >= 0),
    saiu_centavos       bigint NOT NULL CHECK (saiu_centavos >= 0),
    -- SHA-256 dos ids e valores do dia, em ordem: muda se qualquer pagamento ou devolução do dia
    -- mudar, sumir ou aparecer, mesmo que os totais continuem batendo.
    resumo              char(64) NOT NULL,
    fechado_em          timestamptz NOT NULL DEFAULT now(),
    fechado_por         text NOT NULL,
    PRIMARY KEY (certame_id, dia)
);

CREATE TRIGGER arrecadacao_fechamento_imutavel BEFORE UPDATE OR DELETE ON arrecadacao_fechamento
    FOR EACH ROW EXECUTE FUNCTION trilha_imutavel();

-- +goose Down
DROP TABLE arrecadacao_fechamento;
