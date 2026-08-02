// frontend/amol/src/features/scan-result/infrastructure/scanResultSolanaApi.ts

import type { MallPreviewTransferInfo } from "../../shared/types/scanResult";
import { textOrEmpty } from "../../../components/utils/textOrEmpty";
import {
  isFiniteNumber,
  isRecord,
} from "../../../components/utils/typeGuards";

function resolveSolanaRpcUrl(): string {
  return String(
    import.meta.env.VITE_SOLANA_RPC_URL || "",
  ).trim();
}

async function postSolanaRpc(args: {
  rpcUrl: string;
  method: string;
  params: unknown[];
}): Promise<Record<string, unknown>> {
  const response = await fetch(args.rpcUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: 1,
      method: args.method,
      params: args.params,
    }),
  });

  if (!response.ok) {
    throw new Error(
      `Solana RPC error: http ${response.status}`,
    );
  }

  const decoded = await response.json() as unknown;

  if (
    !isRecord(decoded) ||
    Array.isArray(decoded)
  ) {
    throw new Error(
      "Solana RPC error: invalid response",
    );
  }

  if (
    isRecord(decoded.error) &&
    !Array.isArray(decoded.error)
  ) {
    const code = textOrEmpty(
      decoded.error.code,
    );

    const message =
      textOrEmpty(decoded.error.message) ||
      "unknown";

    throw new Error(
      `Solana RPC error: [${code}] ${message}`,
    );
  }

  return decoded;
}

function extractTransfersFromTransaction(
  tx: Record<string, unknown>,
  mintAddress: string,
): MallPreviewTransferInfo[] {
  const meta =
    isRecord(tx.meta) &&
    !Array.isArray(tx.meta)
      ? tx.meta
      : null;

  if (meta?.err != null) {
    return [];
  }

  const transaction =
    isRecord(tx.transaction) &&
    !Array.isArray(tx.transaction)
      ? tx.transaction
      : null;

  const message =
    transaction &&
    isRecord(transaction.message) &&
    !Array.isArray(transaction.message)
      ? transaction.message
      : null;

  const accountKeysRaw = Array.isArray(
    message?.accountKeys,
  )
    ? message.accountKeys
    : [];

  const accountKeys = accountKeysRaw.map(
    (entry) => {
      if (typeof entry === "string") {
        return entry;
      }

      if (
        isRecord(entry) &&
        !Array.isArray(entry)
      ) {
        return textOrEmpty(entry.pubkey);
      }

      return "";
    },
  );

  const preBalances = Array.isArray(
    meta?.preTokenBalances,
  )
    ? meta.preTokenBalances.filter(
        (
          value,
        ): value is Record<string, unknown> =>
          isRecord(value) &&
          !Array.isArray(value),
      )
    : [];

  const postBalances = Array.isArray(
    meta?.postTokenBalances,
  )
    ? meta.postTokenBalances.filter(
        (
          value,
        ): value is Record<string, unknown> =>
          isRecord(value) &&
          !Array.isArray(value),
      )
    : [];

  const ownerByTokenAccount: Record<
    string,
    string
  > = {};

  const applyOwner = (
    balances: Record<string, unknown>[],
  ) => {
    balances.forEach((row) => {
      if (
        textOrEmpty(row.mint) !==
        mintAddress
      ) {
        return;
      }

      const index =
        typeof row.accountIndex === "number"
          ? row.accountIndex
          : Number(row.accountIndex);

      if (!isFiniteNumber(index)) {
        return;
      }

      const tokenAccount =
        accountKeys[Math.trunc(index)] || "";

      const owner = textOrEmpty(
        row.owner,
      );

      if (tokenAccount && owner) {
        ownerByTokenAccount[tokenAccount] =
          owner;
      }
    });
  };

  applyOwner(postBalances);
  applyOwner(preBalances);

  const transferredAt =
    isFiniteNumber(tx.blockTime)
      ? new Date(
          tx.blockTime * 1000,
        ).toISOString()
      : null;

  const output: MallPreviewTransferInfo[] =
    [];

  const collectFromInstructionList = (
    instructions: unknown[],
  ) => {
    instructions.forEach((raw) => {
      if (
        !isRecord(raw) ||
        Array.isArray(raw)
      ) {
        return;
      }

      const program = textOrEmpty(
        raw.program,
      );

      const programId = textOrEmpty(
        raw.programId,
      );

      if (
        program !== "spl-token" &&
        programId !==
          "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
      ) {
        return;
      }

      const parsed =
        isRecord(raw.parsed) &&
        !Array.isArray(raw.parsed)
          ? raw.parsed
          : null;

      if (!parsed) {
        return;
      }

      const type = textOrEmpty(
        parsed.type,
      );

      if (
        type !== "transfer" &&
        type !== "transferChecked"
      ) {
        return;
      }

      const info =
        isRecord(parsed.info) &&
        !Array.isArray(parsed.info)
          ? parsed.info
          : null;

      if (!info) {
        return;
      }

      const instructionMint =
        textOrEmpty(info.mint) ||
        mintAddress;

      if (
        instructionMint !== mintAddress
      ) {
        return;
      }

      const sourceToken = textOrEmpty(
        info.source,
      );

      const destinationToken = textOrEmpty(
        info.destination,
      );

      if (
        !sourceToken ||
        !destinationToken
      ) {
        return;
      }

      const fromWallet =
        ownerByTokenAccount[sourceToken] ||
        "";

      const toWallet =
        ownerByTokenAccount[
          destinationToken
        ] || "";

      if (!fromWallet || !toWallet) {
        return;
      }

      output.push({
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
    Array.isArray(message?.instructions)
      ? message.instructions
      : [],
  );

  const innerInstructions =
    Array.isArray(meta?.innerInstructions)
      ? meta.innerInstructions
      : [];

  innerInstructions.forEach((inner) => {
    if (
      !isRecord(inner) ||
      Array.isArray(inner)
    ) {
      return;
    }

    collectFromInstructionList(
      Array.isArray(inner.instructions)
        ? inner.instructions
        : [],
    );
  });

  return output;
}

export async function listSolanaTransfersByMintAddress(
  args: {
    mintAddress: string;
    limit?: number;
  },
): Promise<MallPreviewTransferInfo[]> {
  const mintAddress =
    args.mintAddress.trim();

  if (!mintAddress) {
    return [];
  }

  const rpcUrl = resolveSolanaRpcUrl();

  if (!rpcUrl) {
    throw new Error(
      "VITE_SOLANA_RPC_URL is not configured",
    );
  }

  const signaturesJson =
    await postSolanaRpc({
      rpcUrl,
      method:
        "getSignaturesForAddress",
      params: [
        mintAddress,
        {
          limit: args.limit ?? 50,
          commitment: "finalized",
        },
      ],
    });

  const result = signaturesJson.result;

  if (!Array.isArray(result)) {
    return [];
  }

  const signatures = result
    .filter(
      (
        value,
      ): value is Record<string, unknown> =>
        isRecord(value) &&
        !Array.isArray(value),
    )
    .map((row) =>
      textOrEmpty(row.signature),
    )
    .filter(Boolean);

  if (signatures.length === 0) {
    return [];
  }

  const output: MallPreviewTransferInfo[] =
    [];

  const seen = new Set<string>();

  for (const signature of signatures) {
    const transactionJson =
      await postSolanaRpc({
        rpcUrl,
        method: "getTransaction",
        params: [
          signature,
          {
            encoding: "jsonParsed",
            commitment: "finalized",
            maxSupportedTransactionVersion:
              0,
          },
        ],
      });

    const transaction =
      transactionJson.result;

    if (
      !isRecord(transaction) ||
      Array.isArray(transaction)
    ) {
      continue;
    }

    const items =
      extractTransfersFromTransaction(
        transaction,
        mintAddress,
      );

    items.forEach((item) => {
      const key =
        `${item.fromWalletAddress}|` +
        `${item.toWalletAddress}|` +
        `${item.transferredAt || ""}`;

      if (seen.has(key)) {
        return;
      }

      seen.add(key);
      output.push(item);
    });
  }

  return output.sort((a, b) => {
    const firstTime = a.transferredAt
      ? new Date(
          a.transferredAt,
        ).getTime()
      : 0;

    const secondTime = b.transferredAt
      ? new Date(
          b.transferredAt,
        ).getTime()
      : 0;

    return secondTime - firstTime;
  });
}