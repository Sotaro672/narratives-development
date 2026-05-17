// frontend/amol/src/features/scan-result/application/scanTransferUsecase.ts
import type { MallScanTransferResponse } from "../types";

export type ScanWalletSnapshot = {
  tokens?: string[] | null;
};

export type ScanTransferUsecaseDeps = {
  fetchMeWallet: (headers?: HeadersInit) => Promise<ScanWalletSnapshot>;
  transferScanPurchased: (input: {
    productId: string;
    headers?: HeadersInit;
  }) => Promise<MallScanTransferResponse>;
};

export type RunScanAutoTransferInput = {
  productId: string;
  headers?: HeadersInit;
};

export type RunScanAutoTransferResult = {
  transferResult: MallScanTransferResponse;
  transferredMintAddress: string;
};

function extractNonEmptyTokens(tokens: string[] | null | undefined): Set<string> {
  const out = new Set<string>();

  (tokens ?? []).forEach((token) => {
    const s = token.trim();
    if (s) {
      out.add(s);
    }
  });

  return out;
}

function getDifference(after: Set<string>, before: Set<string>): string[] {
  return [...after].filter((value) => !before.has(value));
}

export async function runScanAutoTransfer(
  deps: ScanTransferUsecaseDeps,
  input: RunScanAutoTransferInput,
): Promise<RunScanAutoTransferResult> {
  const productId = input.productId.trim();

  if (!productId) {
    throw new Error("productId is empty");
  }

  let beforeTokens = new Set<string>();

  try {
    const beforeWallet = await deps.fetchMeWallet(input.headers);
    beforeTokens = extractNonEmptyTokens(beforeWallet.tokens);
  } catch {
    beforeTokens = new Set<string>();
  }

  const transferResult = await deps.transferScanPurchased({
    productId,
    headers: input.headers,
  });

  const directMintAddress = transferResult.mintAddress.trim();

  if (directMintAddress) {
    return {
      transferResult,
      transferredMintAddress: directMintAddress,
    };
  }

  try {
    const afterWallet = await deps.fetchMeWallet(input.headers);
    const afterTokens = extractNonEmptyTokens(afterWallet.tokens);
    const addedTokens = getDifference(afterTokens, beforeTokens);

    return {
      transferResult,
      transferredMintAddress: addedTokens[0] ?? "",
    };
  } catch {
    return {
      transferResult,
      transferredMintAddress: "",
    };
  }
}