// frontend\amol\src\features\shared\types\orderDetailTypes.ts

import type { WalletOrderShippingQuoteSnapshot } from "../../shared/types/orderTypes";

export type OrderDetailItemType = "list" | "resale";

export type OrderRefundStatus =
  | "none"
  | "pending"
  | "requires_action"
  | "succeeded"
  | "failed"
  | "canceled";

export type OrderDetailColor = {
  name?: string;
  rgb?: number;
};

export type OrderDetailItem = {
  itemType?: OrderDetailItemType;
  modelId?: string;
  inventoryId?: string;
  listId?: string;
  resaleId?: string;
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  productName?: string;
  brandId?: string;
  brandName?: string;
  brandIcon?: string;
  tokenName?: string;
  tokenIcon?: string;
  kind?: string;
  modelNumber?: string;
  size?: string;
  color?: OrderDetailColor;
  measurements?: Record<string, number>;
  volumeValue?: number;
  volumeUnit?: string;

  qty: number;
  price: number;

  isCancelled: boolean;
  isDispatched: boolean;

  isReturnRequested: boolean;
  returnRequestedAt?: string;
  tokenTransferVerifiedAt?: string;

  transferred: boolean;
  transferredAt?: string;
};

export type OrderDetail = {
  id: string;
  userId: string;
  avatarId: string;
  cartId: string;
  shippingQuoteSnapshot: WalletOrderShippingQuoteSnapshot;
  paid: boolean;
  refundStatus: OrderRefundStatus;
  refundedAmount: number;
  refundedAt?: string;
  items: OrderDetailItem[];
  createdAt?: string;
  updatedAt?: string;
};

export type FetchOrderDetailInput = {
  backendUrl: string;
  idToken: string;
  orderId: string;
};