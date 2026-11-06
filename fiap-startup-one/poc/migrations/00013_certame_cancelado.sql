-- +goose Up

-- Certame suspenso ou anulado (02-produto §C, "reembolso em massa"): a banca cancela, toda inscrição
-- viva vai a CANCELADA e todo pagamento confirmado volta. Depois disso o certame não aceita
-- inscrição nem cobrança nova.
ALTER TABLE certame
    ADD COLUMN cancelado_em         timestamptz,
    ADD COLUMN cancelamento_motivo  text,
    ADD CONSTRAINT certame_cancelamento_com_motivo CHECK ((cancelado_em IS NULL) = (cancelamento_motivo IS NULL));

ALTER TABLE devolucao DROP CONSTRAINT devolucao_motivo_check;
ALTER TABLE devolucao ADD CONSTRAINT devolucao_motivo_check
    CHECK (motivo IN ('PAGO_EM_DUPLICIDADE', 'PAGO_APOS_EXPIRACAO', 'CERTAME_CANCELADO'));

-- +goose Down
-- Falha de propósito se houver devolução por cancelamento: apagar registro de dinheiro devolvido para
-- poder voltar a migração seria pior que não voltar.
ALTER TABLE devolucao DROP CONSTRAINT devolucao_motivo_check;
ALTER TABLE devolucao ADD CONSTRAINT devolucao_motivo_check
    CHECK (motivo IN ('PAGO_EM_DUPLICIDADE', 'PAGO_APOS_EXPIRACAO'));
ALTER TABLE certame
    DROP CONSTRAINT certame_cancelamento_com_motivo,
    DROP COLUMN cancelamento_motivo,
    DROP COLUMN cancelado_em;
