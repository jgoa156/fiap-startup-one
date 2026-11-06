package pagamento

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// registrar é RegistrarPagamento; trocável no teste, que conta quantas vezes a conciliação o chama.
var registrar = RegistrarPagamento

// JanelaSemanal é a varredura mais larga da conciliação: tem de cobrir a MargemCompensacao (3 dias
// úteis) mais um fim de semana. Se fosse menor, um boleto pago na quinta, com o aviso perdido, nunca
// apareceria numa conciliação, e a inscrição expiraria paga. (Até 05/11 a mais larga era de 48 h.)
const JanelaSemanal = 7 * 24 * time.Hour

// JanelaDaConciliacao diz quanto extrato a rodada n baixa (uma rodada por minuto, a partir de 0).
// A cada minuto, os últimos 10 min: o aviso perdido agora é achado na rodada seguinte. A cada 10 min,
// 2 h; a cada hora, 48 h; uma vez por dia, a semana. Baixar 2 h a cada minuto custava o volume de
// 2 h por minuto: medido em 05/11, a 1.280× local, 43 mil pagamentos e 7 a 17 s por rodada, com a
// máquina disputada. Assim, cada minuto custa o volume de 10 min, e nada fica sem conferência.
func JanelaDaConciliacao(n int) time.Duration {
	switch {
	case n%1440 == 0:
		return JanelaSemanal
	case n%60 == 0:
		return 48 * time.Hour
	case n%10 == 0:
		return 2 * time.Hour
	default:
		return 10 * time.Minute
	}
}

// ConciliarSozinho concilia só se nenhuma outra instância estiver conciliando agora (trava
// consultiva do Postgres). Com N instâncias da API, o trabalho é feito uma vez por rodada, não N.
// executou = false quando outra instância já estava nisso.
func ConciliarSozinho(ctx context.Context, pool *pgxpool.Pool, prestador Prestador, desde, ate time.Time) (rel Relatorio, executou bool, err error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return rel, false, err
	}
	defer conn.Release()
	const chave = 0x4c697375 // "Lisu": identifica a trava da conciliação
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, chave).Scan(&executou); err != nil || !executou {
		return rel, false, err
	}
	defer conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, chave)
	rel, err = Conciliar(ctx, pool, prestador, desde, ate)
	return rel, true, err
}

type Relatorio struct {
	NoExtrato        int            // pagamentos que o prestador diz ter recebido
	PorResultado     map[string]int // o que aconteceu com cada um deles aqui
	SemRegistroNoPSP int            // confirmados aqui que o prestador não conhece
}

// Conciliar confere o extrato do prestador contra o nosso banco, nos dois sentidos:
//   - pagamento no extrato que não chegou por aviso (aviso perdido) é aplicado agora, pelo mesmo
//     caminho do aviso — então rodar de novo não aplica nada duas vezes;
//   - pagamento confirmado aqui que não aparece no extrato vira divergência para alguém olhar.
//
// O que já foi registrado não passa de novo por RegistrarPagamento: daria JaProcessado do mesmo
// jeito, mas custava uma transação por pagamento, a cada minuto — num dia de 100 mil pagamentos,
// 100 mil transações por minuto disputando o banco com as inscrições. Uma consulta só separa os
// que faltam.
func Conciliar(ctx context.Context, pool *pgxpool.Pool, prestador Prestador, desde, ate time.Time) (Relatorio, error) {
	rel := Relatorio{PorResultado: map[string]int{}}
	extrato, err := prestador.Extrato(ctx, desde, ate)
	if err != nil {
		return rel, err
	}
	ids := make([]string, len(extrato))
	for i, p := range extrato {
		ids[i] = p.ID
	}
	rows, err := pool.Query(ctx, `SELECT psp_pagamento_id FROM pagamento_recebido WHERE psp_pagamento_id = ANY($1)`, ids)
	if err != nil {
		return rel, err
	}
	jaRegistrados, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return rel, err
	}
	conhecido := make(map[string]bool, len(jaRegistrados))
	for _, id := range jaRegistrados {
		conhecido[id] = true
	}

	noPSP := make(map[string]bool, len(extrato))
	for _, p := range extrato {
		noPSP[p.ID] = true
		if conhecido[p.ID] {
			rel.PorResultado[JaProcessado]++
			continue
		}
		r, err := registrar(ctx, pool, p, "conciliacao")
		if err != nil {
			return rel, err
		}
		rel.PorResultado[r]++
	}
	rel.NoExtrato = len(extrato)

	rows, err = pool.Query(ctx, `
		SELECT psp_pagamento_id, cobranca_id FROM pagamento_recebido
		WHERE resultado = $1 AND pago_em BETWEEN $2 AND $3`, Confirmado, desde, ate)
	if err != nil {
		return rel, err
	}
	type confirmado struct {
		PagamentoID string
		CobrancaID  *int64
	}
	confirmados, err := pgx.CollectRows(rows, pgx.RowToStructByPos[confirmado])
	if err != nil {
		return rel, err
	}
	for _, c := range confirmados {
		if !noPSP[c.PagamentoID] {
			rel.SemRegistroNoPSP++
			if err := divergir(ctx, pool, "PAGA_SEM_REGISTRO_NO_PSP", c.PagamentoID, c.CobrancaID,
				"confirmado aqui, ausente do extrato do prestador"); err != nil {
				return rel, err
			}
		}
	}
	return rel, nil
}
