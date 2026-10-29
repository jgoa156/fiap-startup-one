-- Bloco 2: prazos, formulário completo, isenção por regra, cobrança e pagamento.
\set ON_ERROR_STOP on
SET client_min_messages = warning;

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

CREATE FUNCTION pg_temp.confere(ok boolean, o_que text) RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    IF ok IS NOT TRUE THEN RAISE EXCEPTION 'falhou: %', o_que; END IF;
END $$;

-- Cria um candidato com inscrição completa em RASCUNHO e devolve o id da inscrição.
CREATE FUNCTION pg_temp.inscrito(p_cpf text) RETURNS bigint
LANGUAGE plpgsql AS $$
DECLARE cid bigint; iid bigint;
BEGIN
    INSERT INTO candidato (cpf, nome, email) VALUES (p_cpf, 'Sintético ' || p_cpf, p_cpf || '@exemplo.invalid')
    RETURNING id INTO cid;
    PERFORM pg_temp.ctx('candidato:' || p_cpf, 'inscrição iniciada');
    INSERT INTO inscricao (certame_id, candidato_id, cargo_id, cidade_id, modalidade_id, declaracoes_aceitas_em)
    VALUES (1, cid, 1, 1, 1, now()) RETURNING id INTO iid;
    RETURN iid;
END $$;

CREATE FUNCTION pg_temp.finaliza(p_iid bigint) RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_temp.ctx('candidato', 'finalizou o formulário');
    UPDATE inscricao SET estado = 'AGUARDANDO_PAGAMENTO' WHERE id = p_iid;
END $$;

INSERT INTO banca (nome) VALUES ('Banca Sintética');
INSERT INTO certame (banca_id, nome, taxa_centavos, isencao_inicio, isencao_fim,
                     inscricao_inicio, inscricao_fim)
VALUES (1, 'Certame Sintético', 8000, now() - interval '2 days', now() + interval '5 days',
        now() - interval '1 day', now() + interval '29 days');
INSERT INTO edital_versao (certame_id, versao, descricao) VALUES (1, 1, 'edital original');
INSERT INTO cargo (certame_id, nome, vagas) VALUES (1, 'Analista', 10);
INSERT INTO cidade_prova (certame_id, nome) VALUES (1, 'Sede');
INSERT INTO modalidade (certame_id, codigo, nome) VALUES (1, 'AMPLA', 'Ampla concorrência');
INSERT INTO hipotese_isencao (certame_id, codigo, nome, exige_nis, exige_comprovante) VALUES
    (1, 'CADUNICO', 'Inscrito no CadÚnico', true, false),
    (1, 'DOADOR_MEDULA', 'Doador de medula óssea', false, true);

-- ============================================================ prazos e formulário

-- 1. Certame com inscrição encerrada não aceita inscrição nova.
INSERT INTO certame (banca_id, nome, taxa_centavos, inscricao_inicio, inscricao_fim)
VALUES (1, 'Certame encerrado', 8000, now() - interval '30 days', now() - interval '1 day');
INSERT INTO edital_versao (certame_id, versao, descricao) VALUES (2, 1, 'edital');
INSERT INTO candidato (cpf, nome, email) VALUES ('90000000001', 'Atrasado', 'a@exemplo.invalid');
SELECT pg_temp.ctx('candidato:90000000001', 'inscrição iniciada');
SELECT pg_temp.espera_erro(
    $q$INSERT INTO inscricao (certame_id, candidato_id) VALUES (2, (SELECT id FROM candidato WHERE cpf = '90000000001'))$q$,
    'CT007');

-- 2. Formulário incompleto não finaliza.
INSERT INTO candidato (cpf, nome, email) VALUES ('90000000002', 'Incompleto', 'i@exemplo.invalid');
INSERT INTO inscricao (certame_id, candidato_id, cargo_id)
VALUES (1, (SELECT id FROM candidato WHERE cpf = '90000000002'), 1);
SELECT pg_temp.ctx('candidato:90000000002', 'finalizou');
SELECT pg_temp.espera_erro($q$UPDATE inscricao SET estado = 'AGUARDANDO_PAGAMENTO'
    WHERE candidato_id = (SELECT id FROM candidato WHERE cpf = '90000000002')$q$, 'CT006');

-- 3. Quem não tem isenção deferida não finaliza como isento.
SELECT pg_temp.inscrito('90000000003') AS i3 \gset
SELECT pg_temp.espera_erro(format($q$UPDATE inscricao SET estado = 'ISENCAO_ATIVA' WHERE id = %s$q$, :i3), 'CT008');

-- 4. Finalizar depois do fim do prazo é recusado; reabrir também.
SELECT pg_temp.inscrito('90000000004') AS i4 \gset
SELECT pg_temp.finaliza(:i4);
UPDATE certame SET inscricao_fim = now() - interval '1 minute' WHERE id = 1;
SELECT pg_temp.espera_erro(format($q$UPDATE inscricao SET estado = 'RASCUNHO' WHERE id = %s$q$, :i4), 'CT007');
SELECT pg_temp.espera_erro(format($q$UPDATE inscricao SET estado = 'AGUARDANDO_PAGAMENTO' WHERE id = %s$q$, :i3), 'CT007');
UPDATE certame SET inscricao_fim = now() + interval '29 days', pagamento_fim = now() + interval '30 days' WHERE id = 1;

-- 5. Fora do RASCUNHO nenhum dado muda, inclusive os novos.
SELECT pg_temp.espera_erro(format($q$UPDATE inscricao SET atendimento_especial = true WHERE id = %s$q$, :i4), 'CT004');

-- ============================================================ isenção

-- 6. Dígito do NIS.
DO $$ BEGIN
    ASSERT nis_valido('12034567899') AND nis_valido('17000000013');
    ASSERT NOT nis_valido('12034567898') AND NOT nis_valido('11111111111')
       AND NOT nis_valido('123') AND NOT nis_valido(NULL);
END $$;

-- 7. Triagem automática: cada falta vira indeferimento imediato com motivo padronizado.
INSERT INTO candidato (cpf, nome, email) VALUES
    ('91000000001', 'Sem NIS', 'x1@exemplo.invalid'),
    ('91000000002', 'NIS errado', 'x2@exemplo.invalid'),
    ('91000000003', 'Sem comprovante', 'x3@exemplo.invalid'),
    ('91000000004', 'Comprovante alheio', 'x4@exemplo.invalid'),
    ('91000000005', 'Completo', 'x5@exemplo.invalid'),
    ('91000000006', 'Completo medula', 'x6@exemplo.invalid');
SELECT set_config('certamis.ator', '', false), set_config('certamis.motivo', '', false);
INSERT INTO documento (candidato_id, tipo, nome_original, tamanho_bytes, sha256, chave)
SELECT id, 'COMPROVANTE_ISENCAO', 'medula.pdf', 1000, repeat('a', 64), 'doc/' || cpf
  FROM candidato WHERE cpf IN ('91000000005', '91000000006');
INSERT INTO pedido_isencao (certame_id, candidato_id, hipotese, nis, comprovante_id)
SELECT 1, c.id, v.hip, v.nis, (SELECT d.id FROM documento d JOIN candidato o ON o.id = d.candidato_id WHERE o.cpf = v.doc)
  FROM (VALUES ('91000000001', 'CADUNICO', NULL, NULL),
               ('91000000002', 'CADUNICO', '12034567898', NULL),
               ('91000000003', 'DOADOR_MEDULA', NULL, NULL),
               ('91000000004', 'DOADOR_MEDULA', NULL, '91000000005'),
               ('91000000005', 'CADUNICO', '12034567899', NULL),
               ('91000000006', 'DOADOR_MEDULA', NULL, '91000000006')) AS v(cpf, hip, nis, doc)
  JOIN candidato c ON c.cpf = v.cpf;
DO $$
DECLARE r text;
BEGIN
    SELECT string_agg(c.cpf || ':' || p.estado || ':' || coalesce(p.motivo_codigo, '-'), ' ' ORDER BY c.cpf)
      INTO r FROM pedido_isencao p JOIN candidato c ON c.id = p.candidato_id;
    ASSERT r = '91000000001:INDEFERIDA:NIS_AUSENTE 91000000002:INDEFERIDA:NIS_INVALIDO '
               '91000000003:INDEFERIDA:COMPROVANTE_AUSENTE 91000000004:INDEFERIDA:COMPROVANTE_ALHEIO '
               '91000000005:PENDENTE_ANALISE:- 91000000006:PENDENTE_ANALISE:-', 'triagem: ' || r;
    -- A triagem também está na trilha, com o ator do sistema.
    ASSERT (SELECT count(*) FROM pedido_isencao_transicao WHERE ator = 'sistema:triagem') = 6;
END $$;

-- 8. A triagem não aceita o estado que vem de fora: tudo entra pela regra.
INSERT INTO candidato (cpf, nome, email) VALUES ('91000000007', 'Espertinho', 'x7@exemplo.invalid');
INSERT INTO pedido_isencao (certame_id, candidato_id, hipotese, estado)
VALUES (1, (SELECT id FROM candidato WHERE cpf = '91000000007'), 'CADUNICO', 'DEFERIDA');
SELECT pg_temp.confere((SELECT estado || ':' || motivo_codigo FROM pedido_isencao p JOIN candidato c ON c.id = p.candidato_id WHERE c.cpf = '91000000007') = 'INDEFERIDA:NIS_AUSENTE', '(SELECT estado || '':'' || motivo_codigo FROM pedido_isencao p JOIN candidato c ');

-- 9. Decisão humana exige ator e motivo; deferida não volta; dados não mudam.
SELECT pg_temp.espera_erro($q$UPDATE pedido_isencao SET estado = 'DEFERIDA'
    WHERE candidato_id = (SELECT id FROM candidato WHERE cpf = '91000000005')$q$, 'CT002');
SELECT pg_temp.ctx('banca:analista7', 'CadÚnico conferido');
UPDATE pedido_isencao SET estado = 'DEFERIDA' WHERE candidato_id = (SELECT id FROM candidato WHERE cpf = '91000000005');
SELECT pg_temp.espera_erro($q$UPDATE pedido_isencao SET estado = 'INDEFERIDA', motivo_codigo = 'ANALISE_HUMANA'
    WHERE candidato_id = (SELECT id FROM candidato WHERE cpf = '91000000005')$q$, 'CT009');
SELECT pg_temp.espera_erro($q$UPDATE pedido_isencao SET nis = '17000000013'
    WHERE candidato_id = (SELECT id FROM candidato WHERE cpf = '91000000005')$q$, 'CT004');

-- 10. Recurso provido: INDEFERIDA -> DEFERIDA, com motivo RECURSO_PROVIDO.
SELECT pg_temp.ctx('banca:comissao', 'comprovante reenviado em recurso');
UPDATE pedido_isencao SET estado = 'DEFERIDA' WHERE candidato_id = (SELECT id FROM candidato WHERE cpf = '91000000003');
DO $$ BEGIN
    ASSERT (SELECT motivo_codigo FROM pedido_isencao p JOIN candidato c ON c.id = p.candidato_id
             WHERE c.cpf = '91000000003') = 'RECURSO_PROVIDO';
END $$;

-- 11. Pedido fora do prazo de isenção é recusado.
UPDATE certame SET isencao_fim = now() - interval '1 minute' WHERE id = 1;
INSERT INTO candidato (cpf, nome, email) VALUES ('91000000008', 'Tarde', 'x8@exemplo.invalid');
SELECT pg_temp.espera_erro($q$INSERT INTO pedido_isencao (certame_id, candidato_id, hipotese, nis)
    VALUES (1, (SELECT id FROM candidato WHERE cpf = '91000000008'), 'CADUNICO', '12034567899')$q$, 'CT007');

-- 12. Isento deferido finaliza como isento e não pode finalizar como pagante.
SELECT pg_temp.ctx('candidato:91000000005', 'inscrição iniciada');
INSERT INTO inscricao (certame_id, candidato_id, cargo_id, cidade_id, modalidade_id, declaracoes_aceitas_em)
VALUES (1, (SELECT id FROM candidato WHERE cpf = '91000000005'), 1, 1, 1, now());
SELECT pg_temp.ctx('candidato:91000000005', 'finalizou');
SELECT pg_temp.espera_erro($q$UPDATE inscricao SET estado = 'AGUARDANDO_PAGAMENTO'
    WHERE candidato_id = (SELECT id FROM candidato WHERE cpf = '91000000005')$q$, 'CT008');
UPDATE inscricao SET estado = 'ISENCAO_ATIVA' WHERE candidato_id = (SELECT id FROM candidato WHERE cpf = '91000000005');

-- ============================================================ cobrança e pagamento

-- 13. Não há cobrança para inscrição que não aguarda pagamento.
SELECT pg_temp.espera_erro(format($q$SELECT gerar_cobranca(%s, 'chave-rascunho-1', 'PIX')$q$, :i3), 'CT010');

-- 14. Mesma chave devolve a mesma cobrança; outra chave também, enquanto houver uma pendente.
SELECT (gerar_cobranca(:i4, 'chave-aaaaaaaa', 'PIX')).id AS c1 \gset
SELECT (gerar_cobranca(:i4, 'chave-aaaaaaaa', 'PIX')).id AS c1b \gset
SELECT (gerar_cobranca(:i4, 'chave-bbbbbbbb', 'BOLETO')).id AS c1c \gset
SELECT pg_temp.confere(:c1 = :c1b AND :c1 = :c1c, ':c1 = :c1b AND :c1 = :c1c');
SELECT pg_temp.confere((SELECT count(*) FROM cobranca WHERE inscricao_id = :i4) = 1, '(SELECT count(*) FROM cobranca WHERE inscricao_id = :i4) = 1');
SELECT pg_temp.confere((SELECT valor_centavos FROM cobranca WHERE id = :c1) = 8000, '(SELECT valor_centavos FROM cobranca WHERE id = :c1) = 8000');
SELECT pg_temp.confere((SELECT referencia FROM cobranca WHERE id = :c1) = 'CTM' || lpad(:c1::text, 12, '0'), '(SELECT referencia FROM cobranca WHERE id = :c1) = ''CTM'' || lpad(:c1::text, 12');
SELECT referencia AS ref1 FROM cobranca WHERE id = :c1 \gset

-- 15. Valor divergente não aplica; referência desconhecida não aplica.
SELECT pg_temp.confere(confirmar_pagamento('sandbox', 'ev-div', :'ref1', 7999) = 'VALOR_DIVERGENTE', 'confirmar_pagamento(''sandbox'', ''ev-div'', :''ref1'', 7999) = ''VALOR_DIVERGEN');
SELECT pg_temp.confere(confirmar_pagamento('sandbox', 'ev-desc', 'CTM999999999999', 8000) = 'REFERENCIA_DESCONHECIDA', 'confirmar_pagamento(''sandbox'', ''ev-desc'', ''CTM999999999999'', 8000) = ''REF');
SELECT pg_temp.confere((SELECT status FROM cobranca WHERE id = :c1) = 'PENDENTE', '(SELECT status FROM cobranca WHERE id = :c1) = ''PENDENTE''');

-- 16. Confirmação aplica uma vez; o mesmo evento de novo é DUPLICADO; outro evento é COBRANCA_INATIVA.
SELECT pg_temp.confere(confirmar_pagamento('sandbox', 'ev-1', :'ref1', 8000) = 'APLICADO', 'confirmar_pagamento(''sandbox'', ''ev-1'', :''ref1'', 8000) = ''APLICADO''');
SELECT pg_temp.confere(confirmar_pagamento('sandbox', 'ev-1', :'ref1', 8000) = 'DUPLICADO', 'confirmar_pagamento(''sandbox'', ''ev-1'', :''ref1'', 8000) = ''DUPLICADO''');
SELECT pg_temp.confere(confirmar_pagamento('sandbox', 'ev-2', :'ref1', 8000) = 'COBRANCA_INATIVA', 'confirmar_pagamento(''sandbox'', ''ev-2'', :''ref1'', 8000) = ''COBRANCA_INATIVA');
SELECT pg_temp.confere((SELECT estado FROM inscricao WHERE id = :i4) = 'PAGA', '(SELECT estado FROM inscricao WHERE id = :i4) = ''PAGA''');
SELECT pg_temp.confere((SELECT status FROM cobranca WHERE id = :c1) = 'PAGA', '(SELECT status FROM cobranca WHERE id = :c1) = ''PAGA''');
SELECT pg_temp.confere((SELECT ator FROM inscricao_transicao WHERE inscricao_id = :i4 AND para = 'PAGA') = 'provedor:sandbox', '(SELECT ator FROM inscricao_transicao WHERE inscricao_id = :i4 AND para = ''PAGA');

-- 17. Depois de paga, não se gera outra cobrança.
SELECT pg_temp.espera_erro(format($q$SELECT gerar_cobranca(%s, 'chave-cccccccc', 'PIX')$q$, :i4), 'CT011');

-- 18. Voltar ao rascunho cancela a cobrança pendente; finalizar de novo gera outra.
SELECT pg_temp.inscrito('90000000005') AS i5 \gset
SELECT pg_temp.finaliza(:i5);
SELECT (gerar_cobranca(:i5, 'chave-i5-primeira', 'BOLETO')).id AS c5a \gset
SELECT pg_temp.ctx('candidato:90000000005', 'corrigir cidade');
UPDATE inscricao SET estado = 'RASCUNHO' WHERE id = :i5;
SELECT pg_temp.finaliza(:i5);
SELECT (gerar_cobranca(:i5, 'chave-i5-segunda', 'PIX')).id AS c5b \gset
SELECT pg_temp.confere((SELECT status FROM cobranca WHERE id = :c5a) = 'CANCELADA', '(SELECT status FROM cobranca WHERE id = :c5a) = ''CANCELADA''');
SELECT pg_temp.confere((SELECT status FROM cobranca WHERE id = :c5b) = 'PENDENTE' AND :c5a <> :c5b, '(SELECT status FROM cobranca WHERE id = :c5b) = ''PENDENTE'' AND :c5a <> :c5b');

-- 19. Boleto antigo pago depois do cancelamento: não aplica, não some, aparece na conciliação.
SELECT referencia AS ref5a FROM cobranca WHERE id = :c5a \gset
SELECT pg_temp.confere(confirmar_pagamento('sandbox', 'ev-boleto-antigo', :'ref5a', 8000) = 'COBRANCA_INATIVA', 'confirmar_pagamento(''sandbox'', ''ev-boleto-antigo'', :''ref5a'', 8000) = ''COB');
SELECT pg_temp.confere((SELECT estado FROM inscricao WHERE id = :i5) = 'AGUARDANDO_PAGAMENTO', '(SELECT estado FROM inscricao WHERE id = :i5) = ''AGUARDANDO_PAGAMENTO''');
SELECT pg_temp.confere(EXISTS (SELECT 1 FROM conciliacao WHERE tipo = 'EVENTO_NAO_APLICADO' AND detalhe = 'ev-boleto-antigo'), 'EXISTS (SELECT 1 FROM conciliacao WHERE tipo = ''EVENTO_NAO_APLICADO'' AND detal');

-- 20. Cancelar sem pagar cancela a cobrança.
SELECT pg_temp.ctx('candidato:90000000005', 'desistiu');
UPDATE inscricao SET estado = 'CANCELADA' WHERE id = :i5;
SELECT pg_temp.confere((SELECT status FROM cobranca WHERE id = :c5b) = 'CANCELADA', '(SELECT status FROM cobranca WHERE id = :c5b) = ''CANCELADA''');

-- 21. Prazo de pagamento vencido expira a inscrição e a cobrança, com trilha.
SELECT pg_temp.inscrito('90000000006') AS i6 \gset
SELECT pg_temp.finaliza(:i6);
SELECT (gerar_cobranca(:i6, 'chave-i6-aaaaaa', 'BOLETO')).id AS c6 \gset
UPDATE certame SET inscricao_fim = now() - interval '2 minutes', pagamento_fim = now() - interval '1 minute' WHERE id = 1;
SELECT pg_temp.confere(expirar_pagamentos_vencidos() = 1, 'expirar_pagamentos_vencidos() = 1');
SELECT pg_temp.confere((SELECT estado FROM inscricao WHERE id = :i6) = 'EXPIRADA', '(SELECT estado FROM inscricao WHERE id = :i6) = ''EXPIRADA''');
SELECT pg_temp.confere((SELECT status FROM cobranca WHERE id = :c6) = 'EXPIRADA', '(SELECT status FROM cobranca WHERE id = :c6) = ''EXPIRADA''');
SELECT pg_temp.confere((SELECT ator FROM inscricao_transicao WHERE inscricao_id = :i6 AND para = 'EXPIRADA') = 'sistema:expiracao', '(SELECT ator FROM inscricao_transicao WHERE inscricao_id = :i6 AND para = ''EXPI');
SELECT pg_temp.confere(expirar_pagamentos_vencidos() = 0, 'expirar_pagamentos_vencidos() = 0');

-- 22. Eventos de pagamento são imutáveis, e os invariantes da conciliação estão limpos.
SELECT pg_temp.espera_erro($q$UPDATE pagamento_evento SET resultado = 'APLICADO'$q$, 'CT003');
SELECT pg_temp.espera_erro($q$DELETE FROM pagamento_evento$q$, 'CT003');
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM conciliacao
                        WHERE tipo IN ('INSCRICAO_PAGA_SEM_COBRANCA_PAGA', 'MAIS_DE_UM_EVENTO_APLICADO'));
END $$;

SELECT 'BLOCO 2: TODAS AS REGRAS PASSARAM' AS resultado;
