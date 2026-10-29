-- 0009: modo degradado (níveis 1, 4 e 5 de 04-arquitetura-alvo §4.5) e fila de espera (D4).
--
-- Na arquitetura-alvo a fila de espera fica na borda (Cloudflare). Na POC ela fica
-- na aplicação, com o mesmo contrato: quem chega recebe uma senha numerada, vê
-- posição e tempo estimado, e só entra (faz login) quando o portão alcança o
-- número dele. O portão anda a uma taxa fixa, que é a capacidade medida no ensaio.

CREATE TABLE modo_operacao (
    id              integer PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    -- 0 normal · 1 fila de espera · 4 só estático · 5 página de status
    nivel           integer NOT NULL DEFAULT 0 CHECK (nivel IN (0, 1, 4, 5)),
    mensagem        text NOT NULL DEFAULT '',
    atualizado_em   timestamptz NOT NULL DEFAULT now(),
    atualizado_por  text NOT NULL DEFAULT 'sistema'
);
INSERT INTO modo_operacao DEFAULT VALUES;

CREATE TABLE modo_operacao_historico (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nivel       integer NOT NULL,
    mensagem    text NOT NULL,
    por         text NOT NULL,
    em          timestamptz NOT NULL DEFAULT clock_timestamp()
);

CREATE FUNCTION _modo_operacao_historico() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO modo_operacao_historico (nivel, mensagem, por)
    VALUES (NEW.nivel, NEW.mensagem, NEW.atualizado_por);
    RETURN NULL;
END $$;

CREATE TRIGGER modo_operacao_historico
    AFTER INSERT OR UPDATE ON modo_operacao
    FOR EACH ROW EXECUTE FUNCTION _modo_operacao_historico();

CREATE SEQUENCE fila_senha_seq;

CREATE TABLE portao (
    id              integer PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    admitido_ate    bigint NOT NULL DEFAULT 0,
    taxa_por_s      numeric NOT NULL DEFAULT 5 CHECK (taxa_por_s > 0),
    atualizado_em   timestamptz NOT NULL DEFAULT clock_timestamp()
);
INSERT INTO portao DEFAULT VALUES;

-- Anda o portão pelo tempo decorrido, sem passar da última senha emitida (senão a
-- capacidade ociosa viraria crédito e liberaria uma rajada depois). Chamada a cada
-- segundo por uma instância só, que segura a trava consultiva.
CREATE FUNCTION avancar_portao() RETURNS portao
LANGUAGE plpgsql AS $$
DECLARE
    p portao;
    emitidas bigint;
    agora timestamptz := clock_timestamp();
    incremento bigint;
BEGIN
    SELECT CASE WHEN is_called THEN last_value ELSE 0 END INTO emitidas FROM fila_senha_seq;
    SELECT * INTO p FROM portao WHERE id = 1 FOR UPDATE;
    IF p.admitido_ate >= emitidas THEN
        -- Fila vazia: o relógio acompanha, e capacidade ociosa não vira crédito.
        UPDATE portao SET atualizado_em = agora WHERE id = 1 RETURNING * INTO p;
        RETURN p;
    END IF;
    incremento := floor(p.taxa_por_s * extract(epoch FROM agora - p.atualizado_em));
    IF incremento >= 1 THEN
        -- Consome só o tempo correspondente ao que foi admitido; a fração fica para a próxima.
        UPDATE portao
           SET admitido_ate = least(emitidas, admitido_ate + incremento),
               atualizado_em = CASE WHEN admitido_ate + incremento >= emitidas THEN agora
                                    ELSE atualizado_em + make_interval(secs => incremento / taxa_por_s) END
         WHERE id = 1
        RETURNING * INTO p;
    END IF;
    RETURN p;
END $$;
