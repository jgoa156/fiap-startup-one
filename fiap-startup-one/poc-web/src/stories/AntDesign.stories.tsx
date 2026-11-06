import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { Alert, Button, Card, Descriptions, Input, InputNumber, Segmented, Select, Space, Steps, Table, Tag, Timeline } from "antd";
import { BarcodeOutlined, ExportOutlined, QrcodeOutlined } from "@ant-design/icons";

// Os componentes do Ant Design como o tema da Lisura os veste: o que uma tela nova usa antes de
// criar um componente próprio.

const meta: Meta = { title: "Ant Design com o tema", parameters: { layout: "padded" } };
export default meta;

export const Botoes: StoryObj = {
  name: "Botões",
  render: () => (
    <Space wrap>
      <Button type="primary" icon={<QrcodeOutlined />}>Pagar com Pix</Button>
      <Button icon={<BarcodeOutlined />}>Gerar boleto</Button>
      <Button type="primary" loading>Enviando</Button>
      <Button type="primary" disabled>Prazo encerrado</Button>
      <Button type="link">Ver detalhes</Button>
      <Button danger>Parar</Button>
      <Button icon={<ExportOutlined />} iconPlacement="end">Abrir cartão</Button>
    </Space>
  ),
};

export const Campos: StoryObj = {
  render: () => (
    <Space orientation="vertical" style={{ width: 360 }}>
      <Input placeholder="E-mail" aria-label="E-mail" />
      <Input status="error" defaultValue="abc" aria-label="Chave Pix" />
      <InputNumber aria-label="Multiplicador" defaultValue={40} suffix="×" />
      <Select aria-label="UF" style={{ width: 140 }} defaultValue="SP" options={["SP", "RJ", "MG"].map((u) => ({ value: u, label: u }))} />
      <Space.Compact style={{ width: "100%" }}>
        <Input aria-label="Chave Pix" placeholder="E-mail, celular ou chave aleatória" />
        <Button type="primary">Enviar chave</Button>
      </Space.Compact>
    </Space>
  ),
};

export const Alertas: StoryObj = {
  render: () => (
    <Space orientation="vertical" style={{ width: 560 }}>
      <Alert type="success" showIcon title="Pagamento de R$ 80,00 por Pix confirmado." />
      <Alert type="info" showIcon title="Histórico temporariamente indisponível." description="Nada foi perdido: ele volta sozinho." />
      <Alert type="warning" showIcon title="Um aviso não chegou ao seu e-mail." description="Confira o endereço da sua conta." />
      <Alert type="error" showIcon title="Esta inscrição foi cancelada." description="Se você pagou a taxa, o valor volta." />
    </Space>
  ),
};

export const TagsEPassos: StoryObj = {
  name: "Tags, passos e linha do tempo",
  render: () => (
    <Space orientation="vertical" size={20} style={{ width: 640 }}>
      <Space wrap>
        <Tag>Em andamento</Tag><Tag color="success">Devolvido</Tag><Tag color="warning">Falta sua chave Pix</Tag><Tag color="error">Voltou</Tag>
      </Space>
      <Steps current={2} items={["Rascunho", "Enviada", "Paga", "Em análise", "Homologada"].map((title) => ({ title }))} />
      <Timeline items={[
        { content: <><strong>Inscrição enviada</strong><div style={{ fontSize: 12 }}>06/11/2026, 16:59</div></> },
        { content: <><strong>Pagamento confirmado</strong><div style={{ fontSize: 12 }}>06/11/2026, 17:00</div></> },
      ]} />
      <Segmented options={["Normal", "Sala de espera", "Só o essencial", "Só consulta"]} />
    </Space>
  ),
};

export const CartaoETabela: StoryObj = {
  name: "Cartão, descrição e tabela",
  render: () => (
    <Space orientation="vertical" size={16} style={{ width: 640 }}>
      <Card title="Sua inscrição">
        <Descriptions column={1} items={[
          { key: "c", label: "Cargo", children: "Agente Administrativo" },
          { key: "d", label: "Cidade de prova", children: "Campinas" },
        ]} />
      </Card>
      <Table size="small" pagination={false} rowKey="r" dataSource={[{ r: "4005", m: "40×", e: 25554, p: 23077 }]} columns={[
        { title: "Rodada", dataIndex: "r" }, { title: "Patamar", dataIndex: "m" },
        { title: "Enviadas", dataIndex: "e", align: "right" }, { title: "Pagas", dataIndex: "p", align: "right" },
      ]} />
    </Space>
  ),
};
