import { Tag } from "antd";
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  MinusCircleOutlined,
} from "@ant-design/icons";
import type { Tom } from "@/lib/rotulos";

// Selo de situação: sempre ícone + texto — a cor nunca é o único sinal (Inclusa, doc 01 §6).
const icone = {
  ok: <CheckCircleOutlined />,
  aviso: <ClockCircleOutlined />,
  erro: <CloseCircleOutlined />,
  neutro: <MinusCircleOutlined />,
};

export function Selo({ tom, children }: { tom: Tom; children: React.ReactNode }) {
  const props =
    tom === "aviso"
      ? { className: "selo-aviso" }
      : tom === "ok"
        ? { color: "success" }
        : tom === "erro"
          ? { color: "error" }
          : {};
  return (
    <Tag {...props} icon={icone[tom]} style={{ borderRadius: 999, paddingInline: 10 }}>
      {children}
    </Tag>
  );
}
