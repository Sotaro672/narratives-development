// frontend/amol/src/features/shipping-address/api/shippingAddressApi.ts

import {
  requestJson,
} from "../../../lib/http";

import type {
  ShippingAddress,
  UserProfile,
} from "../../shared/types/shippingAddress";
import {
  isShippingAddress,
  isUserProfile,
} from "../utils/zipCode";

type FetchInitialDataInput = {
  backendUrl: string;
  idToken: string;
};

type SaveShippingAddressInput = {
  backendUrl: string;
  idToken: string;
  isEditMode: boolean;
  shippingAddressId: string;
  payload: {
    zipCode: string;
    state: string;
    city: string;
    street: string;
    street2: string;
    country: string;
  };
};

type SaveUserProfileInput = {
  backendUrl: string;
  idToken: string;
  payload: {
    last_name: string;
    first_name: string;
    last_name_kana: string;
    first_name_kana: string;
  };
};

export async function fetchShippingAddressPageInitialData(
  _input: FetchInitialDataInput,
): Promise<{
  userProfile: UserProfile | null;
  shippingAddresses: ShippingAddress[];
}> {
  const [
    userResponseBody,
    shippingAddressResponseBody,
  ] = await Promise.all([
    requestJson<unknown>(
      "/mall/me/users",
      {
        method: "GET",
        auth: "required",
        fallbackValue: null,
        messages: {
          requestErrorMessage:
            "ユーザー情報の取得に失敗しました。",
        },
      },
    ),
    requestJson<unknown>(
      "/mall/me/shipping-addresses",
      {
        method: "GET",
        auth: "required",
        fallbackValue: null,
        messages: {
          requestErrorMessage:
            "配送先情報の取得に失敗しました。",
        },
      },
    ),
  ]);

  const shippingAddresses =
    Array.isArray(shippingAddressResponseBody)
      ? shippingAddressResponseBody.filter(
          isShippingAddress,
        )
      : [];

  return {
    userProfile: isUserProfile(userResponseBody)
      ? userResponseBody
      : null,
    shippingAddresses,
  };
}

export async function saveUserProfile({
  payload,
}: SaveUserProfileInput): Promise<UserProfile | null> {
  const responseBody =
    await requestJson<unknown>(
      "/mall/me/users",
      {
        method: "PATCH",
        auth: "required",
        json: payload,
        fallbackValue: null,
        messages: {
          requestErrorMessage:
            "ユーザー情報の保存に失敗しました。",
        },
      },
    );

  return isUserProfile(responseBody)
    ? responseBody
    : null;
}

export async function saveShippingAddress({
  isEditMode,
  shippingAddressId,
  payload,
}: SaveShippingAddressInput): Promise<ShippingAddress | null> {
  const path = isEditMode
    ? `/mall/me/shipping-addresses/${shippingAddressId}`
    : "/mall/me/shipping-addresses";

  const responseBody =
    await requestJson<unknown>(
      path,
      {
        method: isEditMode
          ? "PATCH"
          : "POST",
        auth: "required",
        json: payload,
        fallbackValue: null,
        messages: {
          requestErrorMessage:
            "配送先情報の保存に失敗しました。",
        },
      },
    );

  return isShippingAddress(responseBody)
    ? responseBody
    : null;
}