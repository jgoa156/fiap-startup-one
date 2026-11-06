package arquivo

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
)

// Um S3 falso dentro do teste: confere que o cliente fala o protocolo (PUT, HEAD, 404), não o S3 real.
// cabecalhos guarda os cabeçalhos de cada PUT, porque o falso não guarda todos (Cache-Control, por
// exemplo) e o que importa é o que o cliente manda.
func s3Falso(t *testing.T) (*s3.Client, map[string]http.Header) {
	t.Helper()
	cabecalhos := map[string]http.Header{}
	var mu sync.Mutex
	falso := gofakes3.New(s3mem.New()).Server()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			mu.Lock()
			cabecalhos[r.URL.Path] = r.Header.Clone()
			mu.Unlock()
		}
		falso.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	c := s3.New(s3.Options{
		Region:       "sa-east-1",
		BaseEndpoint: aws.String(srv.URL),
		UsePathStyle: true,
		Credentials:  credentials.NewStaticCredentialsProvider("teste", "teste", ""),
	})
	if _, err := c.CreateBucket(context.Background(), &s3.CreateBucketInput{Bucket: aws.String("lisura")}); err != nil {
		t.Fatal(err)
	}
	return c, cabecalhos
}

func TestS3GravaComTipoECache(t *testing.T) {
	c, cabecalhos := s3Falso(t)
	d := S3{Cliente: c, Bucket: "lisura", Prefixo: "publico/", CacheControl: "public, max-age=60"}
	ctx := context.Background()

	if ok, err := d.Existe(ctx, "cartoes/abc.html"); ok || err != nil {
		t.Fatalf("antes de gravar: existe=%v err=%v", ok, err)
	}
	if err := d.Put(ctx, "cartoes/abc.html", []byte("<h1>cartão</h1>")); err != nil {
		t.Fatal(err)
	}
	if ok, err := d.Existe(ctx, "cartoes/abc.html"); !ok || err != nil {
		t.Fatalf("depois de gravar: existe=%v err=%v", ok, err)
	}

	obj, err := c.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String("lisura"), Key: aws.String("publico/cartoes/abc.html")})
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Body.Close()
	corpo, _ := io.ReadAll(obj.Body)
	if string(corpo) != "<h1>cartão</h1>" {
		t.Fatalf("conteúdo: %q", corpo)
	}
	if got := aws.ToString(obj.ContentType); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q: o CloudFront entregaria o cartão como download", got)
	}
	if got := cabecalhos["/lisura/publico/cartoes/abc.html"].Get("Cache-Control"); got != "public, max-age=60" {
		t.Errorf("Cache-Control enviado = %q: o CloudFront usaria o cache padrão dele", got)
	}
}

// Um HEAD num bucket inexistente devolve o mesmo 404 de um objeto ausente, então Existe não
// consegue separar os dois. Por isso o servidor confere o bucket ao subir e não sobe sem ele.
func TestS3VerificarRecusaBucketInexistente(t *testing.T) {
	c, _ := s3Falso(t)
	if err := (S3{Cliente: c, Bucket: "lisura"}).Verificar(context.Background()); err != nil {
		t.Fatalf("bucket existente recusado: %v", err)
	}
	if err := (S3{Cliente: c, Bucket: "bucket-que-nao-existe"}).Verificar(context.Background()); err == nil {
		t.Fatal("bucket inexistente aceito")
	}
}
