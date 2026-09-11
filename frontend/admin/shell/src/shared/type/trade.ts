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
  commentCount: number;
  reportCount: number;
  createdAt: string;
  updatedAt: string;
  lastMessageAt?: string | null;
};

export type TradeMessageSenderSide = "buyer" | "seller" | "system";

export type TradeMessageSenderType = "avatar" | "system";

export type TradeMessageImage = {
  fileName: string;
  fileUrl: string;
  objectPath: string;
  fileSize: number;
  mimeType: string;
};

export type TradeMessage = {
  id: string;
  tradeId: string;
  senderSide: TradeMessageSenderSide;
  senderType: TradeMessageSenderType;
  senderId: string;
  senderName: string;
  content: string;
  images: TradeMessageImage[];
  reportCount: number;
  reportCaseId?: string;
  createdAt: string;
};

export type ResaleTradeListResponse = {
  items: Trade[];
  totalCount: number;
};

export type TradeMessageListResponse = {
  items: TradeMessage[];
  totalCount: number;
};