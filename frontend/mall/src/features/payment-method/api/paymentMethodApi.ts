// frontend/amol/src/features/payment-method/api/paymentMethodApi.ts

import {
  HttpError,
  requestJson,
} from "../../../lib/http";

import type {
  CardPaymentMethod,
  ConfirmedCardPayload,
  PaymentMethodDefaultResponse,
  PaymentMethodListResponse,
  SavePaymentMethodResponse,
  SetupIntentResponse,
  StripeConfigResponse,
} from "../../shared/types/paymentMethods";
import {
  selectPrimaryPaymentMethod,
} from "../utils/paymentMethodUtils";

export async function fetchStripeConfig(
  backendUrl: string,
): Promise<StripeConfigResponse | null> {
  void backendUrl;

  return requestJson<StripeConfigResponse | null>(
    "/mall/config/stripe",
    {
      method: "GET",
      auth: "none",
      credentials: "include",
      fallbackValue: null,
      messages: {
        requestErrorMessage:
          "Stripe 公開鍵の取得に失敗しました。",
      },
    },
  );
}

export async function fetchCurrentPaymentMethod(
  backendUrl: string,
  idToken: string,
): Promise<CardPaymentMethod | null> {
  void backendUrl;
  void idToken;

  const [listBody, defaultBody] = await Promise.all([
    requestJson<PaymentMethodListResponse>(
      "/mall/me/payment-methods",
      {
        method: "GET",
        auth: "required",
        credentials: "include",
        messages: {
          requestErrorMessage:
            "支払方法の取得に失敗しました。",
        },
      },
    ),
    requestJson<PaymentMethodDefaultResponse>(
      "/mall/me/payment-methods/default",
      {
        method: "GET",
        auth: "required",
        credentials: "include",
        messages: {
          requestErrorMessage:
            "既定の支払方法の取得に失敗しました。",
        },
      },
    ).catch((error: unknown) => {
      if (
        error instanceof HttpError &&
        error.status === 404
      ) {
        return null;
      }

      throw error;
    }),
  ]);

  return selectPrimaryPaymentMethod(
    listBody,
    defaultBody,
  );
}

export async function createSetupIntent(
  backendUrl: string,
  idToken: string,
  cardholderName: string,
): Promise<SetupIntentResponse | null> {
  void backendUrl;
  void idToken;

  return requestJson<SetupIntentResponse | null>(
    "/mall/me/payment-methods/setup-intent",
    {
      method: "POST",
      auth: "required",
      credentials: "include",
      json: {
        cardholderName,
      },
      fallbackValue: null,
      messages: {
        requestErrorMessage:
          "SetupIntent の作成に失敗しました。",
      },
    },
  );
}

export async function savePaymentMethod(
  backendUrl: string,
  idToken: string,
  payload: ConfirmedCardPayload,
): Promise<CardPaymentMethod | null> {
  void backendUrl;
  void idToken;

  const responseBody =
    await requestJson<SavePaymentMethodResponse | null>(
      "/mall/me/payment-methods",
      {
        method: "POST",
        auth: "required",
        credentials: "include",
        json: {
          stripeCustomerId:
            payload.stripeCustomerId,
          stripePaymentMethodId:
            payload.stripePaymentMethodId,
          brand:
            payload.brand,
          last4:
            payload.last4,
          expMonth:
            payload.expMonth,
          expYear:
            payload.expYear,
          cardholderName:
            payload.cardholderName,
          isDefault:
            true,
        },
        fallbackValue: null,
        messages: {
          requestErrorMessage:
            "支払方法の保存に失敗しました。",
        },
      },
    );

  return responseBody?.data ?? null;
}