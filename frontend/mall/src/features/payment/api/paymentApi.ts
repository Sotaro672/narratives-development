// frontend/amol/src/features/payment/api/paymentApi.ts

import { requestJson } from "../../../lib/http";

import type {
  CreateOrderRequest,
  CreatedOrder,
  PaymentContext,
} from "../../shared/types/payment";
import type {
  CardPaymentMethod,
  PaymentMethodListResponse,
} from "../../shared/types/paymentMethods";

export async function fetchPaymentContext(): Promise<PaymentContext> {
  return requestJson<PaymentContext>(
    "/mall/me/payments",
    {
      method: "GET",
      auth: "required",
      credentials: "include",
    },
  );
}

export async function fetchPaymentMethods(): Promise<{
  methods: CardPaymentMethod[];
  defaultMethod: CardPaymentMethod | null;
}> {
  const listBody = await requestJson<PaymentMethodListResponse>(
    "/mall/me/payment-methods",
    {
      method: "GET",
      auth: "required",
      credentials: "include",
      messages: {
        requestErrorMessage:
          "支払い方法の取得に失敗しました。",
      },
    },
  );

  const methods = Array.isArray(listBody?.data)
    ? listBody.data
    : [];

  const defaultMethod =
    methods.find((method) => method.isDefault) ??
    methods[0] ??
    null;

  return {
    methods,
    defaultMethod,
  };
}

export async function createOrder(
  input: CreateOrderRequest,
): Promise<CreatedOrder> {
  const order = await requestJson<CreatedOrder>(
    "/mall/me/orders",
    {
      method: "POST",
      auth: "required",
      credentials: "include",
      json: input,
    },
  );

  return {
    ...order,
    id: order.id ?? input.id,
    paid: order.paid ?? false,
  };
}