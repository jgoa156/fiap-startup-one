-- 0001: banca, certame, edital versionado, cargos, cidades e cotas.
-- O edital é versionado porque retificação é rotina: uma decisão tomada sob a
-- versão 2 tem de continuar apontando para a versão 2.

CREATE TABLE banca (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nome        text NOT NULL,
    criada_em   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE certame (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    banca_id            bigint NOT NULL REFERENCES banca(id),
    nome                text NOT NULL,
    taxa_centavos       integer NOT NULL CHECK (taxa_centavos >= 0),
    isencao_inicio      timestamptz,
    isencao_fim         timestamptz,
    inscricao_inicio    timestamptz NOT NULL,
    inscricao_fim       timestamptz NOT NULL,
    criado_em           timestamptz NOT NULL DEFAULT now(),
    CHECK (inscricao_fim > inscricao_inicio),
    CHECK (isencao_inicio IS NULL OR isencao_fim > isencao_inicio)
);

CREATE TABLE edital_versao (
    certame_id      bigint NOT NULL REFERENCES certame(id),
    versao          integer NOT NULL CHECK (versao >= 1),
    publicada_em    timestamptz NOT NULL DEFAULT now(),
    descricao       text NOT NULL,
    PRIMARY KEY (certame_id, versao)
);

-- Versão vigente: a maior já publicada. Usada pela trilha de auditoria.
CREATE FUNCTION edital_versao_vigente(p_certame_id bigint) RETURNS integer
LANGUAGE sql STABLE AS $$
    SELECT max(versao) FROM edital_versao
     WHERE certame_id = p_certame_id AND publicada_em <= now()
$$;

CREATE TABLE cargo (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    certame_id  bigint NOT NULL REFERENCES certame(id),
    nome        text NOT NULL,
    vagas       integer NOT NULL CHECK (vagas >= 0),
    UNIQUE (certame_id, id)
);

CREATE TABLE cidade_prova (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    certame_id  bigint NOT NULL REFERENCES certame(id),
    nome        text NOT NULL,
    UNIQUE (certame_id, id)
);

-- Modalidade de concorrência configurada por certame (ampla, PcD, cotas raciais...).
CREATE TABLE modalidade (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    certame_id  bigint NOT NULL REFERENCES certame(id),
    codigo      text NOT NULL,
    nome        text NOT NULL,
    UNIQUE (certame_id, codigo),
    UNIQUE (certame_id, id)
);
