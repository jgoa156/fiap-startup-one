-- +goose NO TRANSACTION
-- +goose Up
-- A tela pergunta a cada 1,5 s "o Pix já chegou?" (cobranças de UMA inscrição). Sem este índice o
-- banco lia a tabela inteira a cada pergunta: medido em 05/11, a 640× local, 94% do tempo do banco
-- (202 mil chamadas, 9,2 ms cada, 58.859 linhas descartadas para achar 1). Os índices parciais
-- (PENDENTE, PAGA) não servem a esta consulta, que não filtra por estado.
-- CONCURRENTLY: cria sem travar a tabela, como seria preciso num banco já em uso.
CREATE INDEX CONCURRENTLY IF NOT EXISTS cobranca_por_inscricao ON cobranca (inscricao_id, id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS cobranca_por_inscricao;
