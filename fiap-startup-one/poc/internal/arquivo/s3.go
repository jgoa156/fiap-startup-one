package arquivo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// Destino é onde a fila publica e onde os documentos ficam: disco na máquina local, S3 na nuvem.
type Destino interface {
	Put(ctx context.Context, chave string, conteudo []byte) error
	Existe(ctx context.Context, chave string) (bool, error)
}

// S3 grava objetos num bucket. Um PUT no S3 é atômico (o objeto aparece inteiro ou não aparece),
// que é a mesma garantia do rename do Disco.
type S3 struct {
	Cliente *s3.Client
	Bucket  string
	Prefixo string // ex.: "publico/" ou "documentos/"
	// CacheControl vai em cada objeto: o CloudFront e o navegador respeitam. Vazio para documentos.
	CacheControl string
}

func (d S3) Put(ctx context.Context, chave string, conteudo []byte) error {
	in := &s3.PutObjectInput{
		Bucket:      aws.String(d.Bucket),
		Key:         aws.String(d.Prefixo + chave),
		Body:        bytes.NewReader(conteudo),
		ContentType: aws.String(tipo(chave)),
	}
	if d.CacheControl != "" {
		in.CacheControl = aws.String(d.CacheControl)
	}
	_, err := d.Cliente.PutObject(ctx, in)
	return err
}

// Verificar confere que o bucket existe e está acessível. Chamado na subida do servidor: um HEAD de
// objeto não distingue "bucket errado" de "ainda não publicado" (os dois dão 404).
func (d S3) Verificar(ctx context.Context) error {
	if _, err := d.Cliente.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(d.Bucket)}); err != nil {
		return fmt.Errorf("bucket %q: %w", d.Bucket, err)
	}
	return nil
}

func (d S3) Existe(ctx context.Context, chave string) (bool, error) {
	_, err := d.Cliente.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(d.Bucket), Key: aws.String(d.Prefixo + chave)})
	if err == nil {
		return true, nil
	}
	var api smithy.APIError
	if errors.As(err, &api) && (api.ErrorCode() == "NotFound" || api.ErrorCode() == "NoSuchKey") {
		return false, nil
	}
	var resp interface{ HTTPStatusCode() int }
	if errors.As(err, &resp) && resp.HTTPStatusCode() == http.StatusNotFound {
		return false, nil
	}
	return false, err
}

func tipo(chave string) string {
	if t := mime.TypeByExtension(path.Ext(chave)); t != "" {
		return t
	}
	return "application/octet-stream"
}
