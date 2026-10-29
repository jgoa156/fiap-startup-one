-- 0006: cobrança e confirmação de pagamento. Orquestramos, não processamos: o
-- dinheiro vai do candidato para a conta da banca no provedor dela. O que é nosso
-- é a garantia de que ninguém paga duas vezes e nenhum pagamento se perde.
--
-- As duas garantias, e onde moram:
--   1. No máximo uma cobrança viva (PENDENTE ou PAGA) por inscrição: índice único parcial.
--   2. Cada evento do provedor é aplicado no máximo uma vez: chave única no id do evento.
-- Pagamento que chega para cobrança que já não está pendente não some: fica
-- registrado como divergência de conciliação, para estorno pela banca.
--
-- Ordem de trava, nas duas pontas (cancelar e confirmar): inscrição, depois cobrança.
-- A mesma ordem evita deadlock entre "candidato cancelou" e "provedor confirmou".
--
-- Códigos de erro novos:
--   CT010  inscrição não está aguardando pagamento
--   CT011  inscrição já paga

ALTER TABLE certame ADD COLUMN pagamento_fim timestamptz;
UPDATE certame SET pagamento_fim = inscricao_fim + interval '1 day' WHERE pagamento_fim IS NULL;
ALTER TABLE certame ALTER COLUMN pagamento_fim SET NOT NULL;
ALTER TABLE certame ADD CHECK (pagamento_fim >= inscricao_fim);

-- Padrão de mercado: o pagamento vai até o dia seguinte ao fim da inscrição.
CREATE FUNCTION _certame_padroes() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.pagamento_fim := coalesce(NEW.pagamento_fim, NEW.inscricao_fim + interval '1 day');
    RETURN NEW;
END $$;

CREATE TRIGGER certame_padroes
    BEFORE INSERT OR UPDATE ON certame
    FOR EACH ROW EXECUTE FUNCTION _certame_padroes();

CREATE TABLE cobranca (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    inscricao_id        bigint NOT NULL REFERENCES inscricao(id),
    chave_idempotencia  text NOT NULL CHECK (length(chave_idempotencia) BETWEEN 8 AND 200),
    meio                text NOT NULL CHECK (meio IN ('PIX', 'BOLETO', 'CARTAO')),
    valor_centavos      integer NOT NULL CHECK (valor_centavos > 0),
    status              text NOT NULL DEFAULT 'PENDENTE'
                        CHECK (status IN ('PENDENTE', 'PAGA', 'CANCELADA', 'EXPIRADA')),
    -- Referência que nós damos ao provedor (txid do Pix, "seu número" do boleto).
    -- Ser nossa, e não dele, torna o registro no provedor repetível sem duplicar.
    referencia          text GENERATED ALWAYS AS ('CTM' || lpad(id::text, 12, '0')) STORED UNIQUE,
    vence_em            timestamptz NOT NULL,
    criada_em           timestamptz NOT NULL DEFAULT now(),
    atualizada_em       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (inscricao_id, chave_idempotencia)
);

CREATE UNIQUE INDEX cobranca_uma_viva_por_inscricao
    ON cobranca (inscricao_id) WHERE status IN ('PENDENTE', 'PAGA');
CREATE INDEX cobranca_pendente_vencimento ON cobranca (vence_em) WHERE status = 'PENDENTE';

CREATE TABLE pagamento_evento (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    provedor            text NOT NULL,
    evento_id           text NOT NULL,
    referencia          text NOT NULL,
    valor_centavos      integer NOT NULL,
    resultado           text NOT NULL CHECK (resultado IN (
                            'APLICADO', 'COBRANCA_INATIVA', 'VALOR_DIVERGENTE',
                            'REFERENCIA_DESCONHECIDA')),
    payload             jsonb NOT NULL DEFAULT '{}',
    recebido_em         timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (provedor, evento_id)
);

CREATE TRIGGER pagamento_evento_imutavel
    BEFORE UPDATE OR DELETE ON pagamento_evento
    FOR EACH ROW EXECUTE FUNCTION _trilha_imutavel();

-- Gera (ou devolve) a cobrança da inscrição. Repetir com a mesma chave devolve a
-- mesma cobrança; repetir com outra chave enquanto há uma pendente também devolve
-- a pendente. Nunca existem duas vivas.
CREATE FUNCTION gerar_cobranca(p_inscricao_id bigint, p_chave text, p_meio text)
RETURNS cobranca
LANGUAGE plpgsql AS $$
DECLARE
    insc inscricao;
    cert certame;
    c cobranca;
BEGIN
    SELECT * INTO c FROM cobranca
     WHERE inscricao_id = p_inscricao_id AND chave_idempotencia = p_chave;
    IF FOUND THEN
        RETURN c;
    END IF;

    SELECT * INTO insc FROM inscricao WHERE id = p_inscricao_id FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'inscrição % não existe', p_inscricao_id USING ERRCODE = 'P0002';
    END IF;

    -- Depois da trava: outra requisição pode ter criado entre a leitura e aqui.
    SELECT * INTO c FROM cobranca
     WHERE inscricao_id = p_inscricao_id AND chave_idempotencia = p_chave;
    IF FOUND THEN
        RETURN c;
    END IF;

    SELECT * INTO c FROM cobranca
     WHERE inscricao_id = p_inscricao_id AND status IN ('PENDENTE', 'PAGA');
    IF FOUND THEN
        IF c.status = 'PAGA' THEN
            RAISE EXCEPTION 'inscrição % já está paga', p_inscricao_id USING ERRCODE = 'CT011';
        END IF;
        RETURN c;
    END IF;

    IF insc.estado <> 'AGUARDANDO_PAGAMENTO' THEN
        RAISE EXCEPTION 'inscrição % está em %, não aguarda pagamento', p_inscricao_id, insc.estado
            USING ERRCODE = 'CT010';
    END IF;

    SELECT * INTO cert FROM certame WHERE id = insc.certame_id;
    INSERT INTO cobranca (inscricao_id, chave_idempotencia, meio, valor_centavos, vence_em)
    VALUES (p_inscricao_id, p_chave, p_meio, cert.taxa_centavos, cert.pagamento_fim)
    RETURNING * INTO c;
    RETURN c;
END $$;

-- Aplica um evento de pagamento do provedor. Idempotente pelo id do evento.
-- Devolve o resultado gravado, ou DUPLICADO se o evento já tinha chegado.
CREATE FUNCTION confirmar_pagamento(
    p_provedor text, p_evento_id text, p_referencia text, p_valor integer,
    p_payload jsonb DEFAULT '{}')
RETURNS text
LANGUAGE plpgsql AS $$
DECLARE
    c cobranca;
    insc_id bigint;
    resultado text;
BEGIN
    -- Serializa eventos repetidos: o segundo espera o primeiro e então vê a linha.
    PERFORM pg_advisory_xact_lock(hashtextextended(p_provedor || ':' || p_evento_id, 0));
    IF EXISTS (SELECT 1 FROM pagamento_evento
                WHERE provedor = p_provedor AND evento_id = p_evento_id) THEN
        RETURN 'DUPLICADO';
    END IF;

    SELECT inscricao_id INTO insc_id FROM cobranca WHERE referencia = p_referencia;
    IF NOT FOUND THEN
        resultado := 'REFERENCIA_DESCONHECIDA';
    ELSE
        PERFORM 1 FROM inscricao WHERE id = insc_id FOR UPDATE;      -- trava 1: inscrição
        SELECT * INTO c FROM cobranca WHERE referencia = p_referencia FOR UPDATE;  -- trava 2
        IF c.status <> 'PENDENTE' THEN
            resultado := 'COBRANCA_INATIVA';
        ELSIF c.valor_centavos <> p_valor THEN
            resultado := 'VALOR_DIVERGENTE';
        ELSE
            UPDATE cobranca SET status = 'PAGA', atualizada_em = now() WHERE id = c.id;
            PERFORM set_config('certamis.ator', 'provedor:' || p_provedor, true),
                    set_config('certamis.motivo', 'pagamento confirmado, evento ' || p_evento_id, true);
            UPDATE inscricao SET estado = 'PAGA' WHERE id = insc_id;
            resultado := 'APLICADO';
        END IF;
    END IF;

    INSERT INTO pagamento_evento (provedor, evento_id, referencia, valor_centavos, resultado, payload)
    VALUES (p_provedor, p_evento_id, p_referencia, p_valor, resultado, p_payload);
    RETURN resultado;
END $$;

-- Quando a inscrição sai de AGUARDANDO_PAGAMENTO sem pagar, a cobrança pendente morre junto.
CREATE FUNCTION _inscricao_encerra_cobranca() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.estado = 'AGUARDANDO_PAGAMENTO' AND NEW.estado IN ('RASCUNHO', 'CANCELADA', 'EXPIRADA') THEN
        UPDATE cobranca
           SET status = CASE WHEN NEW.estado = 'EXPIRADA' THEN 'EXPIRADA' ELSE 'CANCELADA' END,
               atualizada_em = now()
         WHERE inscricao_id = NEW.id AND status = 'PENDENTE';
    END IF;
    RETURN NULL;
END $$;

CREATE TRIGGER inscricao_encerra_cobranca
    AFTER UPDATE OF estado ON inscricao
    FOR EACH ROW EXECUTE FUNCTION _inscricao_encerra_cobranca();

-- Expira as inscrições cujo prazo de pagamento passou. Roda por agendamento.
CREATE FUNCTION expirar_pagamentos_vencidos() RETURNS integer
LANGUAGE plpgsql AS $$
DECLARE
    n integer;
BEGIN
    PERFORM set_config('certamis.ator', 'sistema:expiracao', true),
            set_config('certamis.motivo', 'prazo de pagamento vencido', true);
    WITH vencidas AS (
        SELECT i.id FROM inscricao i JOIN certame c ON c.id = i.certame_id
         WHERE i.estado = 'AGUARDANDO_PAGAMENTO' AND c.pagamento_fim < now()
         FOR UPDATE OF i SKIP LOCKED
    )
    UPDATE inscricao SET estado = 'EXPIRADA' WHERE id IN (SELECT id FROM vencidas);
    GET DIAGNOSTICS n = ROW_COUNT;
    RETURN n;
END $$;

-- Conciliação: tudo que precisa de olho humano, e os invariantes que o ensaio confere.
CREATE VIEW conciliacao AS
SELECT 'EVENTO_NAO_APLICADO' AS tipo, e.referencia, e.evento_id AS detalhe, e.recebido_em AS em
  FROM pagamento_evento e WHERE e.resultado <> 'APLICADO'
UNION ALL
SELECT 'INSCRICAO_PAGA_SEM_COBRANCA_PAGA', i.id::text, i.estado, i.atualizada_em
  FROM inscricao i
 WHERE i.id IN (SELECT inscricao_id FROM inscricao_transicao WHERE para = 'PAGA')
   AND NOT EXISTS (SELECT 1 FROM cobranca c WHERE c.inscricao_id = i.id AND c.status = 'PAGA')
UNION ALL
SELECT 'MAIS_DE_UM_EVENTO_APLICADO', c.referencia, count(*)::text, max(e.recebido_em)
  FROM cobranca c JOIN pagamento_evento e ON e.referencia = c.referencia AND e.resultado = 'APLICADO'
 GROUP BY c.referencia HAVING count(*) > 1;
