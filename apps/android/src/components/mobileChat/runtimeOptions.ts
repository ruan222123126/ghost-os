import type { ConfigPayload, StatusMessage } from "../../mobileTypes";
import type { RuntimeOption } from "./types";

export function runtimeOptions(props: {
  runtimeLabel: string;
  status: StatusMessage;
  config: ConfigPayload | undefined;
}): RuntimeOption[] {
  const provider = props.config?.provider || props.config?.provider_type || "Bridge";
  const model = props.config?.model || "Runtime";

  return [
    {
      id: "bridge-runtime",
      name: "Bridge Runtime",
      desc: props.status.text,
    },
    {
      id: "agent-session",
      name: "Agent Session",
      desc: `${provider} / ${model}`,
    },
    {
      id: "native-driver",
      name: "Native Driver",
      desc: "桌面侧原子执行",
    },
  ];
}
