// Package conta é a conta única do candidato: uma por CPF, viva entre bancas e certames, com a
// carteira de documentos. Nós somos controladores destes dados (02-produto §10, decisão 2).
package conta

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCPFJaCadastrado = errors.New("CPF já cadastrado")
	ErrNaoEncontrada   = errors.New("conta não encontrada")
	ErrDadoInvalido    = errors.New("dado inválido")
	ErrArquivoGrande   = errors.New("arquivo maior que o permitido")
	ErrFormatoArquivo  = errors.New("formato de arquivo não aceito")
)

// MaxArquivo é o tamanho máximo de um documento enviado.
const MaxArquivo = 5 << 20

var formatosAceitos = []string{"application/pdf", "image/jpeg", "image/png"}

// Armazenamento guarda o conteúdo dos documentos (disco na POC, S3 na arquitetura alvo).
type Armazenamento interface {
	Put(ctx context.Context, chave string, conteudo []byte) error
}

func Criar(ctx context.Context, pool *pgxpool.Pool, cpf, nome, email string) (int64, error) {
	cpf = SoDigitos(cpf)
	if !CPFValido(cpf) {
		return 0, fmt.Errorf("%w: CPF", ErrDadoInvalido)
	}
	if strings.TrimSpace(nome) == "" {
		return 0, fmt.Errorf("%w: nome", ErrDadoInvalido)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return 0, fmt.Errorf("%w: e-mail", ErrDadoInvalido)
	}
	var id int64
	err := pool.QueryRow(ctx,
		`INSERT INTO candidato (cpf, nome, email) VALUES ($1, $2, $3) RETURNING id`,
		cpf, strings.TrimSpace(nome), email).Scan(&id)
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return 0, ErrCPFJaCadastrado
	}
	return id, err
}

func PorCPF(ctx context.Context, pool *pgxpool.Pool, cpf string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM candidato WHERE cpf = $1`, SoDigitos(cpf)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNaoEncontrada
	}
	return id, err
}

type Dados struct {
	Nome           string `json:"nome"`
	Email          string `json:"email"`
	DataNascimento string `json:"data_nascimento"` // AAAA-MM-DD
	Municipio      string `json:"municipio"`
	UF             string `json:"uf"`
}

var reUF = regexp.MustCompile(`^[A-Z]{2}$`)

// AtualizarDados é o passo 1 do formulário: dados pessoais e endereço, guardados na conta.
func AtualizarDados(ctx context.Context, pool *pgxpool.Pool, id int64, d Dados) error {
	nasc, err := time.Parse(time.DateOnly, d.DataNascimento)
	if err != nil || nasc.After(time.Now()) || nasc.Year() < 1900 {
		return fmt.Errorf("%w: data de nascimento", ErrDadoInvalido)
	}
	if _, err := mail.ParseAddress(d.Email); err != nil {
		return fmt.Errorf("%w: e-mail", ErrDadoInvalido)
	}
	d.UF = strings.ToUpper(d.UF)
	if strings.TrimSpace(d.Nome) == "" || strings.TrimSpace(d.Municipio) == "" || !reUF.MatchString(d.UF) {
		return fmt.Errorf("%w: nome, município ou UF", ErrDadoInvalido)
	}
	tag, err := pool.Exec(ctx, `
		UPDATE candidato SET nome = $2, email = $3, data_nascimento = $4, municipio = $5, uf = $6
		WHERE id = $1`, id, strings.TrimSpace(d.Nome), d.Email, nasc, strings.TrimSpace(d.Municipio), d.UF)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNaoEncontrada
	}
	return err
}

// Buscar devolve os dados da conta; a data vem como AAAA-MM-DD, ou vazia se nunca preenchida.
func Buscar(ctx context.Context, pool *pgxpool.Pool, id int64) (cpf string, d Dados, err error) {
	var nasc *time.Time
	var municipio, uf *string
	err = pool.QueryRow(ctx,
		`SELECT cpf, nome, email, data_nascimento, municipio, uf FROM candidato WHERE id = $1`, id,
	).Scan(&cpf, &d.Nome, &d.Email, &nasc, &municipio, &uf)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", d, ErrNaoEncontrada
	}
	if nasc != nil {
		d.DataNascimento = nasc.Format(time.DateOnly)
	}
	if municipio != nil {
		d.Municipio = *municipio
	}
	if uf != nil {
		d.UF = *uf
	}
	return cpf, d, err
}

type Documento struct {
	ID          int64     `json:"id"`
	Tipo        string    `json:"tipo"`
	NomeArquivo string    `json:"nome_arquivo"`
	Tamanho     int       `json:"tamanho"`
	EnviadoEm   time.Time `json:"enviado_em"`
}

func Documentos(ctx context.Context, pool *pgxpool.Pool, candidatoID int64) ([]Documento, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, tipo, nome_arquivo, tamanho, enviado_em FROM documento
		WHERE candidato_id = $1 ORDER BY enviado_em DESC`, candidatoID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Documento])
}

// SalvarDocumento valida e guarda um documento na carteira. O formato é conferido pelo conteúdo,
// não pela extensão nem pelo cabeçalho que o navegador mandou.
func SalvarDocumento(ctx context.Context, pool *pgxpool.Pool, store Armazenamento, candidatoID int64, tipo, nomeArquivo string, conteudo []byte) (int64, error) {
	if tipo != "laudo" && tipo != "comprovante_isencao" {
		return 0, fmt.Errorf("%w: tipo %q", ErrDadoInvalido, tipo)
	}
	if len(conteudo) == 0 {
		return 0, fmt.Errorf("%w: arquivo vazio", ErrDadoInvalido)
	}
	if len(conteudo) > MaxArquivo {
		return 0, ErrArquivoGrande
	}
	mime := http.DetectContentType(conteudo)
	if !slices.Contains(formatosAceitos, mime) || (mime == "application/pdf" && !bytes.HasPrefix(conteudo, []byte("%PDF-"))) {
		return 0, fmt.Errorf("%w: %s", ErrFormatoArquivo, mime)
	}

	soma := sha256.Sum256(conteudo)
	chave := fmt.Sprintf("documentos/%d/%x", candidatoID, soma)
	if err := store.Put(ctx, chave, conteudo); err != nil {
		return 0, err
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO documento (candidato_id, tipo, nome_arquivo, mime, tamanho, sha256, chave)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		candidatoID, tipo, nomeArquivo, mime, len(conteudo), soma[:], chave).Scan(&id)
	return id, err
}
