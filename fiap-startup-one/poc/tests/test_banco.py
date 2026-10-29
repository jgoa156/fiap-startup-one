"""Regras que moram no banco: transição, trilha, imutabilidade."""

from pathlib import Path

from app.inscricao.estados import TRANSICOES

SQL = Path(__file__).parent / "sql" / "regras_do_banco.sql"


def test_regras_do_banco(banco_vazio):
    # Cada uma das 18 verificações do arquivo aborta o script se falhar.
    banco_vazio.script(SQL.read_text(encoding="utf-8"))


def test_tabela_do_banco_igual_a_do_codigo(banco_vazio):
    no_banco = {(de, para, ev) for de, para, ev in
                banco_vazio.linhas("SELECT de, para, evento FROM transicao_permitida")}
    no_codigo = {(de.value, para.value, ev) for (de, para), ev in TRANSICOES.items()}
    assert no_banco == no_codigo


def test_estados_finais_do_banco_iguais_aos_do_codigo(banco_vazio):
    from app.inscricao.estados import FINAIS

    finais = {r[0] for r in banco_vazio.linhas(
        "SELECT codigo FROM estado_inscricao WHERE final")}
    assert finais == {e.value for e in FINAIS}
