// Package ancora fixa a trilha fora do banco. A trilha encadeada (migração 00010) denuncia quem
// altera ou apaga um evento, mas não quem reescreve a corrente inteira: esse só o selo impresso no
// cartão denuncia, e só se o candidato guardou o cartão. A âncora é a mesma testemunha para todos:
// de tempos em tempos, o último resumo de cada inscrição do certame vai para um cofre que não aceita
// alteração (S3 com bloqueio de objeto), e um recibo com o resumo da âncora vai para o site público,
// onde qualquer um pode guardá-lo.
//
// Reescrever a trilha depois de ancorada exige reescrever o cofre — o que o bloqueio impede — e os
// recibos que já saíram.
package ancora

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/arquivo"
)

// Linha é o ponto da corrente de uma inscrição no momento da âncora.
type Linha struct {
	Inscricao int64
	Evento    int64  // id do último evento
	Resumo    []byte // o hash desse evento: cobre a corrente inteira até ele
}

// Uma âncora completa lista todas as inscrições do certame; uma incremental, só as que mudaram desde
// a anterior. Somadas desde a última completa, dão o mesmo estado.
const (
	Completa    = "completa"
	Incremental = "incremental"
)

// CompletaCada é de quanto em quanto tempo sai uma âncora completa; entre elas, incrementais. Medido
// em 05/11: a completa de 200 mil inscrições tem ~16 MB; uma por hora em 60 dias de inscrição seriam
// ~23 GB por certame no cofre, quase tudo repetido.
var CompletaCada = 24 * time.Hour

type Ancora struct {
	Certame   int64
	Sequencia int
	Tipo      string // Completa ou Incremental
	Anterior  string // resumo da âncora anterior, em hexadecimal; "-" na primeira
	GeradaEm  time.Time
	Eventos   int64 // quantos eventos o certame tinha
	Total     int   // quantas inscrições a âncora cobre, somada às anteriores desde a última completa
	Linhas    []Linha
}

const (
	cabecalhoV1 = "lisura-ancora 1" // sem tipo nem total: toda âncora era completa
	cabecalho   = "lisura-ancora 2"
)

// Texto é o arquivo que vai para o cofre: texto simples, para ser conferido sem esta aplicação.
func (a Ancora) Texto() []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s\ncertame %d\nsequencia %d\ntipo %s\nanterior %s\ngerada_em %s\neventos %d\ntotal %d\ninscricoes %d\n",
		cabecalho, a.Certame, a.Sequencia, a.Tipo, a.Anterior, a.GeradaEm.UTC().Format(time.RFC3339Nano), a.Eventos, a.Total, len(a.Linhas))
	for _, l := range a.Linhas {
		fmt.Fprintf(&b, "%d %d %s\n", l.Inscricao, l.Evento, hex.EncodeToString(l.Resumo))
	}
	return b.Bytes()
}

// Resumo é o SHA-256 do arquivo inteiro: é o que o recibo público leva e o que a próxima âncora cita.
func Resumo(texto []byte) string {
	h := sha256.Sum256(texto)
	return hex.EncodeToString(h[:])
}

var ErrFormato = errors.New("âncora fora do formato")

func Ler(texto []byte) (Ancora, error) {
	var a Ancora
	sc := bufio.NewScanner(bytes.NewReader(texto))
	sc.Buffer(make([]byte, 0, 4096), 1<<20)
	campo := func(nome string) (string, error) {
		if !sc.Scan() {
			return "", fmt.Errorf("%w: falta %s", ErrFormato, nome)
		}
		v, ok := strings.CutPrefix(sc.Text(), nome+" ")
		if !ok {
			return "", fmt.Errorf("%w: esperado %s, veio %q", ErrFormato, nome, sc.Text())
		}
		return v, nil
	}
	if !sc.Scan() || (sc.Text() != cabecalho && sc.Text() != cabecalhoV1) {
		return a, fmt.Errorf("%w: cabeçalho", ErrFormato)
	}
	v1 := sc.Text() == cabecalhoV1
	type leitor struct {
		nome string
		ler  func(string) error
	}
	var n int
	certame := leitor{"certame", func(v string) (err error) { a.Certame, err = strconv.ParseInt(v, 10, 64); return }}
	sequencia := leitor{"sequencia", func(v string) (err error) { a.Sequencia, err = strconv.Atoi(v); return }}
	tipo := leitor{"tipo", func(v string) error {
		if v != Completa && v != Incremental {
			return fmt.Errorf("tipo %q", v)
		}
		a.Tipo = v
		return nil
	}}
	anterior := leitor{"anterior", func(v string) error { a.Anterior = v; return nil }}
	geradaEm := leitor{"gerada_em", func(v string) (err error) { a.GeradaEm, err = time.Parse(time.RFC3339Nano, v); return }}
	eventos := leitor{"eventos", func(v string) (err error) { a.Eventos, err = strconv.ParseInt(v, 10, 64); return }}
	total := leitor{"total", func(v string) (err error) { a.Total, err = strconv.Atoi(v); return }}
	inscricoes := leitor{"inscricoes", func(v string) (err error) { n, err = strconv.Atoi(v); return }}
	campos := []leitor{certame, sequencia, tipo, anterior, geradaEm, eventos, total, inscricoes}
	if v1 {
		campos = []leitor{certame, sequencia, anterior, geradaEm, eventos, inscricoes}
	}
	for _, c := range campos {
		v, err := campo(c.nome)
		if err != nil {
			return a, err
		}
		if err := c.ler(v); err != nil {
			return a, fmt.Errorf("%w: %s: %v", ErrFormato, c.nome, err)
		}
	}
	a.Linhas = make([]Linha, 0, n)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) != 3 {
			return a, fmt.Errorf("%w: linha %q", ErrFormato, sc.Text())
		}
		var l Linha
		var err1, err2, err3 error
		l.Inscricao, err1 = strconv.ParseInt(f[0], 10, 64)
		l.Evento, err2 = strconv.ParseInt(f[1], 10, 64)
		l.Resumo, err3 = hex.DecodeString(f[2])
		if err := errors.Join(err1, err2, err3); err != nil || len(l.Resumo) != sha256.Size {
			return a, fmt.Errorf("%w: linha %q", ErrFormato, sc.Text())
		}
		a.Linhas = append(a.Linhas, l)
	}
	if len(a.Linhas) != n {
		return a, fmt.Errorf("%w: diz %d inscrições e traz %d", ErrFormato, n, len(a.Linhas))
	}
	if v1 {
		a.Tipo, a.Total = Completa, n
	}
	if a.Tipo == Completa && a.Total != n {
		return a, fmt.Errorf("%w: completa com total %d e %d linhas", ErrFormato, a.Total, n)
	}
	return a, sc.Err()
}

func prefixo(certame int64) string { return fmt.Sprintf("certame-%08d/", certame) }

// Chave no cofre. A sequência com zeros à esquerda faz a ordem do nome ser a ordem das âncoras.
func Chave(certame int64, seq int) string { return fmt.Sprintf("%s%06d.txt", prefixo(certame), seq) }

// CaminhoRecibo é onde o recibo sai no site público (/publico/...).
func CaminhoRecibo(certame int64, seq int) string {
	return fmt.Sprintf("certames/%d/ancoras/%06d.txt", certame, seq)
}

// Recibo é o que vai a público: o cabeçalho e o resumo, sem as linhas (que listam as inscrições).
func Recibo(a Ancora, resumo string) []byte {
	return fmt.Appendf(nil, "lisura-recibo-ancora 2\ncertame %d\nsequencia %d\ntipo %s\nanterior %s\ngerada_em %s\neventos %d\ntotal %d\ninscricoes %d\nresumo %s\n",
		a.Certame, a.Sequencia, a.Tipo, a.Anterior, a.GeradaEm.UTC().Format(time.RFC3339Nano), a.Eventos, a.Total, len(a.Linhas), resumo)
}

// Trocáveis no teste, que desliga a trava e alarga a janela entre a fotografia e a gravação para
// provar que a ordem sozinha basta.
var (
	usarTrava          = true
	depoisDaFotografia = func() {}
)

// Gravador publica o recibo (o publicacao.Publicador serve: disco ou S3).
type Gravador interface {
	Gravar(ctx context.Context, caminho string, conteudo []byte) error
	Existe(ctx context.Context, caminho string) bool
}

// Ancorar grava uma âncora nova do certame se a trilha mudou desde a última.
//
// A lista é lida numa transação REPEATABLE READ: as linhas e a contagem de eventos são da mesma
// fotografia do banco. A ordem é o que garante que uma âncora nunca é mais velha que a anterior:
// a última âncora é lida no cofre ANTES da fotografia. Quem grava a sequência k+1 leu a k depois de
// ela existir — logo, fotografou depois de quem gravou a k. Duas instâncias disputando a mesma
// sequência: o cofre aceita uma e recusa a outra, que volta com gerou = false. A trava consultiva só
// poupa a perdedora de montar a lista à toa.
//
// Sai completa na primeira vez e quando a última completa tem mais que CompletaCada; senão, só com as
// inscrições cujo ponto mudou desde o estado que o cofre já guarda.
func Ancorar(ctx context.Context, pool *pgxpool.Pool, cofre arquivo.Cofre, publico Gravador, certame int64) (a Ancora, gerou bool, err error) {
	h, err := lerHistorico(ctx, cofre, certame)
	if err != nil {
		return a, false, err
	}
	ultima, resumoUltima := h.ultima, h.resumoUltima
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return a, false, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	const trava = 0x416e63 // "Anc"
	var livre bool         // a primeira consulta da transação: é aqui que a fotografia é tirada
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1, $2::int)`, trava, certame).Scan(&livre); err != nil || (!livre && usarTrava) {
		return a, false, err
	}
	depoisDaFotografia()
	a = Ancora{Certame: certame, Sequencia: 1, Anterior: "-"}
	if ultima != nil {
		a.Sequencia, a.Anterior = ultima.Sequencia+1, resumoUltima
	}
	if err := tx.QueryRow(ctx, `
		SELECT count(*), clock_timestamp() FROM inscricao_evento e JOIN inscricao i ON i.id = e.inscricao_id
		WHERE i.certame_id = $1`, certame).Scan(&a.Eventos, &a.GeradaEm); err != nil {
		return a, false, err
	}
	// Recibo que não saiu (a publicação falhou depois do cofre, ou a âncora veio da linha de comando,
	// que não publica) sai agora.
	if err := recibosQueFaltam(ctx, publico, h); err != nil {
		return a, false, err
	}
	if ultima != nil && ultima.Eventos == a.Eventos {
		return *ultima, false, nil // nada novo para ancorar
	}
	if a.Eventos == 0 {
		return a, false, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT ON (e.inscricao_id) e.inscricao_id, e.id, e.hash
		  FROM inscricao_evento e JOIN inscricao i ON i.id = e.inscricao_id
		 WHERE i.certame_id = $1
		 ORDER BY e.inscricao_id, e.id DESC`, certame)
	if err != nil {
		return a, false, err
	}
	a.Tipo = Completa
	if ultima != nil && a.GeradaEm.Sub(h.completaEm) < CompletaCada {
		a.Tipo = Incremental
	}
	var l Linha
	_, err = pgx.ForEachRow(rows, []any{&l.Inscricao, &l.Evento, &l.Resumo}, func() error {
		a.Total++
		if a.Tipo == Incremental {
			if antes, ok := h.estado[l.Inscricao]; ok && antes.Evento == l.Evento && bytes.Equal(antes.Resumo, l.Resumo) {
				return nil // o cofre já tem este ponto
			}
		}
		a.Linhas = append(a.Linhas, Linha{l.Inscricao, l.Evento, append([]byte(nil), l.Resumo...)})
		return nil
	})
	if err != nil {
		return a, false, err
	}
	texto := a.Texto()
	if err := cofre.Criar(ctx, Chave(certame, a.Sequencia), texto); errors.Is(err, arquivo.ErrJaExiste) {
		return a, false, nil // outra instância gravou esta sequência primeiro
	} else if err != nil {
		return a, false, err
	}
	return a, true, publicarRecibo(ctx, publico, a, Resumo(texto))
}

func publicarRecibo(ctx context.Context, publico Gravador, a Ancora, resumo string) error {
	if publico == nil {
		return nil
	}
	return publico.Gravar(ctx, CaminhoRecibo(a.Certame, a.Sequencia), Recibo(a, resumo))
}

// recibosQueFaltam publica o recibo de cada âncora lida (desde a última completa) que ainda não
// está no site.
func recibosQueFaltam(ctx context.Context, publico Gravador, h historico) error {
	if publico == nil {
		return nil
	}
	for i, a := range h.desde {
		if !publico.Existe(ctx, CaminhoRecibo(a.Certame, a.Sequencia)) {
			if err := publicarRecibo(ctx, publico, a, h.resumos[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// historico é o que o cofre já guarda do certame, do ponto de vista de quem vai ancorar.
type historico struct {
	ultima       *Ancora // nil se não há nenhuma
	resumoUltima string
	completaEm   time.Time       // quando saiu a última completa
	estado       map[int64]Linha // a última completa somada às incrementais seguintes
	desde        []Ancora        // da mais nova até a última completa
	resumos      []string        // o resumo de cada uma de desde
}

// lerHistorico lê do fim para o começo até achar a última completa: as anteriores a ela não mudam o
// estado.
func lerHistorico(ctx context.Context, cofre arquivo.Cofre, certame int64) (h historico, err error) {
	chaves, err := cofre.Listar(ctx, prefixo(certame))
	if err != nil || len(chaves) == 0 {
		return h, err
	}
	for i := len(chaves) - 1; i >= 0; i-- {
		texto, err := cofre.Ler(ctx, chaves[i])
		if err != nil {
			return h, err
		}
		a, err := Ler(texto)
		if err != nil {
			return h, fmt.Errorf("%s: %w", chaves[i], err)
		}
		if h.ultima == nil {
			h.ultima, h.resumoUltima = &a, Resumo(texto)
		}
		h.desde, h.resumos = append(h.desde, a), append(h.resumos, Resumo(texto))
		if a.Tipo == Completa {
			h.completaEm = a.GeradaEm
			break
		}
	}
	h.estado = map[int64]Linha{}
	for i := len(h.desde) - 1; i >= 0; i-- {
		for _, l := range h.desde[i].Linhas {
			h.estado[l.Inscricao] = l
		}
	}
	return h, nil
}

// Ultima devolve a âncora mais recente do certame no cofre e o resumo dela (nil se não há nenhuma).
func Ultima(ctx context.Context, cofre arquivo.Cofre, certame int64) (*Ancora, string, error) {
	h, err := lerHistorico(ctx, cofre, certame)
	return h.ultima, h.resumoUltima, err
}
