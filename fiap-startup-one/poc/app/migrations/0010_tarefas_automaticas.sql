-- 0010: homologar enfileira o cartão de confirmação. O PDF nunca é gerado dentro de
-- um request (dívida D3): no pico 3 a base inteira pede o cartão ao mesmo tempo, e
-- ele já tem de existir como arquivo.
CREATE FUNCTION _inscricao_enfileira_cartao() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.estado = 'HOMOLOGADA' AND OLD.estado IS DISTINCT FROM 'HOMOLOGADA' THEN
        INSERT INTO tarefa (tipo, chave, dados)
        VALUES ('gerar_cartao', NEW.id::text, jsonb_build_object('inscricao_id', NEW.id))
        ON CONFLICT (tipo, chave) DO NOTHING;
    END IF;
    RETURN NULL;
END $$;

CREATE TRIGGER inscricao_enfileira_cartao
    AFTER UPDATE OF estado ON inscricao
    FOR EACH ROW EXECUTE FUNCTION _inscricao_enfileira_cartao();

ALTER TABLE inscricao ADD COLUMN cartao_chave text;
