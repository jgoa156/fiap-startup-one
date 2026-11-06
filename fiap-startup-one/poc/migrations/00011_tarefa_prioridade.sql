-- +goose Up
-- Prioridade na fila: 0 urgente (o candidato espera olhando a tela: o Pix), 1 normal, 2 lote
-- (republicar 100 mil cartões depois da homologação). Sem ela, medido em 05/11, a homologação de
-- 100 mil inscrições deixou a republicação da relação 1 min 40 s atrás dos cartões — e o Pix de quem
-- se inscrevesse naquela hora esperaria o mesmo.
ALTER TABLE tarefa ADD COLUMN prioridade smallint NOT NULL DEFAULT 1 CHECK (prioridade BETWEEN 0 AND 2);
DROP INDEX tarefa_a_executar;
CREATE INDEX tarefa_a_executar ON tarefa (prioridade, executar_em) WHERE estado = 'PENDENTE';

-- +goose Down
DROP INDEX tarefa_a_executar;
CREATE INDEX tarefa_a_executar ON tarefa (executar_em) WHERE estado = 'PENDENTE';
ALTER TABLE tarefa DROP COLUMN prioridade;
