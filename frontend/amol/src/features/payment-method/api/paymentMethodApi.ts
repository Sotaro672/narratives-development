//frontend\amol\src\features\payment-method\api\paymentMethodApi.ts
import type {
  CardPaymentMethod,
  ConfirmedCardPayload,
  PaymentMethodDefaultResponse,
  PaymentMethodListResponse,
  SavePaymentMethodResponse,
  SetupIntentResponse,
  StripeConfigResponse,
} from "../types";
import {
  readJsonResponse,
  selectPrimaryPaymentMethod,
} from "../utils/paymentMethodUtils";

export async function fetchStripeConfig(
  backendUrl: string,
): Promise<StripeConfigResponse | null> {
  const response = await fetch(`${backendUrl}/mall/config/stripe`, {
    method: "GET",
    credentials: "include",
  });

  const responseBody = await readJsonResponse<StripeConfigResponse>(response);

  if (!response.ok) {
    throw new Error(
      responseBody?.error || "Stripe 公開鍵の取得に失敗しました。",
    );
  }

  return responseBody;
}

export async function fetchCurrentPaymentMethod(
  backendUrl: string,
  idToken: string,
): Promise<CardPaymentMethod | null> {
  const [listResponse, defaultResponse] = await Promise.all([
    fetch(`${backendUrl}/mall/me/payment-methods`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
      credentials: "include",
    }),
    fetch(`${backendUrl}/mall/me/payment-methods/default`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
      credentials: "include",
    }),
  ]);

  const listBody =
    await readJsonResponse<PaymentMethodListResponse>(listResponse);

  const defaultBody =
    await readJsonResponse<PaymentMethodDefaultResponse>(defaultResponse);

  if (!listResponse.ok) {
    throw new Error(listBody?.error || "支払方法の取得に失敗しました。");
  }

  if (!defaultResponse.ok && defaultResponse.status !== 404) {
    throw new Error(
      defaultBody?.error || "既定の支払方法の取得に失敗しました。",
    );
  }

  return selectPrimaryPaymentMethod(listBody, defaultBody);
}

export async function createSetupIntent(
  backendUrl: string,
  idToken: string,
  cardholderName: string,
): Promise<SetupIntentResponse | null> {
  const response = await fetch(
    `${backendUrl}/mall/me/payment-methods/setup-intent`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${idToken}`,
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify({
        cardholderName,
      }),
    },
  );

  const responseBody = await readJsonResponse<SetupIntentResponse>(response);

  if (!response.ok) {
    throw new Error(responseBody?.error || "SetupIntent の作成に失敗しました。");
  }

  return responseBody;
}

export async function savePaymentMethod(
  backendUrl: string,
  idToken: string,
  payload: ConfirmedCardPayload,
): Promise<CardPaymentMethod | null> {
  const response = await fetch(`${backendUrl}/mall/me/payment-methods`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify({
      stripeCustomerId: payload.stripeCustomerId,
      stripePaymentMethodId: payload.stripePaymentMethodId,
      brand: payload.brand,
      last4: payload.last4,
      expMonth: payload.expMonth,
      expYear: payload.expYear,
      cardholderName: payload.cardholderName,
      isDefault: true,
    }),
  });

  const responseBody = await readJsonResponse<SavePaymentMethodResponse>(
    response,
  );

  if (!response.ok) {
    throw new Error(responseBody?.error || "支払方法の保存に失敗しました。");
  }

  return responseBody?.data ?? null;
}