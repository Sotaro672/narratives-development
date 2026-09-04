//frontend\src\lib\solana.ts
import { createClient, autoDiscover } from "@solana/client";

const endpoint = import.meta.env.VITE_SOLANA_RPC_URL;

if (!endpoint) {
  throw new Error("VITE_SOLANA_RPC_URL is not set");
}

const websocketEndpoint = endpoint.replace(/^http/, "ws");

export const solanaClient = createClient({
  endpoint,
  websocketEndpoint,
  walletConnectors: autoDiscover(),
});