// frontend/amol/src/features/scan-result/application/scanTokenResolveUsecase.ts

import type {
  TokenResolveDTO,
} from "../../shared/types/scanResult";

export type ScanTokenResolveUsecaseDeps = {
  getOptionalAuthHeaders: () => Promise<
    HeadersInit | undefined
  >;
  resolveTokenByMintAddress: (
    mintAddress: string,
    headers?: HeadersInit,
  ) => Promise<TokenResolveDTO>;
  wait: (
    ms: number,
  ) => Promise<void>;
};

function hasPreviewableFile(
  resolved: TokenResolveDTO,
): boolean {
  return resolved.tokenContentsFiles.some(
    (file) => {
      return (
        file.isPreviewable &&
        Boolean(file.viewUri.trim())
      );
    },
  );
}

export async function resolveTransferredTokenWithRetry(
  deps: ScanTokenResolveUsecaseDeps,
  input: {
    mintAddress: string;
    maxAttempts?: number;
    intervalMs?: number;
  },
): Promise<TokenResolveDTO> {
  const mintAddress =
    input.mintAddress.trim();

  if (!mintAddress) {
    throw new Error(
      "transferred mintAddress is empty",
    );
  }

  const maxAttempts =
    input.maxAttempts ??
    6;

  const intervalMs =
    input.intervalMs ??
    700;

  const headers =
    await deps.getOptionalAuthHeaders();

  let lastError: unknown =
    null;

  for (
    let attempt = 0;
    attempt < maxAttempts;
    attempt += 1
  ) {
    try {
      const resolved =
        await deps.resolveTokenByMintAddress(
          mintAddress,
          headers,
        );

      if (
        hasPreviewableFile(
          resolved,
        )
      ) {
        return resolved;
      }

      lastError =
        new Error(
          "resolved token has no signed contents",
        );
    } catch (error) {
      lastError =
        error;
    }

    if (
      attempt <
      maxAttempts - 1
    ) {
      await deps.wait(
        intervalMs,
      );
    }
  }

  throw lastError instanceof Error
    ? lastError
    : new Error(
        "resolve token failed",
      );
}