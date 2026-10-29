-- 0002: candidato, inscrição, máquina de estados e trilha de auditoria.
--
-- As regras da máquina de estados ficam no banco, e não só na aplicação, porque
-- concurso é judicializado por padrão: nenhum caminho de código, script de
-- correção ou acesso manual pode mudar o estado de uma inscrição sem deixar
-- quem, quando, por qual motivo e sob qual versão do edital.
--
-- Códigos de erro próprios (SQLSTATE classe CT):
--   CT001  transição de estado não permitida
--   CT002  transição sem ator ou sem motivo
--   CT003  tentativa de alterar ou apagar a trilha
--   CT004  alteração de dados fora do RASCUNHO, ou de certame/candidato
--   CT005  certame sem edital publicado

CREATE TABLE candidato (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cpf         char(11) NOT NULL UNIQUE CHECK (cpf ~ '^[0-9]{11}$'),
    nome        text NOT NULL,
    email       text NOT NULL,
    criado_em   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE estado_inscricao (
    codigo  text PRIMARY KEY,
    final   boolean NOT NULL
);

INSERT INTO estado_inscricao (codigo, final) VALUES
    ('RASCUNHO',             false),
    ('ISENCAO_ATIVA',        false),
    ('AGUARDANDO_PAGAMENTO', false),
    ('PAGA',                 false),
    ('EXPIRADA',             true),
    ('CANCELADA',            true),
    ('EM_ANALISE',           false),
    ('DEFERIDA',             false),
    ('INDEFERIDA',           false),
    ('HOMOLOGADA',           true);

-- Fonte: 02-produto-features-e-fluxos.md §5. O teste
-- test_tabela_do_banco_igual_a_do_codigo garante que esta lista e a do Python são iguais.
CREATE TABLE transicao_permitida (
    de      text NOT NULL REFERENCES estado_inscricao(codigo),
    para    text NOT NULL REFERENCES estado_inscricao(codigo),
    evento  text NOT NULL,
    PRIMARY KEY (de, para)
);

INSERT INTO transicao_permitida (de, para, evento) VALUES
    ('RASCUNHO',             'ISENCAO_ATIVA',        'finaliza_isento'),
    ('RASCUNHO',             'AGUARDANDO_PAGAMENTO', 'finaliza_pagante'),
    ('RASCUNHO',             'CANCELADA',            'cancela'),
    ('AGUARDANDO_PAGAMENTO', 'PAGA',                 'confirma_pagamento'),
    ('AGUARDANDO_PAGAMENTO', 'EXPIRADA',             'vence_prazo'),
    ('ISENCAO_ATIVA',        'EM_ANALISE',           'envia_para_analise'),
    ('PAGA',                 'EM_ANALISE',           'envia_para_analise'),
    ('EM_ANALISE',           'DEFERIDA',             'defere'),
    ('EM_ANALISE',           'INDEFERIDA',           'indefere'),
    ('DEFERIDA',             'HOMOLOGADA',           'homologa'),
    ('INDEFERIDA',           'HOMOLOGADA',           'provê_recurso');

CREATE TABLE inscricao (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    certame_id      bigint NOT NULL REFERENCES certame(id),
    candidato_id    bigint NOT NULL REFERENCES candidato(id),
    cargo_id        bigint,
    cidade_id       bigint,
    modalidade_id   bigint,
    estado          text NOT NULL DEFAULT 'RASCUNHO' REFERENCES estado_inscricao(codigo),
    criada_em       timestamptz NOT NULL DEFAULT now(),
    atualizada_em   timestamptz NOT NULL DEFAULT now(),
    -- Uma inscrição por candidato por certame.
    UNIQUE (certame_id, candidato_id),
    -- Cargo, cidade e modalidade precisam ser do mesmo certame.
    FOREIGN KEY (certame_id, cargo_id)      REFERENCES cargo(certame_id, id),
    FOREIGN KEY (certame_id, cidade_id)     REFERENCES cidade_prova(certame_id, id),
    FOREIGN KEY (certame_id, modalidade_id) REFERENCES modalidade(certame_id, id)
);

CREATE INDEX inscricao_certame_estado ON inscricao (certame_id, estado);

-- A trilha. Só aceita INSERT, e só pelos triggers abaixo na prática.
CREATE TABLE inscricao_transicao (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    inscricao_id    bigint NOT NULL REFERENCES inscricao(id),
    de              text REFERENCES estado_inscricao(codigo),   -- NULL na criação
    para            text NOT NULL REFERENCES estado_inscricao(codigo),
    ator            text NOT NULL CHECK (length(ator) > 0),
    motivo          text NOT NULL CHECK (length(motivo) > 0),
    versao_edital   integer NOT NULL,
    em              timestamptz NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX inscricao_transicao_inscricao ON inscricao_transicao (inscricao_id, id);

-- Contexto de auditoria: a aplicação define, dentro da transação,
--   SELECT set_config('certamis.ator', '...', true), set_config('certamis.motivo', '...', true);
CREATE FUNCTION _auditoria_contexto(OUT ator text, OUT motivo text)
LANGUAGE plpgsql STABLE AS $$
BEGIN
    ator   := nullif(current_setting('certamis.ator', true), '');
    motivo := nullif(current_setting('certamis.motivo', true), '');
    IF ator IS NULL OR motivo IS NULL THEN
        RAISE EXCEPTION 'transição de inscrição exige ator e motivo'
            USING ERRCODE = 'CT002';
    END IF;
END $$;

CREATE FUNCTION _inscricao_antes() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.estado <> 'RASCUNHO' THEN
            RAISE EXCEPTION 'inscrição nasce em RASCUNHO, não em %', NEW.estado
                USING ERRCODE = 'CT001';
        END IF;
        IF edital_versao_vigente(NEW.certame_id) IS NULL THEN
            RAISE EXCEPTION 'certame % não tem edital publicado', NEW.certame_id
                USING ERRCODE = 'CT005';
        END IF;
        PERFORM _auditoria_contexto();
        RETURN NEW;
    END IF;

    -- UPDATE
    IF NEW.certame_id <> OLD.certame_id OR NEW.candidato_id <> OLD.candidato_id
       OR NEW.criada_em <> OLD.criada_em THEN
        RAISE EXCEPTION 'certame, candidato e data de criação não mudam'
            USING ERRCODE = 'CT004';
    END IF;

    IF (NEW.cargo_id, NEW.cidade_id, NEW.modalidade_id)
       IS DISTINCT FROM (OLD.cargo_id, OLD.cidade_id, OLD.modalidade_id)
       AND OLD.estado <> 'RASCUNHO' THEN
        RAISE EXCEPTION 'dados da inscrição só mudam em RASCUNHO (estado atual: %)', OLD.estado
            USING ERRCODE = 'CT004';
    END IF;

    IF NEW.estado IS DISTINCT FROM OLD.estado THEN
        IF NOT EXISTS (SELECT 1 FROM transicao_permitida
                        WHERE de = OLD.estado AND para = NEW.estado) THEN
            RAISE EXCEPTION 'transição % -> % não é permitida', OLD.estado, NEW.estado
                USING ERRCODE = 'CT001';
        END IF;
        PERFORM _auditoria_contexto();
    END IF;

    NEW.atualizada_em := now();
    RETURN NEW;
END $$;

CREATE FUNCTION _inscricao_depois() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    ctx record;
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.estado IS NOT DISTINCT FROM OLD.estado THEN
        RETURN NULL;
    END IF;
    ctx := _auditoria_contexto();
    INSERT INTO inscricao_transicao (inscricao_id, de, para, ator, motivo, versao_edital)
    VALUES (NEW.id,
            CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE OLD.estado END,
            NEW.estado, ctx.ator, ctx.motivo,
            edital_versao_vigente(NEW.certame_id));
    RETURN NULL;
END $$;

CREATE TRIGGER inscricao_antes
    BEFORE INSERT OR UPDATE ON inscricao
    FOR EACH ROW EXECUTE FUNCTION _inscricao_antes();

CREATE TRIGGER inscricao_depois
    AFTER INSERT OR UPDATE ON inscricao
    FOR EACH ROW EXECUTE FUNCTION _inscricao_depois();

CREATE FUNCTION _trilha_imutavel() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'a trilha de auditoria não pode ser alterada nem apagada'
        USING ERRCODE = 'CT003';
END $$;

CREATE TRIGGER trilha_sem_update_delete
    BEFORE UPDATE OR DELETE ON inscricao_transicao
    FOR EACH ROW EXECUTE FUNCTION _trilha_imutavel();

CREATE TRIGGER trilha_sem_truncate
    BEFORE TRUNCATE ON inscricao_transicao
    FOR EACH STATEMENT EXECUTE FUNCTION _trilha_imutavel();

-- Inscrição também não se apaga: cancelar é uma transição.
CREATE FUNCTION _inscricao_sem_delete() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'inscrição não se apaga; use a transição para CANCELADA'
        USING ERRCODE = 'CT003';
END $$;

CREATE TRIGGER inscricao_sem_delete
    BEFORE DELETE ON inscricao
    FOR EACH ROW EXECUTE FUNCTION _inscricao_sem_delete();
