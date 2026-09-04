// frontend/amol/src/features/shipping-address/api/shippingAddressApi.ts

import { requestJson } from "../../../lib/http";
import type { ShippingAddress, UserProfile } from "../../shared/types/shippingAddress";

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
  userProfile: UserProfile;
  shippingAddresses: ShippingAddress[];
}> {
  const [userProfile, shippingAddresses] = await Promise.all([
    requestJson<UserProfile>("/mall/me/users", {
      method: "GET",
      auth: "required",
      messages: {
        requestErrorMessage: "ユーザー情報の取得に失敗しました。",
      },
    }),
    requestJson<ShippingAddress[]>("/mall/me/shipping-addresses", {
      method: "GET",
      auth: "required",
      messages: {
        requestErrorMessage: "配送先情報の取得に失敗しました。",
      },
    }),
  ]);

  return { userProfile, shippingAddresses };
}

export async function saveUserProfile({
  payload,
}: SaveUserProfileInput): Promise<UserProfile> {
  return requestJson<UserProfile>("/mall/me/users", {
    method: "PATCH",
    auth: "required",
    json: payload,
    messages: {
      requestErrorMessage: "ユーザー情報の保存に失敗しました。",
    },
  });
}

export async function saveShippingAddress({
  isEditMode,
  shippingAddressId,
  payload,
}: SaveShippingAddressInput): Promise<ShippingAddress> {
  const path = isEditMode
    ? `/mall/me/shipping-addresses/${encodeURIComponent(shippingAddressId)}`
    : "/mall/me/shipping-addresses";

  return requestJson<ShippingAddress>(path, {
    method: isEditMode ? "PATCH" : "POST",
    auth: "required",
    json: payload,
    messages: {
      requestErrorMessage: "配送先情報の保存に失敗しました。",
    },
  });
}