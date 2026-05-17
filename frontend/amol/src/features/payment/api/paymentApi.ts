//frontend\amol\src\features\payment\api\paymentApi.ts
import type {
  CreateOrderRequest,
  CreatePaymentRequest,
  CreatedOrder,
  CreatedPayment,
  PaymentContext,
  PaymentMethod,
  PaymentMethodDefaultResponse,
  PaymentMethodListResponse,
} from "../types";
import {
  API_BASE_URL,
  getAuthHeaders,
  getResponseErrorMessage,
  parseJsonOrNull,
  parseJsonOrThrow,
} from "./paymentHttp";

export async function fetchPaymentContext(): Promise<PaymentContext> {
  const headers = await getAuthHeaders();

  const response = await fetch(`${API_BASE_URL}/mall/me/payments`, {
    method: "GET",
    headers,
    credentials: "include",
  });

  return parseJsonOrThrow<PaymentContext>(response);
}

export async function fetchPaymentMethods(): Promise<{
  methods: PaymentMethod[];
  defaultMethod: PaymentMethod | null;
}> {
  const headers = await getAuthHeaders();

  const [listResponse, defaultResponse] = await Promise.all([
    fetch(`${API_BASE_URL}/mall/me/payment-methods`, {
      method: "GET",
      headers,
      credentials: "include",
    }),
    fetch(`${API_BASE_URL}/mall/me/payment-methods/default`, {
      method: "GET",
      headers,
      credentials: "include",
    }),
  ]);

  const listBody =
    await parseJsonOrNull<PaymentMethodListResponse>(listResponse);

  const defaultBody =
    await parseJsonOrNull<PaymentMethodDefaultResponse>(defaultResponse);

  if (!listResponse.ok) {
    throw new Error(
      getResponseErrorMessage(listBody, "支払い方法の取得に失敗しました。"),
    );
  }

  if (!defaultResponse.ok && defaultResponse.status !== 404) {
    throw new Error(
      getResponseErrorMessage(
        defaultBody,
        "既定の支払い方法の取得に失敗しました。",
      ),
    );
  }

  return {
    methods: Array.isArray(listBody?.data) ? listBody.data : [],
    defaultMethod: defaultBody?.data ?? null,
  };
}

export async function createOrder(
  input: CreateOrderRequest,
): Promise<CreatedOrder> {
  const headers = await getAuthHeaders();

  const response = await fetch(`${API_BASE_URL}/mall/me/orders`, {
    method: "POST",
    headers,
    credentials: "include",
    body: JSON.stringify(input),
  });

  const order = await parseJsonOrThrow<CreatedOrder>(response);

  return {
    ...order,
    id: order.id ?? input.id,
    avatarId: order.avatarId ?? input.avatarId,
    cartId: order.cartId ?? input.cartId,
    paid: order.paid ?? false,
  };
}

export async function createPayment(
  input: CreatePaymentRequest,
): Promise<CreatedPayment> {
  const headers = await getAuthHeaders();

  const response = await fetch(`${API_BASE_URL}/mall/me/payments`, {
    method: "POST",
    headers,
    credentials: "include",
    body: JSON.stringify({
      paymentId: input.paymentId,
      paymentMethodId: input.paymentMethodId,
      stripeCustomerId: input.stripeCustomerId,
      stripePaymentMethodId: input.stripePaymentMethodId,
      amount: input.amount,
    }),
  });

  const data = await parseJsonOrThrow<CreatedPayment>(response);

  return {
    ...data,
    paymentId: data.paymentId ?? data.id ?? input.paymentId,
    paymentMethodId: data.paymentMethodId ?? input.paymentMethodId,
    stripeCustomerId: data.stripeCustomerId ?? input.stripeCustomerId,
    stripePaymentMethodId:
      data.stripePaymentMethodId ?? input.stripePaymentMethodId,
    amount: data.amount ?? input.amount,
  };
}