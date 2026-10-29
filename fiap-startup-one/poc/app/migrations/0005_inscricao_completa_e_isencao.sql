-- 0005: dados completos da inscrição, documentos, prazos e isenção com triagem por regra.
--
-- Códigos de erro novos:
--   CT006  inscrição incompleta para finalizar
--   CT007  fora do prazo (inscrição ou isenção)
--   CT008  isenção incoerente com o caminho (isento sem isenção deferida, ou pagante com ela)
--   CT009  transição de pedido de isenção não permitida, ou sem ator e motivo

-- ---------------------------------------------------------------- inscrição

ALTER TABLE inscricao
    ADD COLUMN atendimento_especial boolean NOT NULL DEFAULT false,
    ADD COLUMN declaracoes_aceitas_em timestamptz;

CREATE FUNCTION inscricao_aberta(p_certame_id bigint) RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT now() BETWEEN inscricao_inicio AND inscricao_fim FROM certame WHERE id = p_certame_id
$$;

CREATE FUNCTION isencao_aberta(p_certame_id bigint) RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT coalesce(now() BETWEEN isencao_inicio AND isencao_fim, false)
      FROM certame WHERE id = p_certame_id
$$;

-- ---------------------------------------------------------------- documentos
-- O arquivo fica no armazenamento (disco local na POC, S3 depois). Aqui fica o
-- metadado e o hash, que é o que prova anos depois qual arquivo foi analisado.
-- Laudo é dado sensível e pertence a UMA inscrição: não vaza de um certame para
-- outro sem novo envio (02-produto §3).

CREATE TABLE documento (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    candidato_id    bigint NOT NULL REFERENCES candidato(id),
    inscricao_id    bigint REFERENCES inscricao(id),
    tipo            text NOT NULL CHECK (tipo IN ('LAUDO', 'COMPROVANTE_ISENCAO')),
    nome_original   text NOT NULL,
    tamanho_bytes   integer NOT NULL CHECK (tamanho_bytes BETWEEN 1 AND 10485760),
    sha256          char(64) NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    -- A chave do armazenamento deriva do hash e pode se repetir: o mesmo arquivo enviado
    -- duas vezes ocupa um lugar só, mas cada envio é um documento próprio, do seu dono.
    chave           text NOT NULL,
    criado_em       timestamptz NOT NULL DEFAULT now(),
    CHECK (tipo <> 'LAUDO' OR inscricao_id IS NOT NULL)
);

CREATE INDEX documento_inscricao ON documento (inscricao_id) WHERE inscricao_id IS NOT NULL;

-- ---------------------------------------------------------------- isenção

-- Hipóteses legais aceitas por certame, e o que cada uma exige (02-produto §4).
CREATE TABLE hipotese_isencao (
    certame_id          bigint NOT NULL REFERENCES certame(id),
    codigo              text NOT NULL,
    nome                text NOT NULL,
    exige_nis           boolean NOT NULL,
    exige_comprovante   boolean NOT NULL,
    PRIMARY KEY (certame_id, codigo)
);

-- Motivos padronizados: o candidato lê o mesmo texto para o mesmo problema, e o
-- recurso é agrupável por motivo (02-produto §9).
CREATE TABLE motivo_isencao (
    codigo  text PRIMARY KEY,
    texto   text NOT NULL
);

INSERT INTO motivo_isencao (codigo, texto) VALUES
    ('NIS_AUSENTE',          'O NIS não foi informado.'),
    ('NIS_INVALIDO',         'O NIS informado não tem dígito verificador válido.'),
    ('COMPROVANTE_AUSENTE',  'O comprovante exigido para a hipótese não foi enviado.'),
    ('COMPROVANTE_ALHEIO',   'O comprovante enviado não pertence ao candidato.'),
    ('ANALISE_HUMANA',       'Indeferido na análise documental.'),
    ('RECURSO_PROVIDO',      'Recurso provido.');

CREATE TABLE pedido_isencao (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    certame_id      bigint NOT NULL,
    candidato_id    bigint NOT NULL REFERENCES candidato(id),
    hipotese        text NOT NULL,
    nis             text,
    comprovante_id  bigint REFERENCES documento(id),
    estado          text NOT NULL DEFAULT 'PENDENTE_ANALISE'
                    CHECK (estado IN ('PENDENTE_ANALISE', 'DEFERIDA', 'INDEFERIDA')),
    motivo_codigo   text REFERENCES motivo_isencao(codigo),
    criado_em       timestamptz NOT NULL DEFAULT now(),
    atualizado_em   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (certame_id, candidato_id),
    FOREIGN KEY (certame_id, hipotese) REFERENCES hipotese_isencao(certame_id, codigo),
    CHECK (estado <> 'INDEFERIDA' OR motivo_codigo IS NOT NULL)
);

CREATE TABLE pedido_isencao_transicao (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pedido_id   bigint NOT NULL REFERENCES pedido_isencao(id),
    de          text,
    para        text NOT NULL,
    motivo_codigo text,
    ator        text NOT NULL,
    motivo      text NOT NULL,
    versao_edital integer NOT NULL,
    em          timestamptz NOT NULL DEFAULT clock_timestamp()
);

CREATE TRIGGER pedido_isencao_trilha_imutavel
    BEFORE UPDATE OR DELETE ON pedido_isencao_transicao
    FOR EACH ROW EXECUTE FUNCTION _trilha_imutavel();

-- Dígito verificador do NIS/PIS/PASEP: pesos 3,2,9,8,7,6,5,4,3,2 sobre os 10
-- primeiros dígitos; resto 0 ou 1 dá dígito 0, senão 11 menos o resto.
CREATE FUNCTION nis_valido(p_nis text) RETURNS boolean
LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE
    pesos int[] := ARRAY[3,2,9,8,7,6,5,4,3,2];
    soma int := 0;
    resto int;
BEGIN
    IF p_nis IS NULL OR p_nis !~ '^[0-9]{11}$' OR p_nis ~ '^(.)\1{10}$' THEN
        RETURN false;
    END IF;
    FOR i IN 1..10 LOOP
        soma := soma + substr(p_nis, i, 1)::int * pesos[i];
    END LOOP;
    resto := soma % 11;
    RETURN substr(p_nis, 11, 1)::int = CASE WHEN resto < 2 THEN 0 ELSE 11 - resto END;
END $$;

-- Triagem automática por regra, na entrada do pedido. Não é IA e não decide
-- mérito: confere campo e documento obrigatório. O que passa vai para análise
-- humana; o que falta é indeferido na hora, com motivo padronizado.
CREATE FUNCTION _pedido_isencao_antes() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    h hipotese_isencao;
    dono bigint;
    ctx record;
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NOT isencao_aberta(NEW.certame_id) THEN
            RAISE EXCEPTION 'fora do prazo de isenção do certame %', NEW.certame_id
                USING ERRCODE = 'CT007';
        END IF;
        SELECT * INTO h FROM hipotese_isencao
         WHERE certame_id = NEW.certame_id AND codigo = NEW.hipotese;
        NEW.estado := 'PENDENTE_ANALISE';
        NEW.motivo_codigo := NULL;
        IF h.exige_nis AND NEW.nis IS NULL THEN
            NEW.estado := 'INDEFERIDA'; NEW.motivo_codigo := 'NIS_AUSENTE';
        ELSIF h.exige_nis AND NOT nis_valido(NEW.nis) THEN
            NEW.estado := 'INDEFERIDA'; NEW.motivo_codigo := 'NIS_INVALIDO';
        ELSIF h.exige_comprovante AND NEW.comprovante_id IS NULL THEN
            NEW.estado := 'INDEFERIDA'; NEW.motivo_codigo := 'COMPROVANTE_AUSENTE';
        ELSIF NEW.comprovante_id IS NOT NULL THEN
            SELECT candidato_id INTO dono FROM documento WHERE id = NEW.comprovante_id;
            IF dono IS DISTINCT FROM NEW.candidato_id THEN
                NEW.estado := 'INDEFERIDA'; NEW.motivo_codigo := 'COMPROVANTE_ALHEIO';
            END IF;
        END IF;
        RETURN NEW;
    END IF;

    -- UPDATE: só decisão de estado, com trilha.
    IF (to_jsonb(NEW) - ARRAY['estado','motivo_codigo','atualizado_em'])
       <> (to_jsonb(OLD) - ARRAY['estado','motivo_codigo','atualizado_em']) THEN
        RAISE EXCEPTION 'dados do pedido de isenção não mudam depois de enviado'
            USING ERRCODE = 'CT004';
    END IF;
    IF NEW.estado IS DISTINCT FROM OLD.estado THEN
        -- PENDENTE -> DEFERIDA | INDEFERIDA (análise); INDEFERIDA -> DEFERIDA (recurso provido).
        IF NOT ((OLD.estado = 'PENDENTE_ANALISE' AND NEW.estado IN ('DEFERIDA', 'INDEFERIDA'))
                OR (OLD.estado = 'INDEFERIDA' AND NEW.estado = 'DEFERIDA')) THEN
            RAISE EXCEPTION 'pedido de isenção: % -> % não é permitido', OLD.estado, NEW.estado
                USING ERRCODE = 'CT009';
        END IF;
        IF NEW.estado = 'DEFERIDA' THEN
            NEW.motivo_codigo := CASE WHEN OLD.estado = 'INDEFERIDA' THEN 'RECURSO_PROVIDO' END;
        END IF;
        ctx := _auditoria_contexto();
    END IF;
    NEW.atualizado_em := now();
    RETURN NEW;
END $$;

CREATE FUNCTION _pedido_isencao_depois() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    ator text := coalesce(nullif(current_setting('certamis.ator', true), ''), 'sistema:triagem');
    motivo text := coalesce(nullif(current_setting('certamis.motivo', true), ''), 'triagem automática');
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.estado IS NOT DISTINCT FROM OLD.estado THEN
        RETURN NULL;
    END IF;
    INSERT INTO pedido_isencao_transicao
        (pedido_id, de, para, motivo_codigo, ator, motivo, versao_edital)
    VALUES (NEW.id, CASE WHEN TG_OP = 'UPDATE' THEN OLD.estado END, NEW.estado,
            NEW.motivo_codigo, ator, motivo, edital_versao_vigente(NEW.certame_id));
    RETURN NULL;
END $$;

CREATE TRIGGER pedido_isencao_antes
    BEFORE INSERT OR UPDATE ON pedido_isencao
    FOR EACH ROW EXECUTE FUNCTION _pedido_isencao_antes();

CREATE TRIGGER pedido_isencao_depois
    AFTER INSERT OR UPDATE ON pedido_isencao
    FOR EACH ROW EXECUTE FUNCTION _pedido_isencao_depois();

-- ---------------------------------------------------------------- regras da inscrição, revistas

CREATE OR REPLACE FUNCTION _inscricao_antes() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    isento boolean;
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
        IF NOT inscricao_aberta(NEW.certame_id) THEN
            RAISE EXCEPTION 'fora do prazo de inscrição do certame %', NEW.certame_id
                USING ERRCODE = 'CT007';
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

    -- Qualquer dado, exceto o estado, só muda em RASCUNHO.
    IF (to_jsonb(NEW) - ARRAY['estado','atualizada_em'])
       <> (to_jsonb(OLD) - ARRAY['estado','atualizada_em'])
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

        -- Finalizar ou reabrir exige prazo de inscrição aberto.
        IF (OLD.estado = 'RASCUNHO' AND NEW.estado IN ('AGUARDANDO_PAGAMENTO', 'ISENCAO_ATIVA'))
           OR (OLD.estado = 'AGUARDANDO_PAGAMENTO' AND NEW.estado = 'RASCUNHO') THEN
            IF NOT inscricao_aberta(NEW.certame_id) THEN
                RAISE EXCEPTION 'fora do prazo de inscrição do certame %', NEW.certame_id
                    USING ERRCODE = 'CT007';
            END IF;
        END IF;

        -- Finalizar exige formulário completo e o caminho certo de isenção.
        IF OLD.estado = 'RASCUNHO' AND NEW.estado IN ('AGUARDANDO_PAGAMENTO', 'ISENCAO_ATIVA') THEN
            IF NEW.cargo_id IS NULL OR NEW.cidade_id IS NULL OR NEW.modalidade_id IS NULL
               OR NEW.declaracoes_aceitas_em IS NULL THEN
                RAISE EXCEPTION 'inscrição incompleta: cargo, cidade, modalidade e declarações são obrigatórios'
                    USING ERRCODE = 'CT006';
            END IF;
            SELECT EXISTS (SELECT 1 FROM pedido_isencao
                            WHERE certame_id = NEW.certame_id AND candidato_id = NEW.candidato_id
                              AND estado = 'DEFERIDA') INTO isento;
            IF NEW.estado = 'ISENCAO_ATIVA' AND NOT isento THEN
                RAISE EXCEPTION 'candidato sem isenção deferida não pode finalizar como isento'
                    USING ERRCODE = 'CT008';
            ELSIF NEW.estado = 'AGUARDANDO_PAGAMENTO' AND isento THEN
                RAISE EXCEPTION 'candidato com isenção deferida não paga taxa'
                    USING ERRCODE = 'CT008';
            END IF;
        END IF;

        PERFORM _auditoria_contexto();
    END IF;

    NEW.atualizada_em := now();
    RETURN NEW;
END $$;
