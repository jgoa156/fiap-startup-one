import type { Metadata } from "next";
import { Painel } from "./Painel";

export const metadata: Metadata = { title: "Operação e ensaios" };

export default function PaginaOperacao() {
  return <Painel />;
}
