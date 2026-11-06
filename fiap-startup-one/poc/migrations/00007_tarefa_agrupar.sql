-- +goose Up
-- A janela de agrupamento passa a morar na tarefa. Sem isso, quando chegava mudança durante a
-- execução, a nova rodada era marcada para "já" e furava a janela: sob carga a relação de
-- inscritos rodava em laço (medido em 05/11: ~100 republicações/min com janela de 30 s).
ALTER TABLE tarefa ADD COLUMN agrupar_s double precision NOT NULL DEFAULT 0 CHECK (agrupar_s >= 0);

-- +goose Down
ALTER TABLE tarefa DROP COLUMN agrupar_s;
