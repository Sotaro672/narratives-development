// frontend/amol/src/features/shared/types/trade.ts

import type {
  ResaleColor,
  ResaleCondition,
  ResaleVolume,
} from "./resale";

export const TRADE_STATUSES = ["active", "closed"] as const;

export type TradeStatus = (typeof TRADE_STATUSES)[number];

export const TRADE_MESSAGE_SENDER_SIDES = [
  "buyer",
  "seller",
  "system",
] as const;

export type TradeMessageSenderSide =
  (typeof TRADE_MESSAGE_SENDER_SIDES)[number];

export type TradeViewerSide = Exclude<
  TradeMessageSenderSide,
  "system"
>;

export const TRADE_MESSAGE_SENDER_TYPES = [
  "avatar",
  "system",
] as const;

export type TradeMessageSenderType =
  (typeof TRADE_MESSAGE_SENDER_TYPES)[number];

export const TRADE_RETURN_REQUEST_KINDS = [
  "unopened",
  "opened",
] as const;

export type TradeReturnRequestKind =
  (typeof TRADE_RETURN_REQUEST_KINDS)[number];

export type TradeResaleImage = {
  id: string;
  url: string;
  displayOrder: number;
};

export type TradeResaleDetail = {
  id: string;
  condition: ResaleCondition;
  description?: string;
  modelId?: string;
  kind?: string;
  modelNumber?: string;
  size?: string;
  color?: ResaleColor;
  measurements?: Record<string, number>;
  volume?: ResaleVolume;
  images: TradeResaleImage[];
};

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
  content?: string;
  images?: TradeMessageImage[];
  buyerReadAt?: string;
  sellerReadAt?: string;
  createdAt: string;
};

export type TradeDetail = {
  id: string;
  orderId: string;
  orderItemIndex: number;
  viewerSide: TradeViewerSide;
  productName?: string;
  resale: TradeResaleDetail;
  buyerAvatarId: string;
  buyerAvatarName?: string;
  buyerAvatarIcon?: string;
  sellerAvatarId: string;
  sellerAvatarName?: string;
  sellerAvatarIcon?: string;
  status: TradeStatus;
  isCancelled: boolean;
  isDispatched: boolean;
  isReturnRequested: boolean;
  returnRequestKind?: TradeReturnRequestKind;
  returnRequestedAt?: string;
  isReturnCompleted: boolean;
  returnCompletedAt?: string;
  transferred: boolean;
  transferredAt?: string;
  messages: TradeMessage[];
  createdAt?: string;
  updatedAt?: string;
  lastMessageAt?: string;
};

export type GetTradeByOrderItemParams = {
  orderId: string;
  orderItemIndex: number;
  limit?: number;
  beforeCreatedAt?: string;
  afterCreatedAt?: string;
};

export type GetTradeByIDParams = {
  tradeId: string;
  limit?: number;
  beforeCreatedAt?: string;
  afterCreatedAt?: string;
};

export type TradeDetailResponse = {
  data: TradeDetail;
};

export type CreateTradeMessageParams = {
  tradeId: string;
  content: string;
};

export type CreateTradeMessageRequest = {
  content: string;
};

export type CreateTradeMessageResponse = {
  data: TradeMessage;
};

export type MarkTradeMessagesReadParams = {
  tradeId: string;
};

export type MarkTradeMessagesReadResponse = {
  success: boolean;
};

export type GetTradeUnreadCountParams = {
  tradeId: string;
};

export type GetTradeUnreadCountResponse = {
  count: number;
};