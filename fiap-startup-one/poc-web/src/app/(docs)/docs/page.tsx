import type { Metadata } from "next";
import { Entrada } from "./Entrada";

export const metadata: Metadata = { title: "Visão geral" };

export default function PaginaDocs() {
  return <Entrada />;
}
