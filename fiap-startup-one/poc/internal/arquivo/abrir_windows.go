//go:build windows

package arquivo

import (
	"errors"
	"os"
	"syscall"
	"time"
)

// AbrirParaLer abre só para leitura deixando outro processo apagar ou trocar o arquivo enquanto ele
// está aberto. O os.Open do Go não permite (sem FILE_SHARE_DELETE) e, enquanto a fila republica o
// arquivo, a leitura falhava: medido em 04/11, 3.076 leituras com erro contra 1 com esta abertura.
// Diretórios não abrem (falta FILE_FLAG_BACKUP_SEMANTICS), que é o que a Pasta quer.
//
// O 1 que sobrava é o instante logo depois da troca, em que o arquivo antigo ainda está sendo
// apagado e abrir o nome dá acesso negado (medido em 06/11 com 6 cópias do teste ao mesmo tempo: 11
// de 90 rodadas com 1 ou 2 erros, todos ERROR_ACCESS_DENIED). Esse erro é tentado de novo por até
// ~10 ms; uma falta de permissão de verdade só demora esse tanto para aparecer.
func AbrirParaLer(nome string) (*os.File, error) {
	p, err := syscall.UTF16PtrFromString(nome)
	if err != nil {
		return nil, err
	}
	for tentativa := 0; ; tentativa++ {
		h, err := syscall.CreateFile(p, syscall.GENERIC_READ,
			syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
			nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
		if err == nil {
			return os.NewFile(uintptr(h), nome), nil
		}
		if !errors.Is(err, syscall.ERROR_ACCESS_DENIED) || tentativa == 10 {
			return nil, &os.PathError{Op: "open", Path: nome, Err: err}
		}
		time.Sleep(time.Millisecond)
	}
}
