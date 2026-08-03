// frontend/amol/src/features/scan-result/application/scanOwnedWalletUsecase.ts

export type ScanOwnedWalletUsecaseDeps = {
  getOptionalAuthHeaders: () => Promise<
    HeadersInit | undefined
  >;
  isOwnedByWalletMintAddress: (
    mintAddress: string,
    headers?: HeadersInit,
  ) => Promise<boolean>;
};

export async function resolveScanOwnedWalletState(
  deps: ScanOwnedWalletUsecaseDeps,
  mintAddress: string,
): Promise<boolean | null> {
  const normalizedMintAddress =
    mintAddress.trim();

  if (!normalizedMintAddress) {
    return null;
  }

  const headers =
    await deps.getOptionalAuthHeaders();

  if (!headers) {
    return null;
  }

  return deps.isOwnedByWalletMintAddress(
    normalizedMintAddress,
    headers,
  );
}