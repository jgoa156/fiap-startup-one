package arquivo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// Cofre guarda o que não pode mudar depois de gravado (as âncoras da trilha): grava uma vez e não
// sobrescreve. Quem tem acesso ao banco não tem acesso ao cofre — é isso que faz dele testemunha.
type Cofre interface {
	Criar(ctx context.Context, chave string, conteudo []byte) error // ErrJaExiste se a chave já existe
	Ler(ctx context.Context, chave string) ([]byte, error)
	Listar(ctx context.Context, prefixo string) ([]string, error) // em ordem crescente
}

var ErrJaExiste = errors.New("já existe no cofre: o cofre não sobrescreve")

// CofreDoAmbiente escolhe o cofre como a API e a auditoria o configuram: LISURA_COFRE_BUCKET liga o
// S3 (credenciais pelo papel da tarefa); sem ele, a pasta LISURA_COFRE (padrão ./dados/cofre).
// onde descreve o cofre escolhido, para o log.
func CofreDoAmbiente(ctx context.Context) (c Cofre, onde string, err error) {
	if bucket := os.Getenv("LISURA_COFRE_BUCKET"); bucket != "" {
		cfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, "", err
		}
		cliente := s3.NewFromConfig(cfg)
		if _, err := cliente.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
			return nil, "", fmt.Errorf("cofre %q: %w", bucket, err)
		}
		return CofreS3{Cliente: cliente, Bucket: bucket, Prefixo: "ancoras/"}, "s3://" + bucket + "/ancoras/", nil
	}
	dir := os.Getenv("LISURA_COFRE")
	if dir == "" {
		dir = filepath.Join("dados", "cofre")
	}
	return CofreDisco{Dir: dir}, dir + " (disco: só imita o bloqueio)", nil
}

// CofreDisco é o cofre da máquina local. A gravação é por link (atômico e recusado se o destino
// existe) e o arquivo fica somente leitura. Só imita a regra: quem tem o disco apaga. O cofre de
// verdade é o CofreS3, com bloqueio de objeto.
type CofreDisco struct{ Dir string }

func (c CofreDisco) Criar(_ context.Context, chave string, conteudo []byte) error {
	destino := filepath.Join(c.Dir, filepath.FromSlash(chave))
	if err := os.MkdirAll(filepath.Dir(destino), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(destino), ".parcial-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(conteudo); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// Ao contrário do rename, o link falha se o destino já existe: duas instâncias gravando a mesma
	// âncora não se sobrescrevem.
	if err := os.Link(tmp.Name(), destino); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ErrJaExiste
		}
		return err
	}
	return os.Chmod(destino, 0o444)
}

func (c CofreDisco) Ler(_ context.Context, chave string) ([]byte, error) {
	return os.ReadFile(filepath.Join(c.Dir, filepath.FromSlash(chave)))
}

func (c CofreDisco) Listar(_ context.Context, prefixo string) ([]string, error) {
	dir := filepath.Join(c.Dir, filepath.FromSlash(prefixo))
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.HasPrefix(d.Name(), ".parcial-") {
			return nil
		}
		rel, err := filepath.Rel(c.Dir, p)
		out = append(out, filepath.ToSlash(rel))
		return err
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	sort.Strings(out)
	return out, err
}

// CofreS3 grava num bucket com bloqueio de objeto (Object Lock): a retenção padrão do bucket impede
// apagar ou sobrescrever cada versão até o fim do prazo, inclusive para o administrador da conta no
// modo COMPLIANCE. O If-None-Match: * faz o S3 recusar a gravação se a chave já existe — sem ele, um
// PUT na mesma chave criaria uma versão nova por cima (a antiga ficaria guardada, mas a leitura
// devolveria a nova).
type CofreS3 struct {
	Cliente *s3.Client
	Bucket  string
	Prefixo string
}

func (c CofreS3) Criar(ctx context.Context, chave string, conteudo []byte) error {
	_, err := c.Cliente.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.Bucket),
		Key:         aws.String(c.Prefixo + chave),
		Body:        bytes.NewReader(conteudo),
		ContentType: aws.String(tipo(chave)),
		IfNoneMatch: aws.String("*"),
	})
	var api smithy.APIError
	if errors.As(err, &api) && (api.ErrorCode() == "PreconditionFailed" || api.ErrorCode() == "ConditionalRequestConflict") {
		return ErrJaExiste
	}
	return err
}

func (c CofreS3) Ler(ctx context.Context, chave string) ([]byte, error) {
	obj, err := c.Cliente.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(c.Bucket), Key: aws.String(c.Prefixo + chave)})
	if err != nil {
		return nil, err
	}
	defer obj.Body.Close()
	return io.ReadAll(obj.Body)
}

func (c CofreS3) Listar(ctx context.Context, prefixo string) ([]string, error) {
	var out []string
	pag := s3.NewListObjectsV2Paginator(c.Cliente, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.Bucket), Prefix: aws.String(c.Prefixo + prefixo)})
	for pag.HasMorePages() {
		p, err := pag.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range p.Contents {
			out = append(out, strings.TrimPrefix(aws.ToString(o.Key), c.Prefixo))
		}
	}
	sort.Strings(out) // o S3 já devolve em ordem; o sort garante o contrato para quem implementar outro
	return out, nil
}
