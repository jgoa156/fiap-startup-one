import type { Metadata } from "next";
import { Referencia } from "./Referencia";

export const metadata: Metadata = { title: "API" };

// A referência da API, desenhada pelo Scalar a partir do openapi.yaml que o próprio servidor serve
// (GET /api/openapi.yaml, embutido no binário). O teste TestOpenAPICobreTodasAsRotas confere a
// especificação contra as rotas reais: o que está aqui é o que o servidor responde.
export default function PaginaApi() {
  return <Referencia />;
}
