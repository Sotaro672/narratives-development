// frontend/mall/src/features/trade/presentation/util/tradeDispatchQr.ts

export function createTradeDispatchQrPayload(tradeId: string): string {
  const normalizedTradeId = tradeId.trim();

  if (!normalizedTradeId) {
    return "";
  }

  return JSON.stringify({
    type: "amol-pudo-mock",
    version: 1,
    tradeId: normalizedTradeId,
  });
}