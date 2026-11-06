-- +goose Up

-- O edital tem prazo de pagamento próprio, normalmente alguns dias depois do fim da inscrição.
ALTER TABLE certame_fase DROP CONSTRAINT certame_fase_fase_check;
ALTER TABLE certame_fase ADD CONSTRAINT certame_fase_fase_check CHECK (fase IN (
    'isencao', 'inscricao', 'pagamento', 'homologacao', 'ensalamento',
    'aplicacao', 'gabarito', 'resultado'));

-- Orquestramos o pagamento, não o processamos: a taxa é receita pública e cai na conta do órgão
-- ou da banca. Aqui fica só o registro da cobrança feita no PSP.
CREATE TABLE cobranca (
    id                   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    inscricao_id         bigint NOT NULL REFERENCES inscricao (id),
    metodo               text NOT NULL CHECK (metodo IN ('pix', 'boleto')),
    valor_centavos       int NOT NULL CHECK (valor_centavos > 0),
    estado               text NOT NULL DEFAULT 'PENDENTE' CHECK (estado IN (
                             'PENDENTE', 'PAGA', 'EXPIRADA', 'PAGA_EM_DUPLICIDADE', 'PAGA_APOS_EXPIRACAO')),
    psp_id               text UNIQUE,
    pix_copia_cola       text,
    linha_digitavel      text,
    vence_em             timestamptz NOT NULL,
    criada_em            timestamptz NOT NULL DEFAULT now(),
    paga_em              timestamptz,
    valor_pago_centavos  int,
    -- Dinheiro que entrou sem dever: pagamento em dobro ou depois de expirada. Precisa voltar.
    reembolso_pendente   boolean NOT NULL DEFAULT false
);
-- Duplo clique em "pagar com Pix" não cria duas cobranças.
CREATE UNIQUE INDEX cobranca_uma_pendente_por_metodo ON cobranca (inscricao_id, metodo) WHERE estado = 'PENDENTE';
-- A última linha de defesa: uma inscrição nunca tem dois pagamentos aceitos.
CREATE UNIQUE INDEX cobranca_uma_paga_por_inscricao ON cobranca (inscricao_id) WHERE estado = 'PAGA';
CREATE INDEX ON cobranca (estado, vence_em);

-- Cada pagamento informado pelo PSP é processado uma vez só, venha pelo aviso (webhook) ou pela
-- conciliação, quantas vezes chegar.
CREATE TABLE pagamento_recebido (
    psp_pagamento_id  text PRIMARY KEY,
    cobranca_id       bigint REFERENCES cobranca (id),
    valor_centavos    int NOT NULL,
    pago_em           timestamptz NOT NULL,
    origem            text NOT NULL CHECK (origem IN ('webhook', 'conciliacao')),
    resultado         text NOT NULL,
    recebido_em       timestamptz NOT NULL DEFAULT now()
);

-- O que a conciliação encontrou fora do lugar, para alguém olhar.
CREATE TABLE divergencia (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tipo              text NOT NULL CHECK (tipo IN (
                          'VALOR_DIVERGENTE', 'COBRANCA_DESCONHECIDA', 'PAGA_SEM_REGISTRO_NO_PSP')),
    psp_pagamento_id  text NOT NULL,
    cobranca_id       bigint REFERENCES cobranca (id),
    detalhe           text NOT NULL,
    detectada_em      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tipo, psp_pagamento_id)
);

-- +goose Down
DROP TABLE divergencia;
DROP TABLE pagamento_recebido;
DROP TABLE cobranca;
ALTER TABLE certame_fase DROP CONSTRAINT certame_fase_fase_check;
ALTER TABLE certame_fase ADD CONSTRAINT certame_fase_fase_check CHECK (fase IN (
    'isencao', 'inscricao', 'homologacao', 'ensalamento', 'aplicacao', 'gabarito', 'resultado'));
