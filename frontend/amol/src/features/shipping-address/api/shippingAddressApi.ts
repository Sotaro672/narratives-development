//frontend\amol\src\features\shipping-address\api\shippingAddressApi.ts
import type {
  ErrorResponse,
  ShippingAddress,
  UserProfile,
} from "../types";
import {
  isErrorResponse,
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

export async function fetchShippingAddressPageInitialData({
  backendUrl,
  idToken,
}: FetchInitialDataInput): Promise<{
  userProfile: UserProfile | null;
  shippingAddresses: ShippingAddress[];
}> {
  const [userResponse, shippingAddressResponse] = await Promise.all([
    fetch(`${backendUrl}/mall/me/users`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
    }),
    fetch(`${backendUrl}/mall/me/shipping-addresses`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
    }),
  ]);

  const userContentType = userResponse.headers.get("content-type") || "";
  let userResponseBody: UserProfile | ErrorResponse | null = null;

  if (userContentType.includes("application/json")) {
    userResponseBody = await userResponse.json();
  }

  if (!userResponse.ok) {
    const errorMessage = isErrorResponse(userResponseBody)
      ? userResponseBody.error || "ユーザー情報の取得に失敗しました。"
      : "ユーザー情報の取得に失敗しました。";

    throw new Error(errorMessage);
  }

  const shippingAddressContentType =
    shippingAddressResponse.headers.get("content-type") || "";

  let shippingAddressResponseBody:
    | ShippingAddress[]
    | ErrorResponse
    | null = null;

  if (shippingAddressContentType.includes("application/json")) {
    shippingAddressResponseBody = await shippingAddressResponse.json();
  }

  if (!shippingAddressResponse.ok) {
    const errorMessage = isErrorResponse(shippingAddressResponseBody)
      ? shippingAddressResponseBody.error || "配送先情報の取得に失敗しました。"
      : "配送先情報の取得に失敗しました。";

    throw new Error(errorMessage);
  }

  const shippingAddresses = Array.isArray(shippingAddressResponseBody)
    ? shippingAddressResponseBody.filter(isShippingAddress)
    : [];

  return {
    userProfile: isUserProfile(userResponseBody) ? userResponseBody : null,
    shippingAddresses,
  };
}

export async function saveUserProfile({
  backendUrl,
  idToken,
  payload,
}: SaveUserProfileInput): Promise<UserProfile | null> {
  const response = await fetch(`${backendUrl}/mall/me/users`, {
    method: "PATCH",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  const contentType = response.headers.get("content-type") || "";
  let responseBody: UserProfile | ErrorResponse | null = null;

  if (contentType.includes("application/json")) {
    responseBody = await response.json();
  }

  if (!response.ok) {
    const errorMessage = isErrorResponse(responseBody)
      ? responseBody.error || "ユーザー情報の保存に失敗しました。"
      : "ユーザー情報の保存に失敗しました。";

    throw new Error(errorMessage);
  }

  return isUserProfile(responseBody) ? responseBody : null;
}

export async function saveShippingAddress({
  backendUrl,
  idToken,
  isEditMode,
  shippingAddressId,
  payload,
}: SaveShippingAddressInput): Promise<ShippingAddress | null> {
  const url = isEditMode
    ? `${backendUrl}/mall/me/shipping-addresses/${shippingAddressId}`
    : `${backendUrl}/mall/me/shipping-addresses`;

  const response = await fetch(url, {
    method: isEditMode ? "PATCH" : "POST",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  const contentType = response.headers.get("content-type") || "";
  let responseBody: ShippingAddress | ErrorResponse | null = null;

  if (contentType.includes("application/json")) {
    responseBody = await response.json();
  }

  if (!response.ok) {
    const errorMessage = isErrorResponse(responseBody)
      ? responseBody.error || "配送先情報の保存に失敗しました。"
      : "配送先情報の保存に失敗しました。";

    throw new Error(errorMessage);
  }

  return isShippingAddress(responseBody) ? responseBody : null;
}