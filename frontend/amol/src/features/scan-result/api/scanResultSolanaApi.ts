// frontend/amol/src/features/scan-result/api/scanResultSolanaApi.ts
import type { MallPreviewTransferInfo } from "../types";
import { isRecord, trimText } from "../utils/format";

function resolveSolanaRpcUrl(): string {
  return String(import.meta.env.VITE_SOLANA_RPC_URL || "").trim();
}

async function postSolanaRpc(args: {
  rpcUrl: string;
  method: string;
  params: unknown[];
}): Promise<Record<string, unknown>> {
  const response = await fetch(args.rpcUrl, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: 1,
      method: args.method,
      params: args.params,
    }),
  });

  if (!response.ok) {
    throw new Error(`Solana RPC error: http ${response.status}`);
  }

  const decoded = (await response.json()) as unknown;

  if (!isRecord(decoded)) {
    throw new Error("Solana RPC error: invalid response");
  }

  if (isRecord(decoded.error)) {
    const code = trimText(decoded.error.code);
    const message = trimText(decoded.error.message) || "unknown";
    throw new Error(`Solana RPC error: [${code}] ${message}`);
  }

  return decoded;
}

function extractTransfersFromTransaction(
  tx: Record<string, unknown>,
  mintAddress: string
): MallPreviewTransferInfo[] {
  const meta = isRecord(tx.meta) ? tx.meta : null;
  if (meta?.err != null) return [];

  const transaction = isRecord(tx.transaction) ? tx.transaction : null;
  const message =
    transaction && isRecord(transaction.message) ? transaction.message : null;

  const accountKeysRaw = Array.isArray(message?.accountKeys)
    ? message.accountKeys
    : [];

  const accountKeys = accountKeysRaw.map((entry) => {
    if (typeof entry === "string") return entry;
    if (isRecord(entry)) return trimText(entry.pubkey);
    return "";
  });

  const preBalances = Array.isArray(meta?.preTokenBalances)
    ? meta.preTokenBalances.filter(isRecord)
    : [];

  const postBalances = Array.isArray(meta?.postTokenBalances)
    ? meta.postTokenBalances.filter(isRecord)
    : [];

  const ownerByTokenAccount: Record<string, string> = {};

  const applyOwner = (balances: Record<string, unknown>[]) => {
    balances.forEach((row) => {
      if (trimText(row.mint) !== mintAddress) return;

      const index =
        typeof row.accountIndex === "number"
          ? row.accountIndex
          : Number(row.accountIndex);

      if (!Number.isFinite(index)) return;

      const tokenAccount = accountKeys[Math.trunc(index)] || "";
      const owner = trimText(row.owner);

      if (tokenAccount && owner) {
        ownerByTokenAccount[tokenAccount] = owner;
      }
    });
  };

  applyOwner(postBalances);
  applyOwner(preBalances);

  const transferredAt =
    typeof tx.blockTime === "number"
      ? new Date(tx.blockTime * 1000).toISOString()
      : null;

  const out: MallPreviewTransferInfo[] = [];

  const collectFromInstructionList = (instructions: unknown[]) => {
    instructions.forEach((raw) => {
      if (!isRecord(raw)) return;

      const program = trimText(raw.program);
      const programId = trimText(raw.programId);

      if (
        program !== "spl-token" &&
        programId !== "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
      ) {
        return;
      }

      const parsed = isRecord(raw.parsed) ? raw.parsed : null;
      if (!parsed) return;

      const type = trimText(parsed.type);
      if (type !== "transfer" && type !== "transferChecked") return;

      const info = isRecord(parsed.info) ? parsed.info : null;
      if (!info) return;

      const ixMint = trimText(info.mint) || mintAddress;
      if (ixMint !== mintAddress) return;

      const sourceToken = trimText(info.source);
      const destinationToken = trimText(info.destination);
      if (!sourceToken || !destinationToken) return;

      const fromWallet = ownerByTokenAccount[sourceToken] || "";
      const toWallet = ownerByTokenAccount[destinationToken] || "";
      if (!fromWallet || !toWallet) return;

      out.push({
        transferredAt,
        fromWalletAddress: fromWallet,
        toWalletAddress: toWallet,

        fromAvatarId: "",
        fromAvatarName: "",
        fromAvatarIcon: "",
        fromBrandId: "",
        fromBrandName: "",
        fromBrandIcon: "",

        toAvatarId: "",
        toAvatarName: "",
        toAvatarIcon: "",
        toBrandId: "",
        toBrandName: "",
        toBrandIcon: "",
      });
    });
  };

  collectFromInstructionList(
    Array.isArray(message?.instructions) ? message.instructions : []
  );

  const innerInstructions = Array.isArray(meta?.innerInstructions)
    ? meta.innerInstructions
    : [];

  innerInstructions.forEach((inner) => {
    if (!isRecord(inner)) return;

    collectFromInstructionList(
      Array.isArray(inner.instructions) ? inner.instructions : []
    );
  });

  return out;
}

export async function listSolanaTransfersByMintAddress(args: {
  mintAddress: string;
  limit?: number;
}): Promise<MallPreviewTransferInfo[]> {
  const mintAddress = args.mintAddress.trim();
  if (!mintAddress) return [];

  const rpcUrl = resolveSolanaRpcUrl();
  if (!rpcUrl) throw new Error("VITE_SOLANA_RPC_URL is not configured");

  const signaturesJson = await postSolanaRpc({
    rpcUrl,
    method: "getSignaturesForAddress",
    params: [mintAddress, { limit: args.limit ?? 50, commitment: "finalized" }],
  });

  const result = signaturesJson.result;
  if (!Array.isArray(result)) return [];

  const signatures = result
    .filter(isRecord)
    .map((row) => trimText(row.signature))
    .filter(Boolean);

  if (signatures.length === 0) return [];

  const out: MallPreviewTransferInfo[] = [];
  const seen = new Set<string>();

  for (const signature of signatures) {
    const txJson = await postSolanaRpc({
      rpcUrl,
      method: "getTransaction",
      params: [
        signature,
        {
          encoding: "jsonParsed",
          commitment: "finalized",
          maxSupportedTransactionVersion: 0,
        },
      ],
    });

    const tx = txJson.result;
    if (!isRecord(tx)) continue;

    const items = extractTransfersFromTransaction(tx, mintAddress);

    items.forEach((item) => {
      const key = `${item.fromWalletAddress}|${item.toWalletAddress}|${
        item.transferredAt || ""
      }`;

      if (seen.has(key)) return;

      seen.add(key);
      out.push(item);
    });
  }

  return out.sort((a, b) => {
    const at = a.transferredAt ? new Date(a.transferredAt).getTime() : 0;
    const bt = b.transferredAt ? new Date(b.transferredAt).getTime() : 0;
    return bt - at;
  });
}