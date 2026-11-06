// Package exportacao gera o pacote do certame para o órgão e para a auditoria externa (02-produto §7
// H, "exportação para órgão e auditoria") e o confere sem o banco.
//
// O pacote é uma pasta: um CSV por assunto (trilha, inscrições, pagamentos, devoluções, dias
// fechados, avisos, ensalamento), as âncoras do cofre e um manifesto com o SHA-256 de cada arquivo e
// um selo do conjunto. O auditor confere com a pasta e nada mais:
//   - cada arquivo bate com o manifesto, e o manifesto com o selo;
//   - a corrente da trilha, refeita evento por evento (a mesma conta do banco);
//   - o selo de cada inscrição é o último evento da trilha dela;
//   - cada ponto de cada âncora do cofre está na trilha exportada, com o mesmo resumo;
//   - cada dia fechado, refeito dos pagamentos e das devoluções exportados.
//
// Dado pessoal fica fora: CPF mascarado, sem nome e sem e-mail. O número da inscrição liga tudo.
package exportacao

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/ancora"
	"lisura/internal/arquivo"
)

// Arquivo é uma entrada do manifesto.
type Arquivo struct {
	Nome   string `json:"nome"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type Manifesto struct {
	Certame  int64     `json:"certame"`
	Nome     string    `json:"nome"`
	Banca    string    `json:"banca"`
	GeradoEm time.Time `json:"gerado_em"`
	Arquivos []Arquivo `json:"arquivos"`
	// Selo é o SHA-256 das linhas "nome|sha256" dos arquivos, em ordem de nome: muda se qualquer
	// arquivo mudar, sumir ou aparecer. Quem recebe o pacote guarda só o selo.
	Selo string `json:"selo"`
}

// As consultas de cada CSV. O horário da trilha sai em UTC com microssegundos, exatamente como entra
// na conta do resumo: o auditor refaz a corrente com o que está no arquivo.
var consultas = []struct{ nome, sql string }{
	{"edital.csv", `SELECT versao, publicado_em, resumo FROM edital_versao WHERE certame_id = $1 ORDER BY versao`},
	{"cronograma.csv", `SELECT fase, abre_em, fecha_em FROM certame_fase WHERE certame_id = $1 ORDER BY abre_em`},
	{"inscricoes.csv", `
		SELECT i.id AS inscricao, g.nome AS cargo, coalesce(cp.nome, '') AS cidade, i.estado,
		       '***.' || substr(c.cpf, 4, 3) || '.' || substr(c.cpf, 7, 3) || '-**' AS cpf_mascarado,
		       array_to_string(i.necessidades, ',') AS atendimento,
		       coalesce((SELECT string_agg(ic.cota_codigo, ',' ORDER BY ic.cota_codigo) FROM inscricao_cota ic WHERE ic.inscricao_id = i.id), '') AS cotas,
		       coalesce((SELECT encode(hash, 'hex') FROM inscricao_evento e WHERE e.inscricao_id = i.id ORDER BY e.id DESC LIMIT 1), '') AS selo
		FROM inscricao i JOIN cargo g ON g.id = i.cargo_id JOIN candidato c ON c.id = i.candidato_id
		LEFT JOIN cidade_prova cp ON cp.id = i.cidade_prova_id
		WHERE i.certame_id = $1 ORDER BY i.id`},
	{"trilha.csv", `
		SELECT e.id AS evento_id, e.inscricao_id AS inscricao, e.evento, e.de_estado, e.para_estado, e.ator_tipo, e.ator_id,
		       coalesce(e.motivo_codigo, '') AS motivo, e.edital_versao,
		       to_char(e.ocorrido_em AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US') AS ocorrido_em_utc,
		       encode(e.hash, 'hex') AS resumo
		FROM inscricao_evento e JOIN inscricao i ON i.id = e.inscricao_id
		WHERE i.certame_id = $1 ORDER BY e.inscricao_id, e.id`},
	{"pagamentos.csv", `
		SELECT p.psp_pagamento_id, b.id AS cobranca, b.inscricao_id AS inscricao, b.metodo, p.valor_centavos, p.resultado,
		       p.pago_em, p.recebido_em, (p.recebido_em AT TIME ZONE 'America/Sao_Paulo')::date AS dia, p.origem
		FROM pagamento_recebido p JOIN cobranca b ON b.id = p.cobranca_id JOIN inscricao i ON i.id = b.inscricao_id
		WHERE i.certame_id = $1 ORDER BY p.psp_pagamento_id`},
	{"devolucoes.csv", `
		SELECT d.id, d.psp_pagamento_id, d.inscricao_id AS inscricao, d.valor_centavos, d.motivo, d.estado,
		       coalesce(d.psp_devolucao_id, '') AS psp_devolucao_id, d.concluida_em,
		       (d.concluida_em AT TIME ZONE 'America/Sao_Paulo')::date AS dia
		FROM devolucao d JOIN inscricao i ON i.id = d.inscricao_id WHERE i.certame_id = $1 ORDER BY d.id`},
	{"fechamentos.csv", `
		SELECT dia, pagamentos, entrou_centavos, devolucoes, saiu_centavos, resumo, fechado_em, fechado_por
		FROM arrecadacao_fechamento WHERE certame_id = $1 ORDER BY dia`},
	{"avisos.csv", `
		SELECT n.evento_id, n.inscricao_id AS inscricao, e.evento, n.assunto, n.resumo, n.provedor_id IS NOT NULL AS entregue, n.enviada_em
		FROM notificacao n JOIN inscricao_evento e ON e.id = n.evento_id JOIN inscricao i ON i.id = n.inscricao_id
		WHERE i.certame_id = $1 ORDER BY n.evento_id`},
	// O que aconteceu com o aviso depois de sair (devolução, reclamação, entrega). Sem o endereço e sem
	// o diagnóstico do servidor de e-mail, que costuma repetir o endereço: só o tipo e a classe.
	{"avisos_ocorrencias.csv", `
		SELECT n.evento_id, n.inscricao_id AS inscricao, o.tipo, split_part(coalesce(o.detalhe, ''), ':', 1) AS classe, o.ocorrida_em
		FROM notificacao_ocorrencia o JOIN notificacao n ON n.provedor_id = o.provedor_id
		JOIN inscricao i ON i.id = n.inscricao_id
		WHERE i.certame_id = $1 ORDER BY n.evento_id, o.id`},
	{"alocacao.csv", `
		SELECT a.inscricao_id AS inscricao, cp.nome AS cidade, l.nome AS local, s.nome AS sala, s.tipo, a.versao
		FROM alocacao a JOIN sala s ON s.id = a.sala_id JOIN local_prova l ON l.id = s.local_id
		JOIN cidade_prova cp ON cp.id = l.cidade_prova_id WHERE a.certame_id = $1 ORDER BY a.inscricao_id`},
	{"ensalamento_versoes.csv", `
		SELECT versao, alocadas, mantidas, novas, removidas, resumo, motivo, criado_por, criado_em
		FROM ensalamento_versao WHERE certame_id = $1 ORDER BY versao`},
}

// Exportar grava o pacote do certame na pasta (que tem de estar vazia ou não existir). As consultas
// rodam numa transação só, REPEATABLE READ: os arquivos são da mesma fotografia do banco, e a trilha
// exportada é a mesma que deu os selos das inscrições.
func Exportar(ctx context.Context, pool *pgxpool.Pool, cofre arquivo.Cofre, certameID int64, pasta string) (Manifesto, error) {
	m := Manifesto{Certame: certameID, GeradoEm: time.Now().UTC()}
	if ents, err := os.ReadDir(pasta); err == nil && len(ents) > 0 {
		return m, fmt.Errorf("exportação: a pasta %s não está vazia", pasta)
	}
	if err := os.MkdirAll(filepath.Join(pasta, "ancoras"), 0o755); err != nil {
		return m, err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return m, err
	}
	defer conn.Release()
	err = pgx.BeginTxFunc(ctx, conn, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT c.nome, b.nome FROM certame c JOIN banca b ON b.id = c.banca_id WHERE c.id = $1`,
			certameID).Scan(&m.Nome, &m.Banca); err != nil {
			return fmt.Errorf("certame %d: %w", certameID, err)
		}
		for _, c := range consultas {
			a, err := copiar(ctx, tx, filepath.Join(pasta, c.nome), c.sql, certameID)
			if err != nil {
				return fmt.Errorf("%s: %w", c.nome, err)
			}
			a.Nome = c.nome
			m.Arquivos = append(m.Arquivos, a)
		}
		return nil
	})
	if err != nil {
		return m, err
	}

	// As âncoras do cofre, todas: a incremental só leva o que mudou, e a conferência precisa de cada
	// ponto já fixado fora do banco.
	if cofre != nil {
		prefixo := strings.TrimSuffix(ancora.Chave(certameID, 0), "000000.txt")
		chaves, err := cofre.Listar(ctx, prefixo)
		if err != nil {
			return m, err
		}
		for _, k := range chaves {
			b, err := cofre.Ler(ctx, k)
			if err != nil {
				return m, err
			}
			nome := "ancoras/" + filepath.Base(k)
			if err := os.WriteFile(filepath.Join(pasta, nome), b, 0o644); err != nil {
				return m, err
			}
			h := sha256.Sum256(b)
			m.Arquivos = append(m.Arquivos, Arquivo{Nome: nome, Bytes: int64(len(b)), SHA256: hex.EncodeToString(h[:])})
		}
	}
	slices.SortFunc(m.Arquivos, func(a, b Arquivo) int { return strings.Compare(a.Nome, b.Nome) })
	m.Selo = selo(m.Arquivos)
	b, _ := json.MarshalIndent(m, "", "  ")
	return m, os.WriteFile(filepath.Join(pasta, "manifesto.json"), b, 0o644)
}

// copiar grava o resultado da consulta como CSV (separador ;, com cabeçalho) direto do COPY do
// Postgres, calculando o SHA-256 no caminho: um certame de 200 mil inscrições não passa pela memória.
func copiar(ctx context.Context, tx pgx.Tx, caminho, sql string, certameID int64) (Arquivo, error) {
	f, err := os.Create(caminho)
	if err != nil {
		return Arquivo{}, err
	}
	defer f.Close()
	h := sha256.New()
	w := &contador{w: io.MultiWriter(f, h)}
	// O COPY não aceita parâmetro: o id vai como número, formatado aqui, nunca texto do usuário.
	consulta := strings.ReplaceAll(sql, "$1", fmt.Sprint(certameID))
	if _, err := tx.Conn().PgConn().CopyTo(ctx, w, "COPY ("+consulta+") TO STDOUT WITH (FORMAT csv, DELIMITER ';', HEADER)"); err != nil {
		return Arquivo{}, err
	}
	if err := f.Sync(); err != nil {
		return Arquivo{}, err
	}
	return Arquivo{Bytes: w.n, SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}

type contador struct {
	w io.Writer
	n int64
}

func (c *contador) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

func selo(as []Arquivo) string {
	h := sha256.New()
	for _, a := range as {
		fmt.Fprintf(h, "%s|%s\n", a.Nome, a.SHA256)
	}
	return hex.EncodeToString(h.Sum(nil))
}
