//go:build !windows

package arquivo

import "os"

// AbrirParaLer é o os.Open: fora do Windows, trocar um arquivo aberto por rename já funciona.
func AbrirParaLer(nome string) (*os.File, error) { return os.Open(nome) }
