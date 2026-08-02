// frontend/amol/src/features/wallet/api/avatarApi.ts

import { isRecord } from "../../shared/utils/typeGuards";
import type { WalletAvatar } from "../types";

type FetchWalletPageDataInput = {
  backendUrl: string;
  idToken: string;
};

type FetchPublicWalletAvatarInput = {
  backendUrl: string;
  idToken: string;
  avatarId: string;
};

function unwrapData(value: unknown): unknown {
  if (!isRecord(value)) {
    return value;
  }

  return value.data ?? value;
}

function getErrorMessageFromBody(
  value: unknown,
): string | null {
  const body = unwrapData(value);

  if (!isRecord(body)) {
    return null;
  }

  const error = body.error;

  return typeof error === "string" && error
    ? error
    : null;
}

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
  const body = unwrapData(value);

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
        Authorization: `Bearer ${idToken}`,
      },
    },
  );

  const contentType =
    response.headers.get("content-type") ||
    "";

  if (!response.ok) {
    if (
      contentType.includes(
        "application/json",
      )
    ) {
      const responseBody: unknown =
        await response.json();

      const error =
        getErrorMessageFromBody(
          responseBody,
        );

      if (error) {
        throw new Error(error);
      }
    }

    throw new Error(
      "アバター情報の取得に失敗しました。",
    );
  }

  if (
    !contentType.includes(
      "application/json",
    )
  ) {
    throw new Error(
      "アバター情報APIがJSON以外を返しました。",
    );
  }

  const responseBody: unknown =
    await response.json();

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
        Authorization: `Bearer ${idToken}`,
      },
    },
  );

  const contentType =
    response.headers.get("content-type") ||
    "";

  if (!response.ok) {
    if (
      contentType.includes(
        "application/json",
      )
    ) {
      const responseBody: unknown =
        await response.json();

      const error =
        getErrorMessageFromBody(
          responseBody,
        );

      if (error) {
        throw new Error(error);
      }
    }

    throw new Error(
      "アバター情報の取得に失敗しました。",
    );
  }

  if (
    !contentType.includes(
      "application/json",
    )
  ) {
    throw new Error(
      "アバター情報APIがJSON以外を返しました。",
    );
  }

  const responseBody: unknown =
    await response.json();

  return parseWalletAvatar(
    responseBody,
    avatarId,
  );
}