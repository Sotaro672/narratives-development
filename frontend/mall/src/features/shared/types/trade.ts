// frontend/mall/src/features/shared/types/trade.ts

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

// ============================================================
// Trade Return
// ============================================================

export const TRADE_RETURN_CONSULTATION_REASONS = [
  "not_as_described",
  "damaged",
  "wrong_item",
  "other",
] as const;

export type TradeReturnConsultationReason =
  (typeof TRADE_RETURN_CONSULTATION_REASONS)[number];

export const TRADE_RETURN_STATUSES = [
  "none",
  "discussing",
  "proposed",
  "agreed",
  "return_shipped",
  "return_received",
  "refund_processing",
  "completed",
  "disputed",
] as const;

export type TradeReturnStatus =
  (typeof TRADE_RETURN_STATUSES)[number];

export const TRADE_RETURN_AGREEMENTS = [
  "agree",
  "disagree",
] as const;

export type TradeReturnAgreement =
  (typeof TRADE_RETURN_AGREEMENTS)[number];

export const TRADE_RETURN_REQUIREMENTS = [
  "required",
  "not_required",
] as const;

export type TradeReturnRequirement =
  (typeof TRADE_RETURN_REQUIREMENTS)[number];

export type TradeReturnConsultation = {
  id: string;
  reason: TradeReturnConsultationReason;
  detail: string;
  createdAt: string;
};

export type TradeReturnProposal = {
  id: string;
  agreement: TradeReturnAgreement;
  returnRequirement?: TradeReturnRequirement;
  refundAmount?: number;
  createdAt: string;
  rejectedAt?: string;
};

export type CreateTradeReturnConsultationParams = {
  tradeId: string;
  reason: TradeReturnConsultationReason;
  detail: string;
};

export type CreateTradeReturnConsultationRequest = {
  reason: TradeReturnConsultationReason;
  detail: string;
};

export type CreateTradeReturnConsultationResponse = {
  data: TradeReturnConsultation;
};

export type CreateTradeReturnProposalParams = {
  tradeId: string;
  agreement: TradeReturnAgreement;
  returnRequirement?: TradeReturnRequirement;
  refundAmount?: number;
};

export type CreateTradeReturnProposalRequest = {
  agreement: TradeReturnAgreement;
  returnRequirement?: TradeReturnRequirement;
  refundAmount?: number;
};

export type CreateTradeReturnProposalResponse = {
  data: TradeReturnProposal;
};

export type AcceptTradeReturnProposalParams = {
  tradeId: string;
  proposalId: string;
};

export type RejectTradeReturnProposalParams = {
  tradeId: string;
  proposalId: string;
};

// ============================================================
// Trade Return Shipment
// ============================================================

export const TRADE_RETURN_SHIPMENT_STATUSES = [
  "pending",
  "ready_for_dropoff",
] as const;

export type TradeReturnShipmentStatus =
  (typeof TRADE_RETURN_SHIPMENT_STATUSES)[number];

export const TRADE_RETURN_SHIPMENT_DROP_OFF_METHODS = [
  "pudo",
] as const;

export type TradeReturnShipmentDropOffMethod =
  (typeof TRADE_RETURN_SHIPMENT_DROP_OFF_METHODS)[number];

export type TradeReturnShipment = {
  status: TradeReturnShipmentStatus;
  dropOffMethod: TradeReturnShipmentDropOffMethod;
  qrCodePayload?: string;
  readyAt?: string;
};

export type CreateTradeReturnShipmentParams = {
  tradeId: string;
};

export type GetTradeReturnShipmentParams = {
  tradeId: string;
};

export type TradeReturnShipmentResponse = {
  data: TradeReturnShipment;
};

// ============================================================
// Trade Return Receipt
// ============================================================

export type ReceiveTradeReturnParams = {
  tradeId: string;
};

export type ReceiveTradeReturnResult = {
  financiallyCompleted: boolean;
  returnCompleted: boolean;
  notificationEnsured: boolean;
  alreadyCompleted: boolean;
};

export type ReceiveTradeReturnResponse = {
  data: ReceiveTradeReturnResult;
};

// ============================================================
// Resale
// ============================================================

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

// ============================================================
// Message
// ============================================================

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

// ============================================================
// Trade Detail
// ============================================================

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
  returnStatus: TradeReturnStatus;
  returnConsultation?: TradeReturnConsultation;
  returnProposal?: TradeReturnProposal;
  merchandiseRefundMaxAmount: number;
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

// ============================================================
// Trade Message
// ============================================================

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

// ============================================================
// Read State
// ============================================================

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