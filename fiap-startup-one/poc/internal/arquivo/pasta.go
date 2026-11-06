package arquivo

import (
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// Pasta serve arquivos de um diretório local para o http.FileServer, com duas diferenças do
// http.Dir:
//
//   - não abre diretórios: o http.Dir lista o conteúdo de uma pasta, e GET /publico/cartoes/
//     entregava o endereço de todos os cartões (medido em 04/11: 2.994 links), anulando o código
//     secreto de cada um. Na nuvem o S3 não lista (o CloudFront não tem permissão para isso);
//   - no Windows, abre com FILE_SHARE_DELETE (ver abrir_windows.go), para a leitura não falhar
//     enquanto a fila republica o mesmo arquivo.
type Pasta string

func (p Pasta) Open(nome string) (http.File, error) {
	// Mesma higiene de caminho do http.Dir: nada de separador do sistema e nada acima da raiz.
	if filepath.Separator != '/' && strings.ContainsRune(nome, filepath.Separator) {
		return nil, fs.ErrNotExist
	}
	limpo := path.Clean("/" + nome)
	if limpo == "/" || strings.HasSuffix(nome, "/") {
		return nil, fs.ErrNotExist
	}
	// Qualquer falha vira "não existe": nem um 403 diz a quem testa endereços que a pasta existe.
	f, err := AbrirParaLer(filepath.Join(string(p), filepath.FromSlash(limpo)))
	if err != nil {
		return nil, fs.ErrNotExist
	}
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		f.Close()
		return nil, fs.ErrNotExist
	}
	return f, nil
}
