-- +goose Up

-- Dados pessoais vivem na conta, não na inscrição: são reaproveitados entre certames.
ALTER TABLE candidato
    ADD COLUMN data_nascimento date,
    ADD COLUMN municipio text,
    ADD COLUMN uf char(2) CHECK (uf ~ '^[A-Z]{2}$');

CREATE TABLE cidade_prova (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    certame_id  bigint NOT NULL REFERENCES certame (id),
    nome        text NOT NULL,
    uf          char(2) NOT NULL,
    UNIQUE (id, certame_id)
);

CREATE TABLE cota (
    certame_id   bigint NOT NULL REFERENCES certame (id),
    codigo       text NOT NULL,
    nome         text NOT NULL,
    exige_laudo  boolean NOT NULL DEFAULT false,
    PRIMARY KEY (certame_id, codigo)
);

-- Chaves compostas: o banco garante que cargo, cidade e cota pertencem ao certame da inscrição.
ALTER TABLE cargo ADD UNIQUE (id, certame_id);
ALTER TABLE inscricao
    ADD UNIQUE (id, certame_id),
    ADD UNIQUE (id, candidato_id),
    ADD COLUMN cidade_prova_id bigint,
    ADD COLUMN necessidades text[] NOT NULL DEFAULT '{}',
    ADD COLUMN declaracoes_aceitas_em timestamptz,
    ADD FOREIGN KEY (cargo_id, certame_id) REFERENCES cargo (id, certame_id),
    ADD FOREIGN KEY (cidade_prova_id, certame_id) REFERENCES cidade_prova (id, certame_id);

CREATE TABLE inscricao_cota (
    inscricao_id  bigint NOT NULL,
    certame_id    bigint NOT NULL,
    cota_codigo   text NOT NULL,
    PRIMARY KEY (inscricao_id, cota_codigo),
    FOREIGN KEY (inscricao_id, certame_id) REFERENCES inscricao (id, certame_id),
    FOREIGN KEY (certame_id, cota_codigo) REFERENCES cota (certame_id, codigo)
);

-- Carteira de documentos da conta. O arquivo fica no armazenamento de objetos; aqui só o registro.
CREATE TABLE documento (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    candidato_id  bigint NOT NULL REFERENCES candidato (id),
    tipo          text NOT NULL CHECK (tipo IN ('laudo', 'comprovante_isencao')),
    nome_arquivo  text NOT NULL,
    mime          text NOT NULL,
    tamanho       int NOT NULL CHECK (tamanho > 0),
    sha256        bytea NOT NULL,
    chave         text NOT NULL,
    enviado_em    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, candidato_id)
);
CREATE INDEX ON documento (candidato_id);

-- LGPD: o laudo não vai para a banca automaticamente. Cada inscrição registra o consentimento
-- de compartilhar um documento da própria conta; o banco recusa documento de outra pessoa.
CREATE TABLE inscricao_documento (
    inscricao_id   bigint NOT NULL,
    candidato_id   bigint NOT NULL,
    finalidade     text NOT NULL CHECK (finalidade IN ('laudo')),
    documento_id   bigint NOT NULL,
    consentido_em  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (inscricao_id, finalidade),
    FOREIGN KEY (inscricao_id, candidato_id) REFERENCES inscricao (id, candidato_id),
    FOREIGN KEY (documento_id, candidato_id) REFERENCES documento (id, candidato_id)
);

CREATE TABLE isencao (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    certame_id      bigint NOT NULL REFERENCES certame (id),
    candidato_id    bigint NOT NULL REFERENCES candidato (id),
    hipotese        text NOT NULL CHECK (hipotese IN ('cadunico', 'doador_medula')),
    nis             char(11),
    documento_id    bigint,
    estado          text NOT NULL CHECK (estado IN ('EM_ANALISE', 'DEFERIDA', 'INDEFERIDA')),
    motivo_codigo   text,
    decidida_por    text,
    edital_versao   int NOT NULL,
    solicitada_em   timestamptz NOT NULL DEFAULT now(),
    decidida_em     timestamptz,
    UNIQUE (certame_id, candidato_id),
    FOREIGN KEY (documento_id, candidato_id) REFERENCES documento (id, candidato_id),
    CHECK ((estado = 'INDEFERIDA') = (motivo_codigo IS NOT NULL))
);

-- +goose Down
DROP TABLE isencao;
DROP TABLE inscricao_documento;
DROP TABLE documento;
DROP TABLE inscricao_cota;
ALTER TABLE inscricao
    DROP COLUMN declaracoes_aceitas_em,
    DROP COLUMN necessidades,
    DROP COLUMN cidade_prova_id,
    DROP CONSTRAINT inscricao_id_candidato_id_key,
    DROP CONSTRAINT inscricao_id_certame_id_key;
ALTER TABLE inscricao DROP CONSTRAINT IF EXISTS inscricao_cargo_id_certame_id_fkey;
ALTER TABLE cargo DROP CONSTRAINT cargo_id_certame_id_key;
DROP TABLE cota;
DROP TABLE cidade_prova;
ALTER TABLE candidato DROP COLUMN uf, DROP COLUMN municipio, DROP COLUMN data_nascimento;
