-- +goose Up

-- Recurso contra o gabarito preliminar: um por questão por inscrição, com fundamento padronizado. O
-- fundamento escolhido de uma lista (e não texto livre) é o que deixa agrupar os 500 recursos sobre
-- a mesma questão numa análise só (02-produto §6).
CREATE TABLE recurso_questao (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    inscricao_id     bigint NOT NULL REFERENCES inscricao (id),
    cargo_id         bigint NOT NULL,
    gabarito_versao  int NOT NULL,
    numero           int NOT NULL,
    fundamento       text NOT NULL CHECK (fundamento IN (
                         'gabarito_errado', 'mais_de_uma_correta', 'nenhuma_correta',
                         'fora_do_conteudo', 'enunciado_ambiguo')),
    resposta_pedida  char(1) CHECK (resposta_pedida ~ '^[A-E]$'),
    argumento        text NOT NULL CHECK (length(argumento) BETWEEN 1 AND 4000),
    criado_em        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (inscricao_id, gabarito_versao, numero),
    FOREIGN KEY (cargo_id, gabarito_versao) REFERENCES gabarito (cargo_id, versao),
    FOREIGN KEY (cargo_id, numero) REFERENCES questao (cargo_id, numero),
    CHECK ((fundamento = 'gabarito_errado') = (resposta_pedida IS NOT NULL))
);
CREATE INDEX ON recurso_questao (cargo_id, gabarito_versao, numero);

-- Uma decisão por questão, que responde a todos os recursos dela de uma vez — e vale também para
-- quem não recorreu. A fundamentação é a resposta padronizada que cada recorrente recebe.
CREATE TABLE decisao_questao (
    cargo_id         bigint NOT NULL,
    gabarito_versao  int NOT NULL,
    numero           int NOT NULL,
    decisao          text NOT NULL CHECK (decisao IN ('mantem', 'altera', 'anula')),
    nova_resposta    char(1) CHECK (nova_resposta ~ '^[A-E]$'),
    fundamentacao    text NOT NULL CHECK (length(fundamentacao) > 0),
    julgado_por      text NOT NULL,
    julgado_em       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (cargo_id, gabarito_versao, numero),
    FOREIGN KEY (cargo_id, gabarito_versao) REFERENCES gabarito (cargo_id, versao),
    FOREIGN KEY (cargo_id, numero) REFERENCES questao (cargo_id, numero),
    CHECK ((decisao = 'altera') = (nova_resposta IS NOT NULL))
);
CREATE TRIGGER decisao_questao_imutavel BEFORE UPDATE OR DELETE ON decisao_questao
    FOR EACH ROW EXECUTE FUNCTION trilha_imutavel();

-- +goose Down
DROP TABLE decisao_questao;
DROP TABLE recurso_questao;
