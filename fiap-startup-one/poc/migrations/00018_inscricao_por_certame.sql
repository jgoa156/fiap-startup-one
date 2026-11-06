-- +goose NO TRANSACTION
-- +goose Up
-- A relação de inscritos sai em fatias de números de inscrição (publicacao.TamanhoFatia): cada fatia
-- lê só as inscrições do certame naquela faixa, e o índice da relação acha as fatias pulando de uma
-- para a outra. Os dois precisam de (certame_id, id).
-- CONCURRENTLY: cria sem travar a tabela, como seria preciso num banco já em uso.
CREATE INDEX CONCURRENTLY IF NOT EXISTS inscricao_por_certame_e_id ON inscricao (certame_id, id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS inscricao_por_certame_e_id;
