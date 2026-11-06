// Package auditoria confere, no banco, o que o ensaio de carga afirma: nenhuma inscrição perdida e
// nenhuma cobrança duplicada. O k6 conta o que o candidato VIU (inscrição enviada, pagamento
// confirmado); a auditoria conta o que o banco GUARDOU. Se o banco tem menos do que o candidato
// viu, houve perda.
//
// Os candidatos de uma rodada do ensaio têm e-mail ensaio.<rodada>.<n>@lisura.test.
package auditoria

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/inscricao"
	"lisura/internal/pagamento"
)

type Relatorio struct {
	Rodada     string         `json:"rodada"`
	Candidatos int            `json:"candidatos"`
	Inscricoes map[string]int `json:"inscricoes_por_estado"`
	Enviadas   int            `json:"inscricoes_enviadas"` // saíram do rascunho
	Pagas      int            `json:"inscricoes_pagas"`
	Cobrancas  map[string]int `json:"cobrancas_por_estado"`

	// Duas cobranças do mesmo método para a mesma inscrição: o sistema gerou duas vezes o que o
	// candidato pediu uma (duplo clique, repetição, resposta perdida). Tem de ser zero.
	CobrancasRepetidas int `json:"cobrancas_repetidas"`
	// Um pagamento do prestador aplicado mais de uma vez. A chave primária impede; conferido mesmo assim.
	PagamentosAplicadosDuasVezes int `json:"pagamentos_aplicados_duas_vezes"`
	// Inscrição marcada PAGA sem um pagamento recebido que a sustente.
	PagasSemRecebimento int `json:"pagas_sem_recebimento"`
	// Dinheiro que entrou a mais (o candidato pagou duas vezes): não é erro do sistema, desde que
	// esteja marcado para devolver.
	ReembolsosPendentes int `json:"reembolsos_pendentes"`
	// Inscrição com dois pagamentos confirmados (Pix e boleto, por exemplo, numa corrida): o sistema
	// ficaria com o dinheiro duas vezes, sem devolver nenhum. É a cobrança dupla que o candidato sente.
	ConfirmadasDuasVezes int `json:"inscricoes_confirmadas_duas_vezes"`
	// Pagamento classificado como a mais (duplicidade, depois da expiração) sem devolução registrada:
	// o dinheiro ficaria aqui.
	PagoAMaisSemReembolso int `json:"pago_a_mais_sem_reembolso"`
	// O caminho de volta, por estado (PENDENTE, AGUARDANDO_CHAVE, CONCLUIDA, RECUSADA).
	Devolucoes map[string]int `json:"devolucoes_por_estado"`
	// Devolução de um pagamento que confirmou a inscrição: o candidato perderia o dinheiro e ficaria
	// com a inscrição, ou o órgão ficaria sem a taxa. Tem de ser zero.
	DevolucoesDePagamentoValido int `json:"devolucoes_de_pagamento_valido"`
	// Devolução maior que o pagamento que a originou.
	DevolucoesAcimaDoPago int `json:"devolucoes_acima_do_pago"`
	// Em certame cancelado: inscrição que continua valendo, e pagamento confirmado sem devolução.
	VivasEmCertameCancelado   int `json:"inscricoes_vivas_em_certame_cancelado"`
	RetidosEmCertameCancelado int `json:"pagamentos_retidos_em_certame_cancelado"`
	Divergencias              int `json:"divergencias_da_conciliacao"`
	// Inscrição valendo cujo cartão ainda não foi publicado pela fila.
	CartoesPendentes int `json:"cartoes_pendentes"`
	TarefasFalhas    int `json:"tarefas_falhas"`
	TarefasPendentes int `json:"tarefas_pendentes"`
	// Trilhas da rodada cuja corrente de resumos não confere (evento alterado ou apagado por fora da
	// aplicação). Refeita em Go, evento por evento.
	TrilhasVerificadas int     `json:"trilhas_verificadas"`
	TrilhasQuebradas   []int64 `json:"trilhas_quebradas"`

	Problemas []string `json:"problemas"`
}

// Esperado é o que o k6 contou do lado do candidato. Zero = não conferir.
type Esperado struct {
	Enviadas int // finalizar respondeu 200
	Pagas    int // a tela mostrou PAGA
}

const daRodada = `c.email LIKE 'ensaio.' || $1 || '.%@lisura.test'`

func Auditar(ctx context.Context, pool *pgxpool.Pool, rodada string, esp Esperado) (Relatorio, error) {
	r := Relatorio{Rodada: rodada, Inscricoes: map[string]int{}, Cobrancas: map[string]int{}, Devolucoes: map[string]int{}}
	if rodada == "" {
		return r, fmt.Errorf("rodada vazia: a auditoria conferiria o banco inteiro")
	}

	contar := func(destino *int, sql string, args ...any) error {
		return pool.QueryRow(ctx, sql, args...).Scan(destino)
	}
	porEstado := func(destino map[string]int, sql string) error {
		rows, err := pool.Query(ctx, sql, rodada)
		if err != nil {
			return err
		}
		var e string
		var n int
		_, err = pgx.ForEachRow(rows, []any{&e, &n}, func() error {
			destino[e] = n
			return nil
		})
		return err
	}

	passos := []func() error{
		func() error {
			return contar(&r.Candidatos, `SELECT count(*) FROM candidato c WHERE `+daRodada, rodada)
		},
		func() error {
			return porEstado(r.Inscricoes, `
				SELECT i.estado, count(*) FROM inscricao i JOIN candidato c ON c.id = i.candidato_id
				WHERE `+daRodada+` GROUP BY i.estado`)
		},
		func() error {
			return porEstado(r.Cobrancas, `
				SELECT b.estado, count(*) FROM cobranca b JOIN inscricao i ON i.id = b.inscricao_id
				JOIN candidato c ON c.id = i.candidato_id WHERE `+daRodada+` GROUP BY b.estado`)
		},
		func() error {
			return contar(&r.CobrancasRepetidas, `
				SELECT count(*) FROM (
					SELECT b.inscricao_id, b.metodo FROM cobranca b JOIN inscricao i ON i.id = b.inscricao_id
					JOIN candidato c ON c.id = i.candidato_id WHERE `+daRodada+`
					GROUP BY b.inscricao_id, b.metodo HAVING count(*) > 1) x`, rodada)
		},
		func() error {
			// Mesmo pagamento do prestador apontado por mais de uma linha (impossível pela chave
			// primária; esta consulta é a prova, não a defesa).
			return contar(&r.PagamentosAplicadosDuasVezes, `
				SELECT count(*) FROM (
					SELECT p.psp_pagamento_id FROM pagamento_recebido p JOIN cobranca b ON b.id = p.cobranca_id
					JOIN inscricao i ON i.id = b.inscricao_id JOIN candidato c ON c.id = i.candidato_id
					WHERE `+daRodada+` GROUP BY p.psp_pagamento_id HAVING count(*) > 1) x`, rodada)
		},
		func() error {
			return contar(&r.PagasSemRecebimento, `
				SELECT count(*) FROM inscricao i JOIN candidato c ON c.id = i.candidato_id
				WHERE `+daRodada+` AND i.estado = 'PAGA' AND NOT EXISTS (
					SELECT 1 FROM cobranca b JOIN pagamento_recebido p ON p.cobranca_id = b.id
					WHERE b.inscricao_id = i.id AND b.estado = 'PAGA')`, rodada)
		},
		func() error {
			return contar(&r.ReembolsosPendentes, `
				SELECT count(*) FROM cobranca b JOIN inscricao i ON i.id = b.inscricao_id
				JOIN candidato c ON c.id = i.candidato_id WHERE `+daRodada+` AND b.reembolso_pendente`, rodada)
		},
		func() error {
			return contar(&r.ConfirmadasDuasVezes, `
				SELECT count(*) FROM (
					SELECT b.inscricao_id FROM pagamento_recebido p JOIN cobranca b ON b.id = p.cobranca_id
					JOIN inscricao i ON i.id = b.inscricao_id JOIN candidato c ON c.id = i.candidato_id
					WHERE `+daRodada+` AND p.resultado = $2
					GROUP BY b.inscricao_id HAVING count(*) > 1) x`, rodada, pagamento.Confirmado)
		},
		func() error {
			return contar(&r.PagoAMaisSemReembolso, `
				SELECT count(*) FROM pagamento_recebido p JOIN cobranca b ON b.id = p.cobranca_id
				JOIN inscricao i ON i.id = b.inscricao_id JOIN candidato c ON c.id = i.candidato_id
				WHERE `+daRodada+` AND p.resultado IN ($2, $3)
				AND NOT EXISTS (SELECT 1 FROM devolucao d WHERE d.psp_pagamento_id = p.psp_pagamento_id)`,
				rodada, pagamento.EmDuplicidade, pagamento.AposExpiracao)
		},
		func() error {
			return porEstado(r.Devolucoes, `
				SELECT d.estado, count(*) FROM devolucao d JOIN inscricao i ON i.id = d.inscricao_id
				JOIN candidato c ON c.id = i.candidato_id WHERE `+daRodada+` GROUP BY d.estado`)
		},
		func() error {
			return contar(&r.DevolucoesDePagamentoValido, `
				SELECT count(*) FROM devolucao d JOIN pagamento_recebido p ON p.psp_pagamento_id = d.psp_pagamento_id
				JOIN inscricao i ON i.id = d.inscricao_id JOIN candidato c ON c.id = i.candidato_id
				WHERE `+daRodada+` AND p.resultado = $2 AND d.motivo <> $3`, rodada, pagamento.Confirmado, pagamento.MotivoCertameCancelado)
		},
		func() error {
			return contar(&r.VivasEmCertameCancelado, `
				SELECT count(*) FROM inscricao i JOIN candidato c ON c.id = i.candidato_id
				JOIN certame k ON k.id = i.certame_id
				WHERE `+daRodada+` AND k.cancelado_em IS NOT NULL AND i.estado NOT IN ('CANCELADA', 'EXPIRADA')`, rodada)
		},
		func() error {
			return contar(&r.RetidosEmCertameCancelado, `
				SELECT count(*) FROM pagamento_recebido p JOIN cobranca b ON b.id = p.cobranca_id
				JOIN inscricao i ON i.id = b.inscricao_id JOIN candidato c ON c.id = i.candidato_id
				JOIN certame k ON k.id = i.certame_id
				WHERE `+daRodada+` AND k.cancelado_em IS NOT NULL AND p.resultado = $2
				AND NOT EXISTS (SELECT 1 FROM devolucao d WHERE d.psp_pagamento_id = p.psp_pagamento_id)`,
				rodada, pagamento.Confirmado)
		},
		func() error {
			return contar(&r.DevolucoesAcimaDoPago, `
				SELECT count(*) FROM devolucao d JOIN pagamento_recebido p ON p.psp_pagamento_id = d.psp_pagamento_id
				JOIN inscricao i ON i.id = d.inscricao_id JOIN candidato c ON c.id = i.candidato_id
				WHERE `+daRodada+` AND d.valor_centavos > p.valor_centavos`, rodada)
		},
		func() error {
			return contar(&r.Divergencias, `
				SELECT count(*) FROM divergencia d JOIN cobranca b ON b.id = d.cobranca_id
				JOIN inscricao i ON i.id = b.inscricao_id JOIN candidato c ON c.id = i.candidato_id
				WHERE `+daRodada, rodada)
		},
		func() error {
			return contar(&r.CartoesPendentes, `
				SELECT count(*) FROM inscricao i JOIN candidato c ON c.id = i.candidato_id
				WHERE `+daRodada+` AND i.estado IN ('PAGA','ISENCAO_ATIVA','EM_ANALISE','DEFERIDA','HOMOLOGADA')
				AND NOT EXISTS (SELECT 1 FROM tarefa t WHERE t.tipo = 'publicar_cartao'
				                AND t.chave = i.id::text AND t.estado = 'CONCLUIDA')`, rodada)
		},
		func() error {
			return contar(&r.TarefasFalhas, `SELECT count(*) FROM tarefa WHERE estado = 'FALHOU'`)
		},
		func() error {
			return contar(&r.TarefasPendentes, `SELECT count(*) FROM tarefa WHERE estado = 'PENDENTE'`)
		},
		func() error {
			rows, err := pool.Query(ctx, `SELECT i.id FROM inscricao i JOIN candidato c ON c.id = i.candidato_id WHERE `+daRodada, rodada)
			if err != nil {
				return err
			}
			ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
			if err != nil || len(ids) == 0 {
				return err
			}
			quebras, selos, err := inscricao.VerificarTrilhas(ctx, pool, ids)
			r.TrilhasVerificadas = len(selos)
			for _, q := range quebras {
				r.TrilhasQuebradas = append(r.TrilhasQuebradas, q.Inscricao)
			}
			return err
		},
	}
	for _, p := range passos {
		if err := p(); err != nil {
			return r, err
		}
	}
	for e, n := range r.Inscricoes {
		if e != "RASCUNHO" {
			r.Enviadas += n
		}
	}
	r.Pagas = r.Inscricoes["PAGA"]

	problema := func(cond bool, f string, a ...any) {
		if cond {
			r.Problemas = append(r.Problemas, fmt.Sprintf(f, a...))
		}
	}
	problema(esp.Enviadas > 0 && r.Enviadas < esp.Enviadas,
		"INSCRIÇÃO PERDIDA: o candidato viu %d enviadas, o banco tem %d", esp.Enviadas, r.Enviadas)
	problema(esp.Pagas > 0 && r.Pagas < esp.Pagas,
		"PAGAMENTO PERDIDO: o candidato viu %d pagas, o banco tem %d", esp.Pagas, r.Pagas)
	problema(r.CobrancasRepetidas > 0, "COBRANÇA DUPLICADA: %d inscrições com duas cobranças do mesmo método", r.CobrancasRepetidas)
	problema(r.PagamentosAplicadosDuasVezes > 0, "PAGAMENTO APLICADO DUAS VEZES: %d", r.PagamentosAplicadosDuasVezes)
	problema(r.ConfirmadasDuasVezes > 0, "COBRADA DUAS VEZES: %d inscrições com dois pagamentos confirmados", r.ConfirmadasDuasVezes)
	problema(r.PagoAMaisSemReembolso > 0, "PAGO A MAIS SEM DEVOLUÇÃO: %d pagamentos", r.PagoAMaisSemReembolso)
	problema(r.DevolucoesDePagamentoValido > 0, "DEVOLUÇÃO DE PAGAMENTO VÁLIDO: %d", r.DevolucoesDePagamentoValido)
	problema(r.DevolucoesAcimaDoPago > 0, "DEVOLUÇÃO ACIMA DO PAGO: %d", r.DevolucoesAcimaDoPago)
	problema(r.VivasEmCertameCancelado > 0, "INSCRIÇÃO VIVA EM CERTAME CANCELADO: %d", r.VivasEmCertameCancelado)
	problema(r.RetidosEmCertameCancelado > 0, "PAGAMENTO RETIDO EM CERTAME CANCELADO: %d sem devolução", r.RetidosEmCertameCancelado)
	problema(r.Devolucoes[pagamento.DevolucaoRecusada] > 0,
		"DEVOLUÇÃO RECUSADA PELO PRESTADOR: %d (ver ultimo_erro na tabela devolucao)", r.Devolucoes[pagamento.DevolucaoRecusada])
	problema(r.PagasSemRecebimento > 0, "PAGA SEM RECEBIMENTO: %d inscrições", r.PagasSemRecebimento)
	problema(len(r.TrilhasQuebradas) > 0, "TRILHA ADULTERADA: %d inscrições com a corrente quebrada (%v)",
		len(r.TrilhasQuebradas), r.TrilhasQuebradas)
	problema(r.TarefasFalhas > 0, "TAREFAS FALHAS: %d (ver ultimo_erro na tabela tarefa)", r.TarefasFalhas)
	return r, nil
}
