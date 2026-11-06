package arquivo

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
)

// O mesmo contrato nos dois cofres: grava, lê, lista em ordem e recusa a segunda gravação na mesma
// chave — também quando as duas chegam ao mesmo tempo.
func conferirCofre(t *testing.T, c Cofre) {
	ctx := context.Background()
	if l, err := c.Listar(ctx, "certame-1/"); err != nil || len(l) != 0 {
		t.Fatalf("cofre vazio listou %v (%v)", l, err)
	}
	for _, k := range []string{"certame-1/000002.txt", "certame-1/000001.txt", "certame-2/000001.txt"} {
		if err := c.Criar(ctx, k, []byte("conteúdo de "+k)); err != nil {
			t.Fatal(k, err)
		}
	}
	if err := c.Criar(ctx, "certame-1/000001.txt", []byte("reescrita")); !errors.Is(err, ErrJaExiste) {
		t.Fatalf("sobrescrever: err = %v, esperado ErrJaExiste", err)
	}
	b, err := c.Ler(ctx, "certame-1/000001.txt")
	if err != nil || string(b) != "conteúdo de certame-1/000001.txt" {
		t.Fatalf("leitura depois da tentativa de sobrescrever: %q %v", b, err)
	}
	l, err := c.Listar(ctx, "certame-1/")
	if err != nil || !slices.Equal(l, []string{"certame-1/000001.txt", "certame-1/000002.txt"}) {
		t.Fatalf("listar: %v %v", l, err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	gravaram := 0
	for i := range 8 {
		wg.Go(func() {
			err := c.Criar(ctx, "certame-3/000001.txt", []byte{byte('a' + i)})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				gravaram++
			case !errors.Is(err, ErrJaExiste):
				t.Errorf("gravação concorrente: %v", err)
			}
		})
	}
	wg.Wait()
	if gravaram != 1 {
		t.Fatalf("%d gravações concorrentes venceram; só uma pode", gravaram)
	}
}

func TestCofreDiscoNaoSobrescreve(t *testing.T) {
	conferirCofre(t, CofreDisco{Dir: t.TempDir()})
}

func TestCofreS3NaoSobrescreve(t *testing.T) {
	c, cabecalhos := s3Falso(t)
	conferirCofre(t, CofreS3{Cliente: c, Bucket: "lisura", Prefixo: "ancoras/"})
	if got := cabecalhos["/lisura/ancoras/certame-1/000001.txt"].Get("If-None-Match"); got != "*" {
		t.Errorf("If-None-Match enviado = %q: no S3 real, o PUT criaria uma versão nova por cima", got)
	}
}
