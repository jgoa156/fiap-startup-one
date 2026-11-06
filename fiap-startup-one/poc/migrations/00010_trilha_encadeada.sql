-- +goose Up

-- Trilha encadeada: cada evento guarda um resumo (SHA-256) do próprio conteúdo e do resumo do evento
-- anterior da mesma inscrição. O gatilho de imutabilidade barra a aplicação, mas não um
-- superusuário, que pode desligá-lo; com a corrente, alterar ou apagar um evento antigo quebra todos
-- os resumos dali em diante, e a auditoria acha. Refazer a corrente inteira também é possível para um
-- superusuário — por isso o cartão de confirmação imprime o selo (o resumo do último evento): o
-- candidato fica com uma prova que nenhuma reescrita reproduz.
--
-- A corrente é por inscrição porque os eventos de uma inscrição já são gravados em série (a linha da
-- inscrição fica travada na transição); uma corrente única para todas serializaria o certame inteiro.

-- +goose StatementBegin
CREATE FUNCTION trilha_resumo(anterior bytea, e inscricao_evento) RETURNS bytea LANGUAGE sql STABLE AS $$
    SELECT sha256(coalesce(anterior, '\x'::bytea) || convert_to(concat_ws('|',
        e.inscricao_id, e.evento, e.de_estado, e.para_estado, e.ator_tipo, e.ator_id,
        coalesce(e.motivo_codigo, ''), e.edital_versao,
        to_char(e.ocorrido_em AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US')), 'UTF8'))
$$;
-- +goose StatementEnd

ALTER TABLE inscricao_evento ADD COLUMN hash bytea;

-- Os eventos que já existem ganham a corrente, em ordem. O gatilho de imutabilidade é desligado só
-- dentro desta migração (que roda numa transação: ou tudo, ou nada).
ALTER TABLE inscricao_evento DISABLE TRIGGER inscricao_evento_imutavel;
-- +goose StatementBegin
DO $$
DECLARE
    r inscricao_evento;
    anterior bytea;
    atual bigint := -1;
BEGIN
    FOR r IN SELECT * FROM inscricao_evento ORDER BY inscricao_id, id LOOP
        IF r.inscricao_id <> atual THEN
            anterior := NULL;
            atual := r.inscricao_id;
        END IF;
        anterior := trilha_resumo(anterior, r);
        UPDATE inscricao_evento SET hash = anterior WHERE id = r.id;
    END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE inscricao_evento ENABLE TRIGGER inscricao_evento_imutavel;
ALTER TABLE inscricao_evento ALTER COLUMN hash SET NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION trilha_encadear() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    -- O resumo é sempre calculado aqui: quem grava não escolhe o próprio.
    NEW.hash := trilha_resumo(
        (SELECT hash FROM inscricao_evento WHERE inscricao_id = NEW.inscricao_id ORDER BY id DESC LIMIT 1), NEW);
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER inscricao_evento_encadear BEFORE INSERT ON inscricao_evento
    FOR EACH ROW EXECUTE FUNCTION trilha_encadear();

-- +goose Down
DROP TRIGGER inscricao_evento_encadear ON inscricao_evento;
DROP FUNCTION trilha_encadear();
DROP FUNCTION trilha_resumo(bytea, inscricao_evento);
ALTER TABLE inscricao_evento DROP COLUMN hash;
