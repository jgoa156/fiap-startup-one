-- +goose Up

-- O que aconteceu com o aviso depois de sair: o SES conta, pelo SNS, a entrega, a devolução (bounce:
-- o servidor do destinatário recusou) e a reclamação (o destinatário marcou como spam). A prova do
-- aviso (notificacao) não muda; o que vem depois fica aqui, ligado pelo id do envio no provedor.
-- Também não muda: é o que mostra que o aviso não chegou, e por quê.
CREATE TABLE notificacao_ocorrencia (
    id            bigserial PRIMARY KEY,
    sns_id        text NOT NULL,             -- id da mensagem no SNS: a mesma entregue de novo é a mesma linha
    provedor_id   text NOT NULL,             -- o MessageId do SES, = notificacao.provedor_id
    tipo          text NOT NULL CHECK (tipo IN ('ENTREGUE', 'DEVOLVIDA', 'DEVOLVIDA_TEMPORARIA', 'RECLAMACAO')),
    destinatario  text NOT NULL,
    detalhe       text,
    ocorrida_em   timestamptz NOT NULL,
    recebida_em   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (sns_id, destinatario)
);
CREATE INDEX notificacao_ocorrencia_por_envio ON notificacao_ocorrencia (provedor_id);
CREATE TRIGGER notificacao_ocorrencia_imutavel BEFORE UPDATE OR DELETE ON notificacao_ocorrencia
    FOR EACH ROW EXECUTE FUNCTION trilha_imutavel();

-- Endereço que devolveu de vez ou reclamou: não recebe mais avisos. Insistir derruba a reputação do
-- remetente no SES, e o SES pode pausar o envio da conta inteira. O candidato vê na área dele que o
-- aviso não chegou, e um e-mail novo não está aqui.
CREATE TABLE email_suprimido (
    email          text PRIMARY KEY CHECK (email = lower(email)),
    motivo         text NOT NULL CHECK (motivo IN ('DEVOLVIDA', 'RECLAMACAO')),
    ocorrencia_id  bigint NOT NULL REFERENCES notificacao_ocorrencia (id),
    desde          timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE email_suprimido;
DROP TABLE notificacao_ocorrencia;
