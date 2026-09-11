// frontend/admin/shell/src/shared/type/trade.ts

export type TradeStatus = "active" | "closed";

export type TradeSellerType = "company" | "avatar";

export type Trade = {
  id: string;
  orderId: string;
  orderItemIndex: number;
  buyerAvatarId: string;
  buyerAvatarName: string;
  sellerType: TradeSellerType;
  sellerCompanyId?: string;
  sellerBrandId?: string;
  sellerAvatarId?: string;
  status: TradeStatus;
  createdAt: string;
  updatedAt: string;
  lastMessageAt?: string | null;
};

export type ResaleTradeListResponse = {
  items: Trade[];
  totalCount: number;
};