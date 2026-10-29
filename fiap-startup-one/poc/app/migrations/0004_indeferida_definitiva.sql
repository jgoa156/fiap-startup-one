-- 0004: recurso negado vira estado final próprio (decisão de 29/10/2026).
-- Antes, INDEFERIDA misturava "recurso pendente" e "recurso negado", e o banco
-- aceitaria homologar quem já tinha o recurso negado. Agora negar o recurso é
-- uma transição gravada na trilha, com ator e motivo, e o estado não tem saída.

INSERT INTO estado_inscricao (codigo, final) VALUES ('INDEFERIDA_DEFINITIVA', true);

INSERT INTO transicao_permitida (de, para, evento) VALUES
    ('INDEFERIDA', 'INDEFERIDA_DEFINITIVA', 'nega_recurso');
