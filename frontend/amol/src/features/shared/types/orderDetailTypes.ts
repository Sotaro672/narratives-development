// frontend/amol/src/features/order/types/orderDetailTypes.ts

import type {
  WalletOrderShippingQuoteSnapshot,
} from "../../shared/types/orderTypes";

export type OrderDetailItemType =
  | "list"
  | "resale";

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
  measurements?: Record<
    string,
    number
  >;

  volumeValue?: number;
  volumeUnit?: string;

  qty: number;
  price: number;

  isCanceled: boolean;
  isDispatched: boolean;

  transferred: boolean;
  transferredAt?: string;
};

export type OrderDetail = {
  id: string;
  userId: string;
  avatarId: string;
  cartId: string;

  shippingQuoteSnapshot:
    WalletOrderShippingQuoteSnapshot;

  paid: boolean;

  items: OrderDetailItem[];

  createdAt?: string;
  updatedAt?: string;
};

export type FetchOrderDetailInput = {
  backendUrl: string;
  idToken: string;
  orderId: string;
};