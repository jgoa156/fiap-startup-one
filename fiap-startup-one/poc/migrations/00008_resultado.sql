-- +goose Up

-- Reserva de vagas por cota, parametrizada pelo edital. Na esfera federal: pessoas pretas, pardas,
-- indígenas e quilombolas 30%, fração ≥ 0,5 arredonda para cima, só com 2 vagas ou mais (Lei
-- 15.142/2025, arts. 1º e 5º); pessoa com deficiência no mínimo 5%, fração sempre para cima (Decreto
-- 9.508/2018, art. 1º §§ 1º e 3º). Município e estado seguem lei própria — por isso é parâmetro.
ALTER TABLE cota
    ADD COLUMN reserva_percentual numeric(5,2) NOT NULL DEFAULT 0
        CHECK (reserva_percentual >= 0 AND reserva_percentual <= 100),
    ADD COLUMN arredondamento text NOT NULL DEFAULT 'meio' CHECK (arredondamento IN ('cima', 'meio')),
    ADD COLUMN minimo_vagas int NOT NULL DEFAULT 1 CHECK (minimo_vagas >= 1);

-- Critérios de pontuação do cargo: nota mínima (em pontos) e desempate depois do critério legal da
-- idade (Estatuto da Pessoa Idosa, art. 27, parágrafo único), na ordem do edital.
ALTER TABLE cargo
    ADD COLUMN nota_minima numeric(8,2) NOT NULL DEFAULT 0 CHECK (nota_minima >= 0),
    ADD COLUMN desempate text[] NOT NULL DEFAULT '{}';

-- A prova objetiva de cada cargo: uma linha por questão.
CREATE TABLE questao (
    cargo_id    bigint NOT NULL REFERENCES cargo (id),
    numero      int NOT NULL CHECK (numero > 0),
    disciplina  text NOT NULL,
    peso        numeric(6,2) NOT NULL DEFAULT 1 CHECK (peso > 0),
    PRIMARY KEY (cargo_id, numero)
);

-- O gabarito só cresce, como o edital: alterar ou anular questão publica uma versão nova. Uma letra
-- por questão, '*' para anulada (o ponto vai para todos os presentes).
CREATE TABLE gabarito (
    cargo_id      bigint NOT NULL REFERENCES cargo (id),
    versao        int NOT NULL CHECK (versao > 0),
    tipo          text NOT NULL CHECK (tipo IN ('preliminar', 'definitivo')),
    respostas     text NOT NULL CHECK (respostas ~ '^[A-E*]+$'),
    motivo        text NOT NULL,
    publicado_em  timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (cargo_id, versao)
);
CREATE TRIGGER gabarito_imutavel BEFORE UPDATE OR DELETE ON gabarito
    FOR EACH ROW EXECUTE FUNCTION trilha_imutavel();

-- Folha de respostas como a leitora óptica da banca entrega (a leitura física fica com a banca):
-- uma letra por questão, '.' em branco, '*' rasura ou marcação dupla. Sem folha = ausente.
CREATE TABLE folha_resposta (
    inscricao_id  bigint PRIMARY KEY REFERENCES inscricao (id),
    respostas     text NOT NULL CHECK (respostas ~ '^[A-E.*]+$'),
    lote          text NOT NULL,
    importada_em  timestamptz NOT NULL DEFAULT now()
);

-- Recálculo versionado: cada cálculo é uma versão nova, e as antigas ficam — a classificação de
-- ontem precisa poder ser mostrada a um juiz amanhã.
CREATE TABLE resultado_versao (
    certame_id    bigint NOT NULL REFERENCES certame (id),
    versao        int NOT NULL CHECK (versao > 0),
    tipo          text NOT NULL CHECK (tipo IN ('preliminar', 'definitivo')),
    gabaritos     jsonb NOT NULL, -- {cargo_id: versão do gabarito usada}
    motivo        text NOT NULL,
    calculado_em  timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (certame_id, versao)
);

CREATE TABLE resultado_item (
    certame_id      bigint NOT NULL,
    versao          int NOT NULL,
    inscricao_id    bigint NOT NULL REFERENCES inscricao (id),
    cargo_id        bigint NOT NULL REFERENCES cargo (id),
    situacao        text NOT NULL CHECK (situacao IN ('CLASSIFICADO', 'CADASTRO_RESERVA', 'ELIMINADO', 'AUSENTE')),
    nota            numeric(8,2),
    acertos         int,
    por_disciplina  jsonb,
    posicao_ampla   int,
    vaga            text,    -- dentro das vagas: 'ampla' ou o código da cota
    empate          boolean NOT NULL DEFAULT false, -- empate que os critérios não desfazem (sorteio)
    PRIMARY KEY (certame_id, versao, inscricao_id),
    FOREIGN KEY (certame_id, versao) REFERENCES resultado_versao (certame_id, versao),
    CHECK ((situacao IN ('CLASSIFICADO', 'CADASTRO_RESERVA')) = (posicao_ampla IS NOT NULL)),
    CHECK ((situacao = 'CLASSIFICADO') = (vaga IS NOT NULL))
);

-- Posição de cada cotista na lista da sua cota (a lei manda publicar as duas listas).
CREATE TABLE resultado_posicao_cota (
    certame_id    bigint NOT NULL,
    versao        int NOT NULL,
    inscricao_id  bigint NOT NULL,
    cota_codigo   text NOT NULL,
    posicao       int NOT NULL CHECK (posicao > 0),
    PRIMARY KEY (certame_id, versao, inscricao_id, cota_codigo),
    FOREIGN KEY (certame_id, versao, inscricao_id) REFERENCES resultado_item (certame_id, versao, inscricao_id)
);

CREATE TRIGGER resultado_versao_imutavel BEFORE UPDATE OR DELETE ON resultado_versao
    FOR EACH ROW EXECUTE FUNCTION trilha_imutavel();
CREATE TRIGGER resultado_item_imutavel BEFORE UPDATE OR DELETE ON resultado_item
    FOR EACH ROW EXECUTE FUNCTION trilha_imutavel();
CREATE TRIGGER resultado_posicao_cota_imutavel BEFORE UPDATE OR DELETE ON resultado_posicao_cota
    FOR EACH ROW EXECUTE FUNCTION trilha_imutavel();

-- +goose Down
DROP TABLE resultado_posicao_cota;
DROP TABLE resultado_item;
DROP TABLE resultado_versao;
DROP TABLE folha_resposta;
DROP TABLE gabarito;
DROP TABLE questao;
ALTER TABLE cargo DROP COLUMN desempate, DROP COLUMN nota_minima;
ALTER TABLE cota DROP COLUMN minimo_vagas, DROP COLUMN arredondamento, DROP COLUMN reserva_percentual;
