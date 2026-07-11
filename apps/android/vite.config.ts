import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// @ts-expect-error process is a nodejs global
const host = process.env.TAURI_DEV_HOST;

const disabledMarkstreamOptionalPeers = [
  "@antv/infographic",
  "@terrastruct/d2",
  "mermaid",
  "stream-markdown",
  "stream-monaco",
] as const;
const disabledMarkstreamOptionalPeerId = "\0disabled-markstream-optional-peer";

// https://vite.dev/config/
export default defineConfig(async () => ({
  plugins: [
    react(),
    {
      name: "disable-markstream-optional-peers",
      resolveId(source) {
        if (disabledMarkstreamOptionalPeers.includes(source as (typeof disabledMarkstreamOptionalPeers)[number])) {
          return disabledMarkstreamOptionalPeerId;
        }
        return null;
      },
      load(id) {
        if (id === disabledMarkstreamOptionalPeerId) {
          return "export default undefined;";
        }
        return null;
      },
    },
  ],

  // Vite options tailored for Tauri development and only applied in `tauri dev` or `tauri build`
  //
  // 1. prevent Vite from obscuring rust errors
  clearScreen: false,
  // 2. tauri expects a fixed port, fail if that port is not available
  server: {
    port: 1420,
    strictPort: true,
    host: host || false,
    hmr: host
      ? {
          protocol: "ws",
          host,
          port: 1421,
        }
      : undefined,
    watch: {
      // 3. tell Vite to ignore watching `src-tauri`
      ignored: ["**/src-tauri/**"],
    },
  },
}));
