// frontend/shell/src/shared/types/tracking.ts
// Mirrors frontend/order/src/domain/entity/tracking.ts
// and backend/internal/domain/tracking/entity.go

/**
 * Tracking
 * - Order ごとに付与される配送追跡情報
 * - specialInstructions は任意（null または undefined）
 * - createdAt / updatedAt は ISO8601 文字列（UTC）
 */
export interface Tracking {
  /** 一意のトラッキングID */
  id: string;

  /** 紐づく注文ID */
  orderId: string;

  /** 追跡番号（英数字・ハイフン・アンダースコア・ドットを許容） */
  trackingNumber: string;

  /** 配送キャリア（例: "Yamato", "Sagawa", "JapanPost", "FedEx" など） */
  carrier: string;

  /** 特記事項（任意） */
  specialInstructions?: string | null;

  /** 作成日時（ISO8601 UTC） */
  createdAt: string;

  /** 更新日時（ISO8601 UTC） */
  updatedAt: string;
}

/**
 * UI / バリデーション補助: キャリア名の表示マップ
 */
export const CARRIER_LABELS: Record<string, string> = {
  Yamato: "ヤマト運輸",
  Sagawa: "佐川急便",
  JapanPost: "日本郵便",
  FedEx: "FedEx",
  DHL: "DHL",
};

/**
 * ヘルパー: キャリアコード → 表示名
 */
export function getCarrierLabel(carrier: string): string {
  return CARRIER_LABELS[carrier] ?? carrier;
}
