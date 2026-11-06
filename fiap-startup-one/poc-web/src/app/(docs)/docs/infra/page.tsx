import type { Metadata } from "next";
import { Infra } from "./Infra";

export const metadata: Metadata = { title: "Infraestrutura" };

export default function PaginaInfra() {
  return <Infra />;
}
