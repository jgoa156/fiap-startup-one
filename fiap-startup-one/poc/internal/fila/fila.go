// Package fila é a fila de tarefas assíncronas da POC: o papel do SQS + Lambda da arquitetura alvo,
// feito numa tabela do Postgres. A garantia é a mesma — cada tarefa é executada por um trabalhador
// de cada vez, repetida com espera crescente se falhar, e abandonada (FALHOU) depois de N tentativas.
package fila

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Prioridade decide quem o trabalhador pega primeiro entre as tarefas vencidas. É de quem enfileira,
// não do tipo: o mesmo cartão é normal quando o candidato acabou de pagar e lote quando a homologação
// republica 100 mil de uma vez.
type Prioridade int16

const (
	Urgente Prioridade = 0 // o candidato espera olhando a tela (o Pix)
	Normal  Prioridade = 1
	Lote    Prioridade = 2 // trabalho em massa: roda quando não há nada mais urgente vencido
)

// Enfileirar registra a tarefa. Chamado dentro da transação que mudou o estado, a tarefa nasce junto
// com a mudança ou não nasce. A mesma (tipo, chave) enfileirada de novo não duplica: reabre a tarefa
// se ela já tinha terminado e sobe a versão se ainda está pendente.
func Enfileirar(ctx context.Context, q executor, tipo, chave string, carga any) error {
	return enfileirar(ctx, q, tipo, chave, carga, 0, Normal)
}

// EnfileirarCom é o Enfileirar com prioridade escolhida.
func EnfileirarCom(ctx context.Context, q executor, p Prioridade, tipo, chave string, carga any) error {
	return enfileirar(ctx, q, tipo, chave, carga, 0, p)
}

// EnfileirarDepois agenda a tarefa para daqui a `atraso` e junta nela tudo o que chegar até lá: é
// para trabalho que só precisa do estado mais recente (republicar a relação de inscritos). Uma
// chamada nova NÃO adia uma tarefa já pendente — senão, sob mudanças contínuas, ela nunca rodaria;
// o atraso só vale quando a tarefa nasce ou já tinha terminado.
func EnfileirarDepois(ctx context.Context, q executor, tipo, chave string, carga any, atraso time.Duration) error {
	return enfileirar(ctx, q, tipo, chave, carga, atraso, Normal)
}

// Reenfileirar uma tarefa pendente com prioridade mais alta a promove; com mais baixa, não a rebaixa
// (quem pediu com pressa continua com pressa).
func enfileirar(ctx context.Context, q executor, tipo, chave string, carga any, atraso time.Duration, p Prioridade) error {
	b, err := json.Marshal(carga)
	if err != nil {
		return err
	}
	// clock_timestamp, não now(): now() é a hora em que a transação COMEÇOU. Uma tarefa enfileirada no
	// fim de uma transação longa (o cálculo do resultado, 10 s com 200 mil inscrições) nasceria
	// "atrasada", e a medição de espera na fila contaria a transação como fila.
	_, err = q.Exec(ctx, `
		INSERT INTO tarefa (tipo, chave, carga, agrupar_s, executar_em, prioridade)
		VALUES ($1, $2, $3, $4, clock_timestamp() + $4::float8 * interval '1 second', $5)
		ON CONFLICT (tipo, chave) DO UPDATE SET
			carga = EXCLUDED.carga,
			agrupar_s = EXCLUDED.agrupar_s,
			prioridade = CASE WHEN tarefa.estado = 'PENDENTE' THEN least(tarefa.prioridade, EXCLUDED.prioridade)
			                  ELSE EXCLUDED.prioridade END,
			versao = tarefa.versao + 1,
			tentativas = CASE WHEN tarefa.estado = 'PENDENTE' THEN tarefa.tentativas ELSE 0 END,
			executar_em = CASE WHEN tarefa.estado = 'PENDENTE' THEN tarefa.executar_em ELSE EXCLUDED.executar_em END,
			estado = 'PENDENTE',
			concluida_em = NULL`, tipo, chave, b, atraso.Seconds(), int16(p))
	return err
}

// EnfileirarMuitos é o Enfileirar de várias tarefas do mesmo tipo numa instrução só, com as mesmas
// regras (a mesma chave não duplica: reabre a terminada, sobe a versão da pendente, promove e nunca
// rebaixa). Para trabalho em massa: 100 mil cartões republicados depois do ensalamento seriam 100 mil
// idas ao banco uma por uma.
func EnfileirarMuitos(ctx context.Context, q executor, p Prioridade, tipo string, chaves []string, cargas []any) error {
	if len(chaves) != len(cargas) {
		return fmt.Errorf("fila: %d chaves e %d cargas", len(chaves), len(cargas))
	}
	if len(chaves) == 0 {
		return nil
	}
	// A mesma chave duas vezes numa instrução só o Postgres recusa ("ON CONFLICT DO UPDATE command
	// cannot affect row a second time"): fica a última, como ficaria enfileirando uma por uma.
	pos := make(map[string]int, len(chaves))
	var ks, bs []string
	for i, k := range chaves {
		b, err := json.Marshal(cargas[i])
		if err != nil {
			return err
		}
		if j, ok := pos[k]; ok {
			bs[j] = string(b)
			continue
		}
		pos[k] = len(ks)
		ks = append(ks, k)
		bs = append(bs, string(b))
	}
	chaves = ks
	_, err := q.Exec(ctx, `
		INSERT INTO tarefa (tipo, chave, carga, agrupar_s, executar_em, prioridade)
		SELECT $1, k, c::jsonb, 0, clock_timestamp(), $4 FROM unnest($2::text[], $3::text[]) AS t(k, c)
		ON CONFLICT (tipo, chave) DO UPDATE SET
			carga = EXCLUDED.carga,
			agrupar_s = EXCLUDED.agrupar_s,
			prioridade = CASE WHEN tarefa.estado = 'PENDENTE' THEN least(tarefa.prioridade, EXCLUDED.prioridade)
			                  ELSE EXCLUDED.prioridade END,
			versao = tarefa.versao + 1,
			tentativas = CASE WHEN tarefa.estado = 'PENDENTE' THEN tarefa.tentativas ELSE 0 END,
			executar_em = CASE WHEN tarefa.estado = 'PENDENTE' THEN tarefa.executar_em ELSE EXCLUDED.executar_em END,
			estado = 'PENDENTE',
			concluida_em = NULL`, tipo, chaves, bs, int16(p))
	return err
}

// Manipulador executa uma tarefa. Precisa ser idempotente: pode rodar mais de uma vez (o trabalhador
// caiu no meio, a tarefa foi reenfileirada).
type Manipulador func(ctx context.Context, carga json.RawMessage) error

type Trabalhador struct {
	Pool          *pgxpool.Pool
	Manipuladores map[string]Manipulador
	MaxTentativas int           // padrão 5
	Prazo         time.Duration // quanto tempo a tarefa fica reservada para quem a pegou; padrão 1 min
	Espera        time.Duration // pausa quando a fila está vazia; padrão 500 ms
	// Observar, se dado, recebe cada tarefa executada (para as métricas do painel em tempo real).
	Observar func(tipo string, espera, execucao time.Duration, falhou bool)

	mu      sync.Mutex
	medidas map[string]*Medida
	rodando atomic.Int64
	ativos  atomic.Int64
}

// Rodando é quantas tarefas estão em execução agora; Ativos, quantos trabalhadores estão de pé.
func (t *Trabalhador) Rodando() int64 { return t.rodando.Load() }
func (t *Trabalhador) Ativos() int64  { return t.ativos.Load() }

// Medida resume, por tipo de tarefa, o que aconteceu desde a última leitura: quanto as tarefas
// esperaram na fila (falta de trabalhador) e quanto levaram para rodar (tarefa lenta). Os dois
// atrasam o candidato, e cada um tem um remédio diferente.
type Medida struct {
	Tarefas, Falhas            int
	EsperaMedia, EsperaMax     time.Duration
	ExecucaoMedia, ExecucaoMax time.Duration
}

func (t *Trabalhador) anotar(tipo string, espera, execucao time.Duration, falhou bool) {
	if t.Observar != nil {
		t.Observar(tipo, espera, execucao, falhou)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.medidas == nil {
		t.medidas = map[string]*Medida{}
	}
	m := t.medidas[tipo]
	if m == nil {
		m = &Medida{}
		t.medidas[tipo] = m
	}
	m.Tarefas++
	if falhou {
		m.Falhas++
	}
	m.EsperaMedia += espera // soma; vira média na leitura
	m.EsperaMax = max(m.EsperaMax, espera)
	m.ExecucaoMedia += execucao
	m.ExecucaoMax = max(m.ExecucaoMax, execucao)
}

// Medidas devolve o resumo por tipo desde a última chamada e zera a contagem.
func (t *Trabalhador) Medidas() map[string]Medida {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[string]Medida, len(t.medidas))
	for tipo, m := range t.medidas {
		r := *m
		r.EsperaMedia /= time.Duration(m.Tarefas)
		r.ExecucaoMedia /= time.Duration(m.Tarefas)
		out[tipo] = r
	}
	t.medidas = nil
	return out
}

var ErrSemManipulador = errors.New("tarefa sem manipulador")

// ProcessarUma pega a próxima tarefa vencida e a executa. Devolve false quando a fila está vazia.
// "Próxima" é a de prioridade mais alta e, entre essas, a vencida há mais tempo.
//
// A tarefa é reservada (executar_em empurrado para o futuro) numa transação curta e executada fora
// dela: nenhuma conexão do banco fica presa esperando o PSP. Se o trabalhador morrer, a reserva
// vence e outro pega.
func (t *Trabalhador) ProcessarUma(ctx context.Context) (bool, error) {
	var id int64
	var tipo string
	var carga json.RawMessage
	var tentativas, versao int
	var esperou float64 // segundos entre a hora marcada e a reserva: quanto a tarefa ficou na fila
	err := t.Pool.QueryRow(ctx, `
		WITH proxima AS (
			SELECT id, executar_em FROM tarefa WHERE estado = 'PENDENTE' AND executar_em <= now()
			ORDER BY prioridade, executar_em FOR UPDATE SKIP LOCKED LIMIT 1)
		UPDATE tarefa t SET tentativas = t.tentativas + 1, executar_em = now() + $1::float8 * interval '1 second'
		FROM proxima WHERE t.id = proxima.id
		RETURNING t.id, t.tipo, t.carga, t.tentativas, t.versao, extract(epoch FROM now() - proxima.executar_em)::float8`,
		t.prazo().Seconds(),
	).Scan(&id, &tipo, &carga, &tentativas, &versao, &esperou)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	inicio := time.Now()
	var falha error
	t.rodando.Add(1)
	if m, ok := t.Manipuladores[tipo]; ok {
		falha = m(ctx, carga)
	} else {
		falha = fmt.Errorf("%w: %s", ErrSemManipulador, tipo)
	}
	t.rodando.Add(-1)
	t.anotar(tipo, time.Duration(esperou*float64(time.Second)), time.Since(inicio), falha != nil)

	if falha == nil {
		tag, err := t.Pool.Exec(ctx, `
			UPDATE tarefa SET estado = 'CONCLUIDA', concluida_em = now(), ultimo_erro = NULL
			WHERE id = $1 AND versao = $2`, id, versao)
		if err == nil && tag.RowsAffected() == 0 {
			// Reenfileirada enquanto rodava: roda de novo com o estado mais novo — já, ou depois da
			// janela de agrupamento da tarefa (senão a mudança do meio furava a janela e, sob carga,
			// a tarefa rodava em laço).
			_, err = t.Pool.Exec(ctx, `
				UPDATE tarefa SET executar_em = now() + agrupar_s * interval '1 second', tentativas = 0
				WHERE id = $1`, id)
		}
		return true, err
	}

	estado := "PENDENTE"
	if tentativas >= t.max() {
		estado = "FALHOU"
		log.Printf("tarefa %d (%s) abandonada depois de %d tentativas: %v", id, tipo, tentativas, falha)
	}
	_, err = t.Pool.Exec(ctx, `
		UPDATE tarefa SET estado = $2, ultimo_erro = $3, executar_em = now() + $4::float8 * interval '1 second'
		WHERE id = $1`, id, estado, falha.Error(), espera(tentativas).Seconds())
	return true, err
}

// Rodar mantém n trabalhadores processando até o contexto acabar.
func (t *Trabalhador) Rodar(ctx context.Context, n int) {
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			t.ativos.Add(1)
			defer t.ativos.Add(-1)
			for ctx.Err() == nil {
				fez, err := t.ProcessarUma(ctx)
				if err != nil && ctx.Err() == nil {
					log.Printf("fila: %v", err)
				}
				if !fez || err != nil {
					select {
					case <-ctx.Done():
					case <-time.After(t.pausa()):
					}
				}
			}
		})
	}
	wg.Wait()
}

// Esvaziar processa até não sobrar tarefa vencida (para testes e para o seed).
func (t *Trabalhador) Esvaziar(ctx context.Context) error {
	for {
		fez, err := t.ProcessarUma(ctx)
		if err != nil || !fez {
			return err
		}
	}
}

// espera cresce a cada tentativa: 1 s, 2 s, 4 s, 8 s… até 5 min.
func espera(tentativas int) time.Duration {
	d := time.Second << min(tentativas-1, 9)
	return min(d, 5*time.Minute)
}

func (t *Trabalhador) max() int {
	if t.MaxTentativas > 0 {
		return t.MaxTentativas
	}
	return 5
}

func (t *Trabalhador) prazo() time.Duration {
	if t.Prazo > 0 {
		return t.Prazo
	}
	return time.Minute
}

func (t *Trabalhador) pausa() time.Duration {
	if t.Espera > 0 {
		return t.Espera
	}
	return 500 * time.Millisecond
}
