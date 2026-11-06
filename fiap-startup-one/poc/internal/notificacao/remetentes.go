package notificacao

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Memoria é o provedor dos testes: guarda as mensagens, deduplica pela chave e simula os defeitos de
// um provedor real (a resposta que se perde depois do envio, o endereço recusado).
type Memoria struct {
	Falhar atomic.Int32 // as próximas N entregas dão erro depois de enviar (resposta perdida)

	mu        sync.Mutex
	enviadas  map[string]Mensagem
	ids       map[string]string
	recusados map[string]bool
}

// Recusar faz o provedor recusar, de vez, o endereço.
func (m *Memoria) Recusar(endereco string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recusados == nil {
		m.recusados = map[string]bool{}
	}
	m.recusados[endereco] = true
}

func (m *Memoria) Enviar(_ context.Context, msg Mensagem) (string, error) {
	m.mu.Lock()
	if m.recusados[msg.Para] {
		m.mu.Unlock()
		return "", fmt.Errorf("%w: %s", ErrEnderecoRecusado, msg.Para)
	}
	if m.enviadas == nil {
		m.enviadas, m.ids = map[string]Mensagem{}, map[string]string{}
	}
	id, ja := m.ids[msg.Chave]
	if !ja {
		id = "mem-" + msg.Chave // único entre provedores de teste, como o id de um provedor real
		m.ids[msg.Chave], m.enviadas[msg.Chave] = id, msg
	}
	m.mu.Unlock()
	if !ja && m.Falhar.Load() > 0 && m.Falhar.Add(-1) >= 0 {
		return "", errors.New("provedor instável (simulado): a mensagem saiu, a resposta não chegou")
	}
	return id, nil
}

// Enviadas devolve as mensagens distintas que saíram.
func (m *Memoria) Enviadas() []Mensagem {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Mensagem, 0, len(m.enviadas))
	for _, msg := range m.enviadas {
		out = append(out, msg)
	}
	return out
}

// Caixa é o provedor da máquina local: grava cada mensagem numa linha JSON de um arquivo (a caixa de
// saída) e o relê ao abrir, para a mesma chave continuar sendo o mesmo envio depois de reiniciar.
type Caixa struct {
	mu  sync.Mutex
	f   *os.File
	ids map[string]string
}

type linhaCaixa struct {
	ID        string    `json:"id"`
	Chave     string    `json:"chave"`
	Para      string    `json:"para"`
	Assunto   string    `json:"assunto"`
	Texto     string    `json:"texto"`
	EnviadaEm time.Time `json:"enviada_em"`
}

func AbrirCaixa(caminho string) (*Caixa, error) {
	c := &Caixa{ids: map[string]string{}}
	if f, err := os.Open(caminho); err == nil {
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 0, 64<<10), 1<<20)
		for s.Scan() {
			var l linhaCaixa
			if json.Unmarshal(s.Bytes(), &l) == nil {
				c.ids[l.Chave] = l.ID
			} // linha final pela metade (processo morto no meio): ignorada, a chave é reenviada
		}
		f.Close()
		if err := s.Err(); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	f, err := os.OpenFile(caminho, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	c.f = f
	return c, nil
}

func (c *Caixa) Enviar(_ context.Context, msg Mensagem) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if id, ok := c.ids[msg.Chave]; ok {
		return id, nil
	}
	id := fmt.Sprintf("caixa-%08d", len(c.ids)+1)
	b, _ := json.Marshal(linhaCaixa{ID: id, Chave: msg.Chave, Para: msg.Para, Assunto: msg.Assunto, Texto: msg.Texto, EnviadaEm: time.Now()})
	if _, err := c.f.Write(append(b, '\n')); err != nil {
		return "", err
	}
	c.ids[msg.Chave] = id
	return id, nil
}

func (c *Caixa) Fechar() error { return c.f.Close() }

// Descarte é o provedor do ensaio de carga: aceita tudo, conta e não guarda o texto. O ensaio mede a
// fila com os avisos dentro, como em produção, sem um provedor de e-mail de verdade no caminho.
type Descarte struct{ Enviadas atomic.Int64 }

func (d *Descarte) Enviar(_ context.Context, msg Mensagem) (string, error) {
	d.Enviadas.Add(1)
	return "descarte-" + msg.Chave, nil
}
