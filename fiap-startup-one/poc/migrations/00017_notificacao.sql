-- +goose Up

-- O aviso ao candidato a cada mudança de estado que importa para ele (02-produto §7 B). Uma linha por
-- evento da trilha: a prova de que o candidato foi avisado, quando, com que assunto e com que texto
-- (o SHA-256, para conferir uma cópia). Não muda depois: é o que sustenta um prazo de recurso.
CREATE TABLE notificacao (
    evento_id     bigint PRIMARY KEY REFERENCES inscricao_evento (id),
    inscricao_id  bigint NOT NULL REFERENCES inscricao (id),
    destinatario  text NOT NULL,
    assunto       text NOT NULL,
    resumo        char(64) NOT NULL,          -- SHA-256 de assunto + texto
    provedor_id   text,                       -- nulo quando o provedor recusou o endereço
    falha         text,                       -- por que não saiu (endereço recusado)
    enviada_em    timestamptz NOT NULL DEFAULT now(),
    CHECK ((provedor_id IS NULL) <> (falha IS NULL))
);
CREATE INDEX notificacao_por_inscricao ON notificacao (inscricao_id, evento_id);
CREATE TRIGGER notificacao_imutavel BEFORE UPDATE OR DELETE ON notificacao
    FOR EACH ROW EXECUTE FUNCTION trilha_imutavel();

-- +goose Down
DROP TABLE notificacao;
