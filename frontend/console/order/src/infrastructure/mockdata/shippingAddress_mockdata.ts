// frontend/order/src/infrastructure/mockdata/shippingAddress_mockdata.ts
// Mock data for frontend/shell/src/shared/types/shippingAddress.ts

import type { ShippingAddress } from "../../../../shell/src/shared/types/shippingAddress";

/**
 * モック用 ShippingAddress データ
 * backend/internal/domain/shippingAddress/entity.go に準拠。
 */
export const SHIPPING_ADDRESSES: ShippingAddress[] = [
  {
    id: "ship_001",
    userId: "user_001",
    street: "東京都渋谷区神南1-10-1",
    city: "渋谷区",
    state: "東京都",
    zipCode: "150-0041",
    country: "Japan",
    createdAt: "2024-03-01T10:00:00Z",
    updatedAt: "2024-03-05T15:30:00Z",
  },
  {
    id: "ship_002",
    userId: "user_002",
    street: "大阪府大阪市北区梅田2-4-9",
    city: "大阪市",
    state: "大阪府",
    zipCode: "530-0001",
    country: "Japan",
    createdAt: "2024-03-02T12:00:00Z",
    updatedAt: "2024-03-06T09:45:00Z",
  },
  {
    id: "ship_003",
    userId: "user_003",
    street: "福岡県福岡市中央区天神1-4-2",
    city: "福岡市",
    state: "福岡県",
    zipCode: "810-0001",
    country: "Japan",
    createdAt: "2024-03-03T08:30:00Z",
    updatedAt: "2024-03-07T14:20:00Z",
  },
  {
    id: "ship_004",
    userId: "user_004",
    street: "北海道札幌市中央区北5条西2丁目",
    city: "札幌市",
    state: "北海道",
    zipCode: "060-0005",
    country: "Japan",
    createdAt: "2024-03-04T11:00:00Z",
    updatedAt: "2024-03-08T16:10:00Z",
  },
  {
    id: "ship_005",
    userId: "user_005",
    street: "愛知県名古屋市中村区名駅3-16-1",
    city: "名古屋市",
    state: "愛知県",
    zipCode: "450-0002",
    country: "Japan",
    createdAt: "2024-03-05T09:00:00Z",
    updatedAt: "2024-03-09T17:45:00Z",
  },
];
