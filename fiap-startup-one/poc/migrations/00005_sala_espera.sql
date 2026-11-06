-- +goose Up
-- Sala de espera compartilhada: com várias instâncias da API atrás do balanceador, o nível e a
-- numeração das senhas precisam ser um só. Cada instância lê esta linha a cada segundo; o banco só
-- é tocado de novo quando um visitante sem senha chega (nextval, que não trava ninguém).
CREATE SEQUENCE senha_fila;

CREATE TABLE sala_espera (
    id       int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    nivel    int NOT NULL DEFAULT 0 CHECK (nivel BETWEEN 0 AND 3),
    taxa     double precision NOT NULL DEFAULT 0 CHECK (taxa >= 0),
    folga    bigint NOT NULL DEFAULT 0 CHECK (folga >= 0),
    abriu_em timestamptz NOT NULL DEFAULT now(),
    base     bigint NOT NULL DEFAULT 0,
    mudou_em timestamptz NOT NULL DEFAULT now()
);
INSERT INTO sala_espera DEFAULT VALUES;

-- +goose Down
DROP TABLE sala_espera;
DROP SEQUENCE senha_fila;
