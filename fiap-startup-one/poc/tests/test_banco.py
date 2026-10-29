"""Regras que moram no banco: transição, trilha, prazos, isenção, cobrança."""

from pathlib import Path

from app.inscricao.estados import FINAIS, TRANSICOES

SQL = Path(__file__).parent / "sql"


def test_regras_do_bloco_1(banco_vazio):
    # 18 verificações; qualquer falha aborta o arquivo.
    assert "TODAS AS REGRAS DO BANCO PASSARAM" in banco_vazio.arquivo(SQL / "regras_do_banco.sql")


def test_regras_do_bloco_2(banco_vazio):
    # 22 verificações: prazos, formulário, triagem de isenção, cobrança, pagamento, expiração.
    assert "BLOCO 2: TODAS AS REGRAS PASSARAM" in banco_vazio.arquivo(SQL / "regras_bloco2.sql")


def test_tabela_do_banco_igual_a_do_codigo(banco_vazio):
    no_banco = set(banco_vazio.linhas("SELECT de, para, evento FROM transicao_permitida"))
    no_codigo = {(de.value, para.value, ev) for (de, para), ev in TRANSICOES.items()}
    assert no_banco == no_codigo


def test_estados_finais_do_banco_iguais_aos_do_codigo(banco_vazio):
    finais = {r[0] for r in banco_vazio.linhas("SELECT codigo FROM estado_inscricao WHERE final")}
    assert finais == {e.value for e in FINAIS}


def test_regras_do_bloco_3(banco_vazio):
    # Fila de tarefas (pega, trava, recupera, recua, falha) e portão da fila de espera.
    assert "BLOCO 3: TODAS AS REGRAS PASSARAM" in banco_vazio.arquivo(SQL / "regras_bloco3.sql")
