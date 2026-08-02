// frontend/amol/src/features/wallet/api/avatarApi.ts

import {
  readJsonResponse,
  unwrapApiData,
} from "../../../components/utils/apiResponse";
import { isRecord } from "../../../components/utils/typeGuards";

import type { WalletAvatar } from "../types";

type FetchWalletPageDataInput = {
  backendUrl: string;
  idToken: string;
};

type FetchPublicWalletAvatarInput =
  FetchWalletPageDataInput & {
    avatarId: string;
  };

function toStringValue(
  value: unknown,
): string {
  return typeof value === "string"
    ? value
    : "";
}

function parseWalletAvatar(
  value: unknown,
  fallbackAvatarId = "",
): WalletAvatar {
  const body =
    unwrapApiData<unknown>(value);

  if (!isRecord(body)) {
    throw new Error(
      "アバター情報APIのレスポンス形式が不正です。",
    );
  }

  const avatarId =
    toStringValue(body.avatarId) ||
    fallbackAvatarId;

  if (!avatarId) {
    throw new Error(
      "アバター情報APIのレスポンスにavatarIdがありません。",
    );
  }

  return {
    avatarId,
    avatarName:
      toStringValue(body.avatarName),
    avatarIcon:
      toStringValue(body.avatarIcon),
    profile:
      toStringValue(body.profile),
  };
}

export async function fetchWalletAvatar({
  backendUrl,
  idToken,
}: FetchWalletPageDataInput): Promise<WalletAvatar> {
  const response = await fetch(
    `${backendUrl}/mall/me/avatars`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization:
          `Bearer ${idToken}`,
      },
    },
  );

  const responseBody =
    await readJsonResponse<unknown>(
      response,
      {
        requestErrorMessage:
          "アバター情報の取得に失敗しました。",
        nonJsonErrorMessage:
          "アバター情報APIがJSON以外を返しました。",
        invalidJsonErrorMessage:
          "アバター情報APIのJSON形式が不正です。",
      },
    );

  return parseWalletAvatar(
    responseBody,
  );
}

export async function fetchPublicWalletAvatar({
  backendUrl,
  idToken,
  avatarId,
}: FetchPublicWalletAvatarInput): Promise<WalletAvatar> {
  const encodedAvatarId =
    encodeURIComponent(avatarId);

  const response = await fetch(
    `${backendUrl}/mall/avatars/${encodedAvatarId}`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization:
          `Bearer ${idToken}`,
      },
    },
  );

  const responseBody =
    await readJsonResponse<unknown>(
      response,
      {
        requestErrorMessage:
          "アバター情報の取得に失敗しました。",
        nonJsonErrorMessage:
          "アバター情報APIがJSON以外を返しました。",
        invalidJsonErrorMessage:
          "アバター情報APIのJSON形式が不正です。",
      },
    );

  return parseWalletAvatar(
    responseBody,
    avatarId,
  );
}