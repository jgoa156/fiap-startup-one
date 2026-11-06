import type { StorybookConfig } from "@storybook/nextjs-vite";

// O sistema de design da Lisura no Storybook. `npm run storybook` abre para editar (porta 6006);
// `npm run design` gera o site estático em public/design, que o Next serve em /design.
const config: StorybookConfig = {
  stories: ["../src/**/*.mdx", "../src/**/*.stories.@(ts|tsx)"],
  addons: ["@storybook/addon-docs"],
  framework: { name: "@storybook/nextjs-vite", options: {} },
  core: { disableTelemetry: true },
  docs: { defaultName: "Documentação" },
};

export default config;
