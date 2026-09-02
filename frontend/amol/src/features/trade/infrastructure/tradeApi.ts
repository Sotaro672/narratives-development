// frontend/amol/src/features/trade/infrastructure/tradeApi.ts

import {
  requestJson,
  type ApiQueryParams,
} from "../../../lib/http";

import type {
  CreateTradeMessageParams,
  CreateTradeMessageRequest,
  CreateTradeMessageResponse,
  GetTradeByIDParams,
  GetTradeByOrderItemParams,
  GetTradeUnreadCountParams,
  GetTradeUnreadCountResponse,
  MarkTradeMessagesReadParams,
  MarkTradeMessagesReadResponse,
  TradeDetail,
  TradeDetailResponse,
  TradeMessage,
  TradeStatus,
  TradeViewerSide,
} from "../../shared/types/trade";

const TRADE_BASE_PATH = "/mall/me/trades";

type TradeRequestOptions = {
  signal?: AbortSignal;
};

export type TradeChatListItem = {
  id: string;
  orderId: string;
  orderItemIndex: number;
  viewerSide: TradeViewerSide;
  productName?: string;
  counterpartAvatarId: string;
  counterpartAvatarName?: string;
  counterpartAvatarIcon?: string;
  status: TradeStatus;
  isCancelled: boolean;
  isDispatched: boolean;
  isReturnRequested: boolean;
  isReturnCompleted: boolean;
  transferred: boolean;
  latestMessage?: TradeMessage;
  unreadMessageCount: number;
  latestActivityAt: string;
  createdAt?: string;
  updatedAt?: string;
};

export type TradeChatListResponse = {
  items: TradeChatListItem[];
};

export type CancelTradeOrderItemParams = {
  orderId: string;
  orderItemIndex: number;
};

export type TradeDispatchCarrier = "post" | "yamato";

export type TradeDispatchBoxSize =
  | 60
  | 80
  | 100
  | 120
  | 140
  | 160;

export type DispatchTradeParams = {
  tradeId: string;
  carrier: TradeDispatchCarrier;
  boxSize: TradeDispatchBoxSize;
};

type TradeRequestInit = Omit<RequestInit, "body"> & {
  json?: unknown;
  query?: ApiQueryParams;
};

function requireTradeId(tradeId: string): string {
  const normalizedTradeId = tradeId.trim();

  if (!normalizedTradeId) {
    throw new Error("tradeId is required");
  }

  return normalizedTradeId;
}

function requireOrderId(orderId: string): string {
  const normalizedOrderId = orderId.trim();

  if (!normalizedOrderId) {
    throw new Error("orderId is required");
  }

  return normalizedOrderId;
}

function requireOrderItemIndex(orderItemIndex: number): number {
  if (
    !Number.isInteger(orderItemIndex) ||
    orderItemIndex < 0
  ) {
    throw new Error("orderItemIndex must be a non-negative integer");
  }

  return orderItemIndex;
}

function requireMessageContent(content: string): string {
  if (!content || !/\S/u.test(content)) {
    throw new Error("メッセージを入力してください。");
  }

  return content;
}

function buildMessageQuery(params: {
  limit?: number;
  beforeCreatedAt?: string;
  afterCreatedAt?: string;
}): ApiQueryParams {
  return {
    limit: params.limit,
    beforeCreatedAt: params.beforeCreatedAt,
    afterCreatedAt: params.afterCreatedAt,
  };
}

function buildTradePath(tradeId: string): string {
  return `${TRADE_BASE_PATH}/${encodeURIComponent(
    requireTradeId(tradeId),
  )}`;
}

async function fetchTradeWithAuth<T>(
  path: string,
  init?: TradeRequestInit,
): Promise<T> {
  const { json, query, ...requestInit } = init ?? {};

  return requestJson<T>(path, {
    ...requestInit,
    auth: "required",
    query,
    ...(json !== undefined ? { json } : {}),
    messages: {
      requestErrorMessage: "取引チャットのAPIリクエストに失敗しました。",
    },
  });
}

export async function fetchMyTradeChats(
  options: TradeRequestOptions = {},
): Promise<TradeChatListResponse> {
  return fetchTradeWithAuth<TradeChatListResponse>(
    TRADE_BASE_PATH,
    {
      method: "GET",
      signal: options.signal,
    },
  );
}

export async function fetchTradeByOrderItem(
  params: GetTradeByOrderItemParams,
  options: TradeRequestOptions = {},
): Promise<TradeDetail> {
  const orderId = requireOrderId(params.orderId);
  const orderItemIndex = requireOrderItemIndex(
    params.orderItemIndex,
  );

  const result = await fetchTradeWithAuth<TradeDetailResponse>(
    `${TRADE_BASE_PATH}/order-items/${encodeURIComponent(
      orderId,
    )}/${orderItemIndex}`,
    {
      method: "GET",
      signal: options.signal,
      query: buildMessageQuery(params),
    },
  );

  return result.data;
}

export async function fetchTradeById(
  params: GetTradeByIDParams,
  options: TradeRequestOptions = {},
): Promise<TradeDetail> {
  const result = await fetchTradeWithAuth<TradeDetailResponse>(
    buildTradePath(params.tradeId),
    {
      method: "GET",
      signal: options.signal,
      query: buildMessageQuery(params),
    },
  );

  return result.data;
}

export async function cancelTradeOrderItem(
  params: CancelTradeOrderItemParams,
): Promise<void> {
  const orderId = requireOrderId(params.orderId);
  const orderItemIndex = requireOrderItemIndex(
    params.orderItemIndex,
  );

  await fetchTradeWithAuth<unknown>(
    `/mall/me/orders/${encodeURIComponent(
      orderId,
    )}/items/${orderItemIndex}/cancel`,
    {
      method: "PATCH",
    },
  );
}

export async function dispatchTrade(
  params: DispatchTradeParams,
): Promise<void> {
  await fetchTradeWithAuth<unknown>(
    `${buildTradePath(params.tradeId)}/dispatch`,
    {
      method: "POST",
      json: {
        carrier: params.carrier,
        boxSize: params.boxSize,
      },
    },
  );
}

export async function createTradeMessage(
  params: CreateTradeMessageParams,
): Promise<TradeMessage> {
  const tradeId = requireTradeId(params.tradeId);
  const content = requireMessageContent(params.content);

  const request: CreateTradeMessageRequest = {
    content,
  };

  const result =
    await fetchTradeWithAuth<CreateTradeMessageResponse>(
      `${buildTradePath(tradeId)}/messages`,
      {
        method: "POST",
        json: request,
      },
    );

  return result.data;
}

export async function markTradeMessagesAsRead(
  params: MarkTradeMessagesReadParams,
): Promise<MarkTradeMessagesReadResponse> {
  return fetchTradeWithAuth<MarkTradeMessagesReadResponse>(
    `${buildTradePath(params.tradeId)}/read`,
    {
      method: "POST",
    },
  );
}

export async function getTradeUnreadCount(
  params: GetTradeUnreadCountParams,
  options: TradeRequestOptions = {},
): Promise<GetTradeUnreadCountResponse> {
  return fetchTradeWithAuth<GetTradeUnreadCountResponse>(
    `${buildTradePath(params.tradeId)}/unread-count`,
    {
      method: "GET",
      signal: options.signal,
    },
  );
}