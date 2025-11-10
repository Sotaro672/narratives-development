// frontend/order/src/infrastructure/mockdata/tracking_mockdata.ts
import type { Tracking } from "../../../../shell/src/shared/types/tracking";

/**
 * モック用 Tracking データ
 * frontend/shell/src/shared/types/tracking.ts に準拠。
 *
 * - 各 orderId ごとに配送追跡番号を設定
 * - specialInstructions は任意フィールド
 */
export const TRACKINGS: Tracking[] = [
  {
    id: "track_001",
    orderId: "order_0001",
    trackingNumber: "YMT123456789JP",
    carrier: "Yamato",
    specialInstructions: "配達前にお電話ください。",
    createdAt: "2024-03-21T10:30:00Z",
    updatedAt: "2024-03-21T10:30:00Z",
  },
  {
    id: "track_002",
    orderId: "order_0002",
    trackingNumber: "SGW987654321JP",
    carrier: "Sagawa",
    specialInstructions: null,
    createdAt: "2024-03-21T13:30:00Z",
    updatedAt: "2024-03-21T13:30:00Z",
  },
  {
    id: "track_003",
    orderId: "order_0003",
    trackingNumber: "JP1234567890US",
    carrier: "JapanPost",
    specialInstructions: "玄関前に置き配希望。",
    createdAt: "2024-03-22T09:00:00Z",
    updatedAt: "2024-03-22T09:15:00Z",
  },
  {
    id: "track_004",
    orderId: "order_0004",
    trackingNumber: "FDX987654321US",
    carrier: "FedEx",
    specialInstructions: "法人宛：9時～17時の間に配達可。",
    createdAt: "2024-03-23T08:00:00Z",
    updatedAt: "2024-03-23T08:30:00Z",
  },
  {
    id: "track_005",
    orderId: "order_0005",
    trackingNumber: "DHL543216789EU",
    carrier: "DHL",
    specialInstructions: null,
    createdAt: "2024-03-25T11:00:00Z",
    updatedAt: "2024-03-25T11:00:00Z",
  },
];
