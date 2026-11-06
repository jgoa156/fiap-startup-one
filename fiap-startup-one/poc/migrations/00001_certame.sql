-- +goose Up

-- Inquilino: a banca organizadora.
CREATE TABLE banca (
    id    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nome  text NOT NULL,
    cnpj  char(14) NOT NULL UNIQUE
);

CREATE TABLE certame (
    id        bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    banca_id  bigint NOT NULL REFERENCES banca (id),
    nome      text NOT NULL
);
CREATE INDEX ON certame (banca_id);

-- O edital é versionado e só cresce: retificação publica uma versão nova, nunca altera a antiga.
-- Toda decisão registra a versão em vigor no momento em que foi tomada.
CREATE TABLE edital_versao (
    certame_id    bigint NOT NULL REFERENCES certame (id),
    versao        int NOT NULL CHECK (versao > 0),
    publicado_em  timestamptz NOT NULL,
    resumo        text NOT NULL,
    PRIMARY KEY (certame_id, versao)
);

-- O cronograma é dado de entrada da infraestrutura (escalonamento por agenda).
CREATE TABLE certame_fase (
    certame_id  bigint NOT NULL REFERENCES certame (id),
    fase        text NOT NULL CHECK (fase IN (
                    'isencao', 'inscricao', 'homologacao', 'ensalamento',
                    'aplicacao', 'gabarito', 'resultado')),
    abre_em     timestamptz NOT NULL,
    fecha_em    timestamptz NOT NULL,
    PRIMARY KEY (certame_id, fase),
    CHECK (fecha_em > abre_em)
);

CREATE TABLE cargo (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    certame_id      bigint NOT NULL REFERENCES certame (id),
    nome            text NOT NULL,
    vagas           int NOT NULL CHECK (vagas > 0),
    taxa_centavos   int NOT NULL CHECK (taxa_centavos >= 0)
);
CREATE INDEX ON cargo (certame_id);

-- Conta única por CPF: atravessa bancas e certames, por isso não tem banca_id.
CREATE TABLE candidato (
    id     bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cpf    char(11) NOT NULL UNIQUE CHECK (cpf ~ '^[0-9]{11}$'),
    nome   text NOT NULL,
    email  text NOT NULL
);

CREATE TABLE inscricao (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    certame_id     bigint NOT NULL REFERENCES certame (id),
    candidato_id   bigint NOT NULL REFERENCES candidato (id),
    cargo_id       bigint NOT NULL REFERENCES cargo (id),
    estado         text NOT NULL DEFAULT 'RASCUNHO' CHECK (estado IN (
                       'RASCUNHO', 'ISENCAO_ATIVA', 'AGUARDANDO_PAGAMENTO', 'PAGA',
                       'EXPIRADA', 'CANCELADA', 'EM_ANALISE', 'DEFERIDA', 'INDEFERIDA',
                       'HOMOLOGADA')),
    criada_em      timestamptz NOT NULL DEFAULT now(),
    atualizada_em  timestamptz NOT NULL DEFAULT now(),
    -- Uma inscrição por candidato por certame: o reenvio sob carga não duplica.
    UNIQUE (certame_id, candidato_id)
);
CREATE INDEX ON inscricao (candidato_id);
CREATE INDEX ON inscricao (certame_id, estado);

-- Trilha de auditoria: quem, quando, por qual motivo e sob qual versão do edital.
CREATE TABLE inscricao_evento (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    inscricao_id    bigint NOT NULL REFERENCES inscricao (id),
    evento          text NOT NULL,
    de_estado       text NOT NULL,
    para_estado     text NOT NULL,
    ator_tipo       text NOT NULL CHECK (ator_tipo IN ('candidato', 'analista', 'sistema')),
    ator_id         text NOT NULL,
    motivo_codigo   text,
    edital_versao   int NOT NULL,
    ocorrido_em     timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX ON inscricao_evento (inscricao_id, id);

-- +goose StatementBegin
CREATE FUNCTION trilha_imutavel() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'a trilha de auditoria é imutável (% em %)', TG_OP, TG_TABLE_NAME
        USING ERRCODE = 'insufficient_privilege';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER inscricao_evento_imutavel
    BEFORE UPDATE OR DELETE ON inscricao_evento
    FOR EACH ROW EXECUTE FUNCTION trilha_imutavel();
CREATE TRIGGER inscricao_evento_sem_truncate
    BEFORE TRUNCATE ON inscricao_evento
    FOR EACH STATEMENT EXECUTE FUNCTION trilha_imutavel();

-- +goose Down
DROP TABLE inscricao_evento;
DROP FUNCTION trilha_imutavel();
DROP TABLE inscricao;
DROP TABLE candidato;
DROP TABLE cargo;
DROP TABLE certame_fase;
DROP TABLE edital_versao;
DROP TABLE certame;
DROP TABLE banca;
