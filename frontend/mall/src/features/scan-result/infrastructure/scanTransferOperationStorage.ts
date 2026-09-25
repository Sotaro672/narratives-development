// frontend/mall/src/features/scan-result/infrastructure/scanTransferOperationStorage.ts

const TRANSFER_OPERATION_STORAGE_PREFIX = "scan-transfer-operation:";

function getTransferOperationStorageKey(productId: string): string {
  return `${TRANSFER_OPERATION_STORAGE_PREFIX}${productId}`;
}

export function readStoredTransferOperationId(productId: string): string {
  if (!productId) {
    return "";
  }

  try {
    return (
      globalThis.sessionStorage
        .getItem(getTransferOperationStorageKey(productId))
        ?.trim() ?? ""
    );
  } catch {
    return "";
  }
}

export function getOrCreateTransferOperationId(productId: string): string {
  const storedOperationId = readStoredTransferOperationId(productId);

  if (storedOperationId) {
    return storedOperationId;
  }

  const operationId = globalThis.crypto.randomUUID();

  try {
    globalThis.sessionStorage.setItem(
      getTransferOperationStorageKey(productId),
      operationId,
    );
  } catch {
    // sessionStorage が利用できない場合も、呼び出し元で operationId を保持して利用する。
  }

  return operationId;
}

export function clearStoredTransferOperationId(
  productId: string,
  operationId: string,
): void {
  if (!productId || !operationId) {
    return;
  }

  try {
    const storageKey = getTransferOperationStorageKey(productId);

    if (globalThis.sessionStorage.getItem(storageKey) === operationId) {
      globalThis.sessionStorage.removeItem(storageKey);
    }
  } catch {
    // sessionStorage が利用できない場合は何もしない。
  }
}