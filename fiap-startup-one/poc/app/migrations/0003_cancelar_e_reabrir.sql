-- 0003: decisões de 29/10/2026 sobre o que o diagrama de 02-produto §5 não previa.
--   1. O candidato pode cancelar enquanto não pagou.
--   3. O candidato pode voltar ao rascunho para corrigir antes de pagar. A cobrança
--      pendente é invalidada quando o pagamento existir (bloco 2).
-- Depois de PAGA, nenhuma das duas vale: o dinheiro já entrou e a saída é outro fluxo.

INSERT INTO transicao_permitida (de, para, evento) VALUES
    ('AGUARDANDO_PAGAMENTO', 'CANCELADA', 'cancela'),
    ('AGUARDANDO_PAGAMENTO', 'RASCUNHO',  'reabre_para_edicao');
