// frontend/amol/src/features/wallet/types/orderTypes.ts

export type WalletOrderColor = {
  name?: string;
  hex?: string;
};

export type WalletOrderMeasurements = Record<string, number>;

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

  size?: string;
  color?: WalletOrderColor;
  modelNumber?: string;
  measurements?: WalletOrderMeasurements;

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