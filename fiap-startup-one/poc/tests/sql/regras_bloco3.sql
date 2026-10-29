-- Bloco 3: fila de tarefas e portão da fila de espera.
\set ON_ERROR_STOP on
SET client_min_messages = warning;

CREATE FUNCTION pg_temp.confere(ok boolean, o_que text) RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    IF ok IS NOT TRUE THEN RAISE EXCEPTION 'falhou: %', o_que; END IF;
END $$;

-- ============================================================ tarefas

-- 1. Mesma tarefa não entra duas vezes.
INSERT INTO tarefa (tipo, chave) VALUES ('t', 'a'), ('t', 'b'), ('t', 'c'), ('outro', 'x');
INSERT INTO tarefa (tipo, chave) VALUES ('t', 'a') ON CONFLICT (tipo, chave) DO NOTHING;
SELECT pg_temp.confere((SELECT count(*) FROM tarefa WHERE tipo = 't') = 3, 'sem duplicata');

-- 2. Pegar respeita tipo e limite, e trava.
SELECT pg_temp.confere((SELECT count(*) FROM pegar_tarefas(ARRAY['t'], 2, 30)) = 2, 'pega 2');
SELECT pg_temp.confere((SELECT count(*) FROM pegar_tarefas(ARRAY['t'], 10, 30)) = 1, 'sobra 1');
SELECT pg_temp.confere((SELECT count(*) FROM pegar_tarefas(ARRAY['t'], 10, 30)) = 0, 'nada pronto');

-- 3. Trava vencida (worker morreu) devolve a tarefa.
UPDATE tarefa SET travada_ate = now() - interval '1 second' WHERE tipo = 't' AND chave = 'a';
SELECT pg_temp.confere((SELECT chave FROM pegar_tarefas(ARRAY['t'], 10, 30)) = 'a', 'recupera a travada');
SELECT pg_temp.confere((SELECT tentativas FROM tarefa WHERE chave = 'a') = 2, 'conta a tentativa');

-- 4. Concluir e falhar com recuo.
SELECT concluir_tarefa(id) FROM tarefa WHERE chave = 'b';
SELECT falhar_tarefa(id, 'provedor fora') FROM tarefa WHERE chave = 'c';
SELECT pg_temp.confere((SELECT estado FROM tarefa WHERE chave = 'b') = 'FEITA', 'feita');
SELECT pg_temp.confere((SELECT estado = 'PENDENTE' AND disponivel_em > now() AND ultimo_erro = 'provedor fora'
                          FROM tarefa WHERE chave = 'c'), 'volta com recuo');

-- 5. Esgotadas as tentativas, FALHOU.
UPDATE tarefa SET estado = 'EXECUTANDO', tentativas = max_tentativas WHERE chave = 'a';
SELECT falhar_tarefa(id, 'definitivo') FROM tarefa WHERE chave = 'a';
SELECT pg_temp.confere((SELECT estado FROM tarefa WHERE chave = 'a') = 'FALHOU', 'falhou de vez');

-- ============================================================ portão

-- 6. Fila vazia: o portão não acumula crédito.
UPDATE portao SET atualizado_em = clock_timestamp() - interval '100 seconds', taxa_por_s = 5;
SELECT avancar_portao();
SELECT pg_temp.confere((SELECT admitido_ate FROM portao) = 0, 'sem senha, nada admitido');
SELECT pg_temp.confere((SELECT atualizado_em > clock_timestamp() - interval '1 second' FROM portao), 'relógio acompanhou');

-- 7. 100 senhas emitidas, 10 segundos a 5/s: admite 50.
SELECT nextval('fila_senha_seq') FROM generate_series(1, 100);
UPDATE portao SET atualizado_em = clock_timestamp() - interval '10 seconds';
SELECT avancar_portao();
SELECT pg_temp.confere((SELECT admitido_ate FROM portao) = 50,
                       'admitiu ' || (SELECT admitido_ate FROM portao));

-- 8. Nunca passa da última senha emitida.
UPDATE portao SET atualizado_em = clock_timestamp() - interval '1000 seconds';
SELECT avancar_portao();
SELECT pg_temp.confere((SELECT admitido_ate FROM portao) = 100, 'para na última senha');

-- 9. Mudança de nível fica no histórico.
UPDATE modo_operacao SET nivel = 1, mensagem = 'pico', atualizado_por = 'plantao:ana';
SELECT pg_temp.confere((SELECT count(*) FROM modo_operacao_historico WHERE por = 'plantao:ana' AND nivel = 1) = 1,
                       'histórico');

SELECT 'BLOCO 3: TODAS AS REGRAS PASSARAM' AS resultado;
