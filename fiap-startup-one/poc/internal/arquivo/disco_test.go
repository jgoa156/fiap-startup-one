package arquivo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

// A fila republica o certame enquanto a base inteira o lê (o pico do cartão e do resultado): quem lê
// nunca pode receber erro. No Windows, com o os.Open padrão, ~7% das leituras falhavam.
func TestLeituraNaoFalhaDuranteRepublicacao(t *testing.T) {
	d := Disco{Dir: t.TempDir()}
	ctx := context.Background()
	if err := d.Put(ctx, "certames/1.json", []byte("0")); err != nil {
		t.Fatal(err)
	}
	alvo := filepath.Join(d.Dir, "certames", "1.json")
	var leituras, erros atomic.Int64
	parar := make(chan struct{})
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-parar:
					return
				default:
				}
				f, err := AbrirParaLer(alvo)
				leituras.Add(1)
				if err != nil {
					if erros.Add(1) == 1 {
						t.Logf("primeiro erro: %v (%#v)", err, errors.Unwrap(err))
					}
					continue
				}
				f.Close()
			}
		}()
	}
	gravadas := 0
	for range 100 {
		if d.Put(ctx, "certames/1.json", []byte("x")) == nil {
			gravadas++
		}
	}
	close(parar)
	wg.Wait()
	// Uma gravação que ainda assim falhe não perde nada: a tarefa da fila repete com espera.
	t.Logf("%d leituras, %d erros; %d de 100 gravações entraram de primeira", leituras.Load(), erros.Load(), gravadas)
	if erros.Load() > 0 {
		t.Fatalf("%d de %d leituras falharam durante a republicação", erros.Load(), leituras.Load())
	}
}

func TestPastaNaoListaDiretorios(t *testing.T) {
	d := Disco{Dir: t.TempDir()}
	ctx := context.Background()
	if err := d.Put(ctx, "cartoes/abc123.html", []byte("<h1>cartão</h1>")); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.FileServer(Pasta(d.Dir)))
	defer srv.Close()
	for caminho, esperado := range map[string]int{
		"/cartoes/abc123.html":     200,
		"/cartoes/":                404, // listaria todos os cartões
		"/cartoes":                 404,
		"/":                        404,
		"/../disco.go":             404,
		"/cartoes/nao-existe.html": 404,
	} {
		res, err := http.Get(srv.URL + caminho)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != esperado {
			t.Errorf("GET %s = %d, esperava %d", caminho, res.StatusCode, esperado)
		}
	}
}
