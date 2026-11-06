import type { ThemeConfig } from "antd";

// Tema do Ant Design: a estrutura é a do protótipo do Inclusa (mockups/antd-heroui-sidenav), com
// os raios e sombras do HeroUI; as cores são as da paleta 5 · Grafite (src/tema/tokens.css).
export const cor = {
  acao: "#2F3B4C",
  titulo: "#1F2833",
  secundaria: "#445166",
  selecao: "#E9EDF2",
  superficie: "#F5F7FA",
  marcador: "#FF7A1A",
  avisoFundo: "#FFF1E6",
  avisoTexto: "#8A3B00",
  tinta: "#14233A",
  tinta2: "#516178",
  divisor: "#E6EBF2",
  fundo: "#F4F5F8",
  sucesso: "#1E7B4F",
  atencao: "#B45309",
  erro: "#B42318",
} as const;

export const tema: ThemeConfig = {
  token: {
    colorPrimary: cor.acao,
    colorLink: cor.acao,
    colorInfo: cor.acao,
    // Derivados destas cores escuras, os fundos dos alertas saem acinzentados e escuros (texto com
    // pouco contraste); fixa os fundos claros.
    colorInfoBg: cor.selecao,
    colorInfoBorder: "#CBD3DE",
    colorSuccessBg: "#E8F5EE",
    colorSuccessBorder: "#B7DEC8",
    colorWarningBg: "#FEF3E2",
    colorWarningBorder: "#F5C98E",
    colorErrorBg: "#FDECEA",
    colorErrorBorder: "#F4B8B2",
    colorSuccess: cor.sucesso,
    colorWarning: cor.atencao,
    colorError: cor.erro,
    colorText: cor.tinta,
    colorTextSecondary: cor.tinta2,
    colorBorderSecondary: cor.divisor,
    colorBgLayout: cor.fundo,
    fontFamily: "var(--fonte-nunito), Nunito, system-ui, sans-serif",
    fontSize: 14,
    borderRadius: 12, // HeroUI médio
    borderRadiusLG: 14, // HeroUI grande
    borderRadiusSM: 8, // HeroUI pequeno
    controlHeight: 44, // alvo mínimo de toque (Inclusa, doc 01 §6)
    boxShadow: "var(--shadow-md)",
    boxShadowSecondary: "var(--shadow-lg)",
    boxShadowTertiary: "var(--shadow-sm)",
  },
  components: {
    Menu: {
      itemHeight: 44,
      itemBorderRadius: 12,
      itemMarginInline: 12,
      itemSelectedBg: cor.selecao,
      itemSelectedColor: cor.acao,
      itemHoverBg: cor.superficie,
      itemColor: cor.tinta2,
      iconSize: 18,
      collapsedIconSize: 18,
      collapsedWidth: 72,
    },
    Button: { borderRadius: 999, fontWeight: 700 },
    Input: { borderRadius: 999 },
    Card: { borderRadiusLG: 14 },
    Tooltip: { borderRadius: 8 },
    Segmented: { itemSelectedBg: "#FFFFFF", trackBg: cor.selecao, itemSelectedColor: cor.acao },
    Tag: { defaultBg: cor.selecao, defaultColor: cor.acao },
  },
};
