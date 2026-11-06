-- +goose NO TRANSACTION
-- +goose Up
-- A prestação de contas e a auditoria vão do certame aos pagamentos pela cobrança; sem este índice,
-- cada uma lia a tabela inteira de pagamentos (343 mil linhas no banco local em 05/11).
-- CONCURRENTLY: cria sem travar a tabela, como seria preciso num banco já em uso.
CREATE INDEX CONCURRENTLY IF NOT EXISTS pagamento_por_cobranca ON pagamento_recebido (cobranca_id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS pagamento_por_cobranca;
