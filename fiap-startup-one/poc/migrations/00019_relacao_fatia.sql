-- +goose NO TRANSACTION
-- +goose Up
-- As fatias da relação de inscritos de cada certame, guardadas, em vez de deduzidas das inscrições.
--
-- A 00018 criou (certame_id, id) para o índice da relação achar as fatias pulando de uma para a
-- outra. O planejador não o usou: medido em 06/11 no certame 5 (491 mil inscrições, 45% da tabela), ele
-- preferiu a chave primária, que é só por id, e cada salto atravessava as inscrições dos outros
-- certames intercaladas ali (10.510 linhas descartadas por salto, 271 ms o índice inteiro; com
-- comparação de linha, 8,6 s). O certame grande é justamente o que importa. Em vez de brigar com a
-- estimativa, a fatia é anotada quando a primeira inscrição dela pede publicação, e o índice da
-- relação vira uma leitura desta tabela. O índice da 00018 sai: não servia e custava uma escrita em
-- cada inscrição.
DROP INDEX CONCURRENTLY IF EXISTS inscricao_por_certame_e_id;

-- tamanho é o publicacao.TamanhoFatia com que a fatia foi anotada: mudar o tamanho não deixa o
-- índice apontar fatias de outro tamanho.
CREATE TABLE IF NOT EXISTS relacao_fatia (
    certame_id  bigint NOT NULL REFERENCES certame (id),
    tamanho     bigint NOT NULL CHECK (tamanho > 0),
    fatia       bigint NOT NULL CHECK (fatia >= 0),
    PRIMARY KEY (certame_id, tamanho, fatia)
);

-- +goose Down
DROP TABLE IF EXISTS relacao_fatia;
CREATE INDEX CONCURRENTLY IF NOT EXISTS inscricao_por_certame_e_id ON inscricao (certame_id, id);
