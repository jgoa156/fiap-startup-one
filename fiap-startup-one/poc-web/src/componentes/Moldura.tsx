"use client";

import { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Avatar, Button, Drawer, Grid, Input, Layout, Menu, Modal, Spin, Tooltip, Typography } from "antd";
import type { InputRef, MenuProps } from "antd";
import {
  FileSearchOutlined,
  HomeOutlined,
  LogoutOutlined,
  MenuOutlined,
  QuestionOutlined,
  SafetyCertificateFilled,
  SearchOutlined,
} from "@ant-design/icons";
import { api, ErroApi, EVENTO_DEGRADADO, formatarCPF, useApi, type Eu } from "@/lib/api";
import { ModoConsulta, SalaDeEspera } from "./Espera";

// Moldura da área logada: a do protótipo do Inclusa (mockups/antd-heroui-sidenav) — lateral em
// cartão flutuante que recolhe para ícones, barra superior com busca Ctrl+K e ajuda "?". No
// celular a lateral vira gaveta.

type Ajuda = { titulo: string; texto: React.ReactNode };

const SessaoCtx = createContext<{ eu: Eu; recarregarEu: () => void } | null>(null);
const AjudaCtx = createContext<(a: Ajuda) => void>(() => {});

export function useSessao() {
  const s = useContext(SessaoCtx);
  if (!s) throw new Error("useSessao fora da moldura");
  return s;
}

// Cada tela declara o texto do seu "?" (Inclusa, doc 01 §5: ajuda em toda tela).
// O texto é registrado uma vez, quando a tela abre: ele é JSX novo a cada render e, se fosse
// dependência do efeito, cada registro renderizaria a moldura de novo, sem fim.
export function useAjuda(titulo: string, texto: React.ReactNode) {
  const definir = useContext(AjudaCtx);
  const textoRef = useRef(texto);
  useEffect(() => definir({ titulo, texto: textoRef.current }), [definir, titulo]);
}

const secoes: { titulo: string; itens: [string, string, React.ReactNode][] }[] = [
  {
    titulo: "Área do candidato",
    itens: [
      ["/", "Início", <HomeOutlined key="i" />],
      ["/certames", "Certames abertos", <FileSearchOutlined key="c" />],
    ],
  },
];
const todos = secoes.flatMap((s) => s.itens);

type ItemDoMenu = NonNullable<MenuProps["items"]>[number];

function itensDoMenu(recolhido: boolean): ItemDoMenu[] {
  return secoes.flatMap<ItemDoMenu>((s, i) => {
    const filhos = s.itens.map(([key, label, icon]) => ({ key, icon, label: <Link href={key}>{label}</Link> }));
    if (recolhido) return i === 0 ? filhos : [{ type: "divider" as const }, ...filhos];
    return [{ type: "group" as const, label: s.titulo, children: filhos }];
  });
}

function ativo(caminho: string) {
  const achado = todos.filter(([k]) => (k === "/" ? caminho === "/" : caminho.startsWith(k)));
  return achado.sort((a, b) => b[0].length - a[0].length)[0];
}

function lerRecolhido() {
  try {
    return localStorage.getItem("lisura.lateral") === "1";
  } catch {
    return false;
  }
}

export function Moldura({ children }: { children: React.ReactNode }) {
  const caminho = usePathname();
  const router = useRouter();
  const telas = Grid.useBreakpoint();
  const celular = telas.lg === false;
  const { dados: eu, erro, recarregar } = useApi<Eu>("/eu");
  // Lido no primeiro render: a moldura só aparece depois de /eu responder, então o servidor
  // nunca renderiza a lateral e não há divergência na hidratação.
  const [recolhido, setRecolhido] = useState(lerRecolhido);
  const [gaveta, setGaveta] = useState(false);
  const [ajuda, setAjuda] = useState<Ajuda>({ titulo: "Ajuda", texto: "" });
  const [verAjuda, setVerAjuda] = useState(false);
  const busca = useRef<InputRef>(null);
  const [barrado, setBarrado] = useState<ErroApi>();
  const [rodada, setRodada] = useState(0);

  useEffect(() => {
    const ouvir = (e: Event) => setBarrado((e as CustomEvent<ErroApi>).detail);
    window.addEventListener(EVENTO_DEGRADADO, ouvir);
    return () => window.removeEventListener(EVENTO_DEGRADADO, ouvir);
  }, []);
  // Se ainda estiver na fila, o próprio api() avisa de novo, com a posição nova.
  const tentarDeNovo = useCallback(async () => {
    try {
      await api("/eu");
    } catch (e) {
      if (!(e instanceof ErroApi) || e.status !== 401) return;
    }
    setBarrado(undefined);
    setRodada((r) => r + 1);
    recarregar();
  }, [recarregar]);

  useEffect(() => {
    // Volta para o endereço completo, busca incluída.
    if (erro?.status === 401) router.replace(`/entrar?volta=${encodeURIComponent(caminho + window.location.search)}`);
  }, [erro, caminho, router]);
  useEffect(() => {
    const atalho = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        busca.current?.focus();
      }
    };
    window.addEventListener("keydown", atalho);
    return () => window.removeEventListener("keydown", atalho);
  }, []);

  // Fila ou modo só consulta tomam a tela inteira, venha o 503 do /eu ou de qualquer chamada da
  // tela. Quando a API volta a aceitar, a tela é montada de novo e busca tudo outra vez.
  const bloqueio = barrado ?? (!eu ? erro : undefined);
  if (bloqueio?.fila) return <SalaDeEspera fila={bloqueio.fila} tentar={tentarDeNovo} />;
  if (bloqueio?.degradado === 3) return <ModoConsulta aoVoltar={tentarDeNovo} />;
  if (!eu) {
    return (
      <div style={{ minHeight: "100vh", display: "grid", placeItems: "center" }}>
        {erro && erro.status !== 401 ? (
          <Typography.Text type="danger">{erro.message}</Typography.Text>
        ) : (
          <Spin size="large" description="Carregando…">
            <div style={{ width: 120 }} />
          </Spin>
        )}
      </div>
    );
  }

  const alternar = () => {
    const v = !recolhido;
    setRecolhido(v);
    try {
      localStorage.setItem("lisura.lateral", v ? "1" : "0");
    } catch {}
  };
  const sair = async () => {
    await api("/sessoes/sair", { metodo: "POST" });
    router.replace("/entrar");
  };
  const atual = ativo(caminho);
  const iniciais = eu.dados.nome
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0].toUpperCase())
    .join("");
  const fechado = recolhido && !celular;

  const marca = (
    <div className="marca">
      {!fechado && (
        <Link href="/" className="marca-nome">
          <SafetyCertificateFilled style={{ color: "var(--marker-400)", fontSize: 24 }} /> Lisura
        </Link>
      )}
      {!celular && (
        <Tooltip title={recolhido ? "Expandir menu" : "Recolher menu"} placement="right">
          <Button
            type="text"
            shape="circle"
            aria-label={recolhido ? "Expandir menu" : "Recolher menu"}
            aria-expanded={!recolhido}
            icon={<MenuOutlined style={{ color: "var(--ink-600)" }} />}
            onClick={alternar}
          />
        </Tooltip>
      )}
    </div>
  );
  const menu = (
    <Menu
      mode="inline"
      inlineCollapsed={fechado}
      selectedKeys={atual ? [atual[0]] : []}
      items={itensDoMenu(fechado)}
      onClick={() => setGaveta(false)} // no celular, escolher um item fecha a gaveta
    />
  );
  const avatar = (
    <Avatar style={{ background: "#fff", color: "var(--brand-700)", fontWeight: 700, boxShadow: "var(--shadow-sm)" }}>
      {iniciais}
    </Avatar>
  );
  const perfil = (
    <div className="perfil">
      {fechado ? (
        <Tooltip title={`${eu.dados.nome} · sair`} placement="right">
          <Button type="text" shape="circle" onClick={sair} aria-label="Sair" style={{ padding: 0, height: "auto" }}>
            {avatar}
          </Button>
        </Tooltip>
      ) : (
        <>
          <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
            {avatar}
            <div style={{ minWidth: 0 }}>
              <div style={{ fontWeight: 700, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {eu.dados.nome}
              </div>
              <div style={{ fontSize: 12, color: "var(--ink-600)" }}>CPF {formatarCPF(eu.cpf)}</div>
            </div>
          </div>
          <Button type="link" icon={<LogoutOutlined />} onClick={sair} style={{ padding: 0, marginTop: 6, height: 32 }}>
            Sair
          </Button>
        </>
      )}
    </div>
  );

  return (
    <SessaoCtx.Provider value={{ eu, recarregarEu: recarregar }}>
      <AjudaCtx.Provider value={setAjuda}>
        <Layout style={{ minHeight: "100vh", background: "transparent" }}>
          {celular ? (
            <Drawer
              open={gaveta}
              onClose={() => setGaveta(false)}
              placement="left"
              size={280}
              closable={false}
              styles={{ body: { padding: 0 } }}
            >
              <div className="lateral-gaveta">
                {marca}
                {menu}
                {perfil}
              </div>
            </Drawer>
          ) : (
            <Layout.Sider className="lateral" width={248} collapsedWidth={72} collapsed={recolhido} trigger={null}>
              {marca}
              {menu}
              {perfil}
            </Layout.Sider>
          )}

          <Layout style={{ background: "transparent", minWidth: 0 }}>
            <header className="topo">
              {celular && (
                <Button
                  type="text"
                  shape="circle"
                  aria-label="Abrir menu"
                  icon={<MenuOutlined />}
                  onClick={() => setGaveta(true)}
                />
              )}
              <Typography.Text strong style={{ flex: 1 }}>
                {caminho.startsWith("/inscricoes") ? "Minha inscrição" : (atual?.[1] ?? "Lisura")}
              </Typography.Text>
              <Input
                ref={busca}
                className="busca"
                prefix={<SearchOutlined />}
                placeholder="Buscar certame…"
                suffix={<span style={{ color: "var(--ink-600)" }}>Ctrl+K</span>}
                onPressEnter={(e) => {
                  const q = e.currentTarget.value.trim();
                  router.push(`/certames${q ? `?q=${encodeURIComponent(q)}` : ""}`);
                }}
                style={{ maxWidth: 320, height: 40 }}
                aria-label="Buscar certame"
              />
              <Tooltip title="Ajuda desta tela">
                <Button shape="circle" icon={<QuestionOutlined />} aria-label="Ajuda desta tela" onClick={() => setVerAjuda(true)} />
              </Tooltip>
            </header>
            <main className="conteudo" key={rodada}>
              {children}
            </main>
          </Layout>
        </Layout>

        <Modal title={ajuda.titulo} open={verAjuda} onCancel={() => setVerAjuda(false)} footer={null}>
          {ajuda.texto}
        </Modal>
      </AjudaCtx.Provider>
    </SessaoCtx.Provider>
  );
}
