-- Testes das regras que moram no banco. Rodam contra um banco recém-migrado.
-- Cada bloco falha com ASSERT ou com exceção inesperada; o arquivo inteiro roda
-- com ON_ERROR_STOP, então qualquer falha interrompe e o pytest acusa.

\set ON_ERROR_STOP on
SET client_min_messages = warning;

-- Dado sintético mínimo: uma banca, um certame com edital v1, um candidato.
INSERT INTO banca (nome) VALUES ('Banca Sintética');
INSERT INTO certame (banca_id, nome, taxa_centavos, inscricao_inicio, inscricao_fim)
VALUES (1, 'Certame Sintético', 8000, now() - interval '1 day', now() + interval '29 days');
INSERT INTO certame (banca_id, nome, taxa_centavos, inscricao_inicio, inscricao_fim)
VALUES (1, 'Certame sem edital', 8000, now() - interval '1 day', now() + interval '29 days');
INSERT INTO edital_versao (certame_id, versao, descricao) VALUES (1, 1, 'edital original');
INSERT INTO candidato (cpf, nome, email) VALUES
    ('00000000191', 'Candidato Um', 'um@exemplo.invalid'),
    ('00000000272', 'Candidato Dois', 'dois@exemplo.invalid');

CREATE FUNCTION pg_temp.espera_erro(sql text, codigo text) RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    BEGIN
        EXECUTE sql;
    EXCEPTION WHEN OTHERS THEN
        IF SQLSTATE <> codigo THEN
            RAISE EXCEPTION 'esperava SQLSTATE %, veio % (%) em: %', codigo, SQLSTATE, SQLERRM, sql;
        END IF;
        RETURN;
    END;
    RAISE EXCEPTION 'esperava erro % e não houve erro em: %', codigo, sql;
END $$;

CREATE FUNCTION pg_temp.ctx(ator text, motivo text) RETURNS void
LANGUAGE sql AS $$
    SELECT set_config('certamis.ator', ator, false), set_config('certamis.motivo', motivo, false);
$$;

-- 1. Criar inscrição sem ator/motivo é recusado.
SELECT pg_temp.espera_erro(
    $q$INSERT INTO inscricao (certame_id, candidato_id) VALUES (1, 1)$q$, 'CT002');

-- 2. Inscrição em certame sem edital publicado é recusada.
SELECT pg_temp.ctx('candidato:00000000191', 'inscrição iniciada');
SELECT pg_temp.espera_erro(
    $q$INSERT INTO inscricao (certame_id, candidato_id) VALUES (2, 1)$q$, 'CT005');

-- 3. Inscrição não nasce fora de RASCUNHO.
SELECT pg_temp.espera_erro(
    $q$INSERT INTO inscricao (certame_id, candidato_id, estado) VALUES (1, 1, 'PAGA')$q$, 'CT001');

-- 4. Criação válida grava a transição inicial NULL -> RASCUNHO, com versão 1.
INSERT INTO inscricao (certame_id, candidato_id) VALUES (1, 1);
DO $$ BEGIN
    ASSERT (SELECT count(*) FROM inscricao_transicao WHERE inscricao_id = (SELECT id FROM inscricao WHERE candidato_id = 1)) = 1;
    ASSERT (SELECT de IS NULL AND para = 'RASCUNHO' AND versao_edital = 1
                   AND ator = 'candidato:00000000191'
              FROM inscricao_transicao WHERE inscricao_id = (SELECT id FROM inscricao WHERE candidato_id = 1));
END $$;

-- 5. Uma inscrição por candidato por certame.
SELECT pg_temp.espera_erro(
    $q$INSERT INTO inscricao (certame_id, candidato_id) VALUES (1, 1)$q$, '23505');

-- 6. Pulos de estado são recusados (ex.: RASCUNHO -> PAGA, RASCUNHO -> HOMOLOGADA).
SELECT pg_temp.ctx('candidato:00000000191', 'tentativa de pular etapa');
SELECT pg_temp.espera_erro($q$UPDATE inscricao SET estado = 'PAGA' WHERE candidato_id = 1$q$, 'CT001');
SELECT pg_temp.espera_erro($q$UPDATE inscricao SET estado = 'HOMOLOGADA' WHERE candidato_id = 1$q$, 'CT001');

-- 7. Transição válida sem motivo é recusada.
SELECT set_config('certamis.motivo', '', false);
SELECT pg_temp.espera_erro(
    $q$UPDATE inscricao SET estado = 'AGUARDANDO_PAGAMENTO' WHERE candidato_id = 1$q$, 'CT002');

-- 8. Dados mudam em RASCUNHO, sem gerar transição.
INSERT INTO cargo (certame_id, nome, vagas) VALUES (1, 'Analista', 10);
UPDATE inscricao SET cargo_id = 1 WHERE candidato_id = 1;
DO $$ BEGIN
    ASSERT (SELECT count(*) FROM inscricao_transicao WHERE inscricao_id = (SELECT id FROM inscricao WHERE candidato_id = 1)) = 1;
END $$;

-- 9. Caminho completo do pagante, com retificação do edital no meio.
SELECT pg_temp.ctx('candidato:00000000191', 'finalizou o formulário');
UPDATE inscricao SET estado = 'AGUARDANDO_PAGAMENTO' WHERE candidato_id = 1;
INSERT INTO edital_versao (certame_id, versao, descricao) VALUES (1, 2, 'retificação 1');
SELECT pg_temp.ctx('sistema:conciliacao', 'pagamento confirmado pelo provedor');
UPDATE inscricao SET estado = 'PAGA' WHERE candidato_id = 1;
SELECT pg_temp.ctx('sistema', 'fim do prazo de inscrição');
UPDATE inscricao SET estado = 'EM_ANALISE' WHERE candidato_id = 1;
SELECT pg_temp.ctx('banca:analista7', 'documentação conforme');
UPDATE inscricao SET estado = 'DEFERIDA' WHERE candidato_id = 1;
SELECT pg_temp.ctx('banca:coordenacao', 'homologação preliminar');
UPDATE inscricao SET estado = 'HOMOLOGADA' WHERE candidato_id = 1;

DO $$
DECLARE
    caminho text;
    versoes text;
BEGIN
    SELECT string_agg(coalesce(de, '-') || '>' || para, ' ' ORDER BY id),
           string_agg(versao_edital::text, ' ' ORDER BY id)
      INTO caminho, versoes
      FROM inscricao_transicao WHERE inscricao_id = (SELECT id FROM inscricao WHERE candidato_id = 1);
    ASSERT caminho = '->RASCUNHO RASCUNHO>AGUARDANDO_PAGAMENTO AGUARDANDO_PAGAMENTO>PAGA '
                     'PAGA>EM_ANALISE EM_ANALISE>DEFERIDA DEFERIDA>HOMOLOGADA',
        'caminho errado: ' || caminho;
    -- As duas primeiras foram sob a v1; depois da retificação, v2.
    ASSERT versoes = '1 1 2 2 2 2', 'versões erradas: ' || versoes;
END $$;

-- 10. Estado final não sai do lugar.
SELECT pg_temp.espera_erro($q$UPDATE inscricao SET estado = 'EM_ANALISE' WHERE candidato_id = 1$q$, 'CT001');

-- 11. Fora do RASCUNHO, dados da inscrição ficam congelados.
INSERT INTO cargo (certame_id, nome, vagas) VALUES (1, 'Técnico', 20);
SELECT pg_temp.espera_erro($q$UPDATE inscricao SET cargo_id = 2 WHERE candidato_id = 1$q$, 'CT004');

-- 12. Certame e candidato da inscrição nunca mudam.
SELECT pg_temp.espera_erro($q$UPDATE inscricao SET candidato_id = 2 WHERE candidato_id = 1$q$, 'CT004');

-- 13. A trilha é imutável: sem UPDATE, sem DELETE, sem TRUNCATE.
SELECT pg_temp.espera_erro($q$UPDATE inscricao_transicao SET motivo = 'x'$q$, 'CT003');
SELECT pg_temp.espera_erro($q$DELETE FROM inscricao_transicao$q$, 'CT003');
SELECT pg_temp.espera_erro($q$TRUNCATE inscricao_transicao$q$, 'CT003');

-- 14. Inscrição não se apaga.
SELECT pg_temp.espera_erro($q$DELETE FROM inscricao WHERE candidato_id = 1$q$, 'CT003');

-- 15. Caminho do isento indeferido com recurso provido.
SELECT pg_temp.ctx('candidato:00000000272', 'inscrição iniciada');
INSERT INTO inscricao (certame_id, candidato_id) VALUES (1, 2);
SELECT pg_temp.ctx('candidato:00000000272', 'isenção deferida');
UPDATE inscricao SET estado = 'ISENCAO_ATIVA' WHERE candidato_id = 2;
SELECT pg_temp.ctx('sistema', 'fim do prazo');
UPDATE inscricao SET estado = 'EM_ANALISE' WHERE candidato_id = 2;
SELECT pg_temp.ctx('banca:analista7', 'laudo ilegível');
UPDATE inscricao SET estado = 'INDEFERIDA' WHERE candidato_id = 2;
SELECT pg_temp.ctx('banca:comissao', 'recurso provido: laudo reenviado legível');
UPDATE inscricao SET estado = 'HOMOLOGADA' WHERE candidato_id = 2;
DO $$ BEGIN
    ASSERT (SELECT estado FROM inscricao WHERE candidato_id = 2) = 'HOMOLOGADA';
    ASSERT (SELECT count(*) FROM inscricao_transicao t JOIN inscricao i ON i.id = t.inscricao_id
             WHERE i.candidato_id = 2) = 5;
END $$;

SELECT 'TODAS AS REGRAS DO BANCO PASSARAM' AS resultado;
