// frontend/amol/src/features/shared/types/orderTypes.ts

import type { PageResult } from "../pageResult";
import type { ProductCategoryKind } from "./category";

export type WalletOrderItemType = "list" | "resale";

export type WalletOrderShippingSnapshot = {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: string;
};

export type WalletOrderShippingQuoteItemSnapshot = {
  listId: string;
  inventoryId: string;
  modelId: string;
  originShippingAddressId: string;
  destinationShippingAddressId: string;
  carrier: string;
  transportationId: string;
  size: number;
  qty: number;
  unitAmount: number;
  amount: number;
  currency: string;
};

export type WalletOrderShippingQuoteSnapshot = {
  items: WalletOrderShippingQuoteItemSnapshot[];
  amount: number;
  currency: string;
};

export type WalletOrderPaymentMethodSnapshot = {
  customerId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
  isDefault: boolean;
};

export type WalletOrderColor = {
  name?: string;
  hex?: string;
  rgb?: number;
};

export type WalletOrderMeasurements = Record<string, number>;

export type WalletOrderItemSnapshot = {
  itemType?: WalletOrderItemType;
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
  kind?: ProductCategoryKind;
  modelNumber?: string;

  /** apparel 用 */
  size?: string;
  color?: WalletOrderColor;
  measurements?: WalletOrderMeasurements;

  /** alcohol 用 */
  volumeValue?: number;
  volumeUnit?: string;

  tokenName?: string;
  tokenIcon?: string;

  qty: number;
  price: number;
  isCanceled: boolean;
  isDispatched: boolean;
  transferred: boolean;
  transferredAt?: string;
};

export type WalletOrder = {
  id: string;
  userId: string;
  avatarId: string;
  cartId: string;
  shippingSnapshot: WalletOrderShippingSnapshot;
  shippingQuoteSnapshot: WalletOrderShippingQuoteSnapshot;
  paymentMethodSnapshot: WalletOrderPaymentMethodSnapshot;
  paid: boolean;
  items: WalletOrderItemSnapshot[];
  createdAt?: string;
  updatedAt?: string;
};

export type WalletOrdersPage = PageResult<WalletOrder>;

export type FetchWalletOrdersInput = {
  backendUrl: string;
  idToken: string;
  page?: number;
  perPage?: number;
  sort?: string;
  order?: "asc" | "desc";
};