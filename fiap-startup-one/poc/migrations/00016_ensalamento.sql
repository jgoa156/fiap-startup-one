-- +goose Up

-- Ensalamento simples por capacidade (02-produto §7 F e §8): cada inscrição homologada ganha uma sala
-- na cidade de prova que escolheu, respeitando a capacidade e o atendimento especial.
CREATE TABLE local_prova (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    certame_id       bigint NOT NULL,
    cidade_prova_id  bigint NOT NULL,
    nome             text NOT NULL,
    endereco         text NOT NULL,
    amamentacao      boolean NOT NULL DEFAULT false, -- tem espaço para a lactante amamentar
    UNIQUE (certame_id, cidade_prova_id, nome),
    FOREIGN KEY (cidade_prova_id, certame_id) REFERENCES cidade_prova (id, certame_id)
);

CREATE TABLE sala (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    local_id    bigint NOT NULL REFERENCES local_prova (id),
    nome        text NOT NULL,
    capacidade  int NOT NULL CHECK (capacidade > 0),
    -- comum; especial (tempo adicional, prova ampliada: o grupo fica mais tempo, com fiscal próprio);
    -- individual (ledor, transcritor: um candidato e quem o atende).
    tipo        text NOT NULL CHECK (tipo IN ('comum', 'especial', 'individual')),
    acessivel   boolean NOT NULL DEFAULT false, -- térreo ou com rampa e elevador
    UNIQUE (local_id, nome),
    CHECK (tipo <> 'individual' OR capacidade = 1)
);

-- A sala de cada inscrição, agora. A chave primária na inscrição: ninguém tem duas salas.
CREATE TABLE alocacao (
    inscricao_id  bigint PRIMARY KEY REFERENCES inscricao (id),
    certame_id    bigint NOT NULL REFERENCES certame (id),
    sala_id       bigint NOT NULL REFERENCES sala (id),
    versao        int NOT NULL,   -- a versão do ensalamento em que a inscrição ganhou esta sala
    alocada_em    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX alocacao_por_sala ON alocacao (sala_id);
CREATE INDEX alocacao_por_certame ON alocacao (certame_id);

-- A última linha de defesa: sala lotada recusa mais um, venha de onde vier. A trava na sala serializa
-- duas alocações simultâneas na mesma sala.
-- +goose StatementBegin
CREATE FUNCTION sala_com_lugar() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    cap int;
    ocupados int;
BEGIN
    SELECT capacidade INTO cap FROM sala WHERE id = NEW.sala_id FOR UPDATE;
    SELECT count(*) INTO ocupados FROM alocacao WHERE sala_id = NEW.sala_id AND inscricao_id <> NEW.inscricao_id;
    IF ocupados >= cap THEN
        RAISE EXCEPTION 'sala % lotada (capacidade %)', NEW.sala_id, cap USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER alocacao_sala_com_lugar BEFORE INSERT OR UPDATE OF sala_id ON alocacao
    FOR EACH ROW EXECUTE FUNCTION sala_com_lugar();

-- Cada ensalamento é um ato publicado: quantos, o que mudou e o SHA-256 da alocação inteira. Não muda
-- depois (o mesmo gatilho da trilha); a alocação atual tem de bater com o resumo da última versão.
CREATE TABLE ensalamento_versao (
    certame_id  bigint NOT NULL REFERENCES certame (id),
    versao      int NOT NULL CHECK (versao > 0),
    alocadas    int NOT NULL,
    mantidas    int NOT NULL,
    novas       int NOT NULL,
    removidas   int NOT NULL,
    resumo      char(64) NOT NULL,
    motivo      text NOT NULL,
    criado_por  text NOT NULL,
    criado_em   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (certame_id, versao)
);
CREATE TRIGGER ensalamento_versao_imutavel BEFORE UPDATE OR DELETE ON ensalamento_versao
    FOR EACH ROW EXECUTE FUNCTION trilha_imutavel();

-- +goose Down
DROP TABLE ensalamento_versao;
DROP TABLE alocacao;
DROP FUNCTION sala_com_lugar();
DROP TABLE sala;
DROP TABLE local_prova;
