import type { NextConfig } from "next";

// A API é o servidor Go (../poc). Na máquina local o Next repassa /api para ele, então front e API
// ficam no mesmo endereço para o navegador — e o cookie de sessão viaja sozinho. Na nuvem quem
// faz isso é o CloudFront (/api → balanceador → Go, /publico → S3) e estas regras nem são usadas.
const API = process.env.LISURA_API ?? "http://localhost:8082";
// O executor de ensaios (cmd/ensaiador), só na máquina local: a página /operacao fala com ele pelo
// mesmo endereço do front, e ele só aceita pedidos vindos daqui.
const ENSAIADOR = process.env.LISURA_ENSAIADOR ?? "http://127.0.0.1:8083";

const nextConfig: NextConfig = {
  // Gera um servidor Node autônomo em .next/standalone: é o que vai na imagem do contêiner.
  output: "standalone",
  // Desligado (vinha ligado no gerador): o modo de cache serve a páginas renderizadas no servidor;
  // aqui as telas rodam no navegador e buscam /api depois de abrir, então ele não traz ganho e
  // só acusava o Math.random() interno do Ant Design e a "navegação instantânea".
  cacheComponents: false,
  async rewrites() {
    return [
      { source: "/api/:caminho*", destination: `${API}/api/:caminho*` },
      // Arquivos publicados (dados do certame, cartão, relação): na produção vêm do CDN.
      { source: "/publico/:caminho*", destination: `${API}/publico/:caminho*` },
      { source: "/ensaiador/:caminho*", destination: `${ENSAIADOR}/:caminho*` },
    ];
  },
  // Os componentes: o Storybook estático em public/design (npm run design). Redireciona, não repassa:
  // os arquivos dele são relativos, e de /design (sem a barra) o navegador os procuraria na raiz.
  async redirects() {
    return [{ source: "/design", destination: "/design/index.html", permanent: false }];
  },
};

export default nextConfig;
