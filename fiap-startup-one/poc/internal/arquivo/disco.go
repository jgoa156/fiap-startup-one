// Package arquivo guarda conteúdo binário. Na POC, em disco; na arquitetura alvo, em S3 — a
// interface é a mesma (Put por chave).
package arquivo

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Disco struct{ Dir string }

// Put grava de forma atômica: escreve num temporário e renomeia, para nunca expor arquivo parcial.
func (d Disco) Put(_ context.Context, chave string, conteudo []byte) error {
	destino := filepath.Join(d.Dir, filepath.FromSlash(chave))
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
	// No Windows, renomear por cima de um arquivo aberto por alguém falha enquanto ele está aberto
	// (medido em 04/11: 2.816 de 3.000 trocas falharam com 8 leitores sem pausa). Tenta de novo por
	// ~200 ms; se ainda falhar, a tarefa da fila repete com espera crescente. No Linux e no S3 a troca
	// é atômica e passa na primeira.
	for tentativa := 0; ; tentativa++ {
		err = os.Rename(tmp.Name(), destino)
		if err == nil || tentativa == 10 {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (d Disco) Existe(_ context.Context, chave string) (bool, error) {
	_, err := os.Stat(filepath.Join(d.Dir, filepath.FromSlash(chave)))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}
