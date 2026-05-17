// frontend/amol/src/features/wallet/types/orderTypes.ts

export type WalletOrderColor = {
  name?: string;
  hex?: string;
  rgb?: number;
};

export type WalletOrderMeasurements = Record<string, number>;

export type WalletOrderItemKind = "apparel" | "alcohol" | "unknown" | string;

export type WalletOrderItemSnapshot = {
  modelId: string;
  inventoryId: string;
  listId: string;

  productBlueprintId?: string;
  tokenBlueprintId?: string;

  productName?: string;

  brandId?: string;
  brandName?: string;
  brandIcon?: string;

  kind?: WalletOrderItemKind;
  modelNumber?: string;

  /**
   * apparel 用
   */
  size?: string;
  color?: WalletOrderColor;
  measurements?: WalletOrderMeasurements;

  /**
   * alcohol 用
   */
  volumeValue?: number;
  volumeUnit?: string;

  tokenName?: string;
  tokenIcon?: string;

  qty: number;
  price: number;

  isCanceled: boolean;
  isDispatched: boolean;

  transferred?: boolean;
  transferredAt?: string;
};

export type WalletOrder = {
  id: string;
  userId: string;
  avatarId: string;
  cartId: string;

  paid?: boolean;

  items: WalletOrderItemSnapshot[];

  createdAt?: string;
  updatedAt?: string;
};

export type WalletOrdersPage = {
  items: WalletOrder[];
  totalCount?: number;
  totalPages?: number;
  page?: number;
  perPage?: number;
};

export type FetchWalletOrdersInput = {
  backendUrl: string;
  idToken: string;
  page?: number;
  perPage?: number;
  sort?: string;
  order?: "asc" | "desc";
};