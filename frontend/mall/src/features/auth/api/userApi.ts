// frontend/mall/src/features/auth/api/userApi.ts

import type { User } from "firebase/auth";

import { buildApiUrl } from "../../../lib/apiBaseUrl";
import { requestJson } from "../../../lib/http";

export type MallUserProfile = {
  id: string;
  first_name: string;
  first_name_kana: string;
  last_name: string;
  last_name_kana: string;
  createdAt: string;
  updatedAt: string;
};

export type SaveUserResult =
  | {
      ok: true;
      message: string;
    }
  | {
      ok: false;
      error: string;
    };

export type SaveUserBody = {
  lastName: string;
  lastNameKana: string;
  firstName: string;
  firstNameKana: string;
};

type SaveUserParams = {
  currentUser: User | null;
  backendUrl: string;
  body: SaveUserBody;
};

async function readErrorMessage(
  response: Response,
): Promise<string> {
  const contentType =
    response.headers.get("content-type") || "";

  if (
    contentType.includes(
      "application/json",
    )
  ) {
    const body = (await response.json().catch(
      () => null,
    )) as
      | {
          error?: string;
          message?: string;
        }
      | null;

    if (body?.error) {
      return body.error;
    }

    if (body?.message) {
      return body.message;
    }
  }

  const text = await response.text().catch(
    () => "",
  );

  return (
    text ||
    `保存に失敗しました (${response.status})`
  );
}

export async function fetchCurrentUserProfile(): Promise<MallUserProfile> {
  return requestJson<MallUserProfile>(
    "/mall/me/users",
    {
      method: "GET",
      auth: "required",
      messages: {
        requestErrorMessage:
          "ユーザー情報の取得に失敗しました。",
      },
    },
  );
}

export async function saveUserProfile({
  currentUser,
  backendUrl,
  body,
}: SaveUserParams): Promise<SaveUserResult> {
  if (!currentUser) {
    return {
      ok: false,
      error: "サインインが必要です。",
    };
  }

  if (!currentUser.uid) {
    return {
      ok: false,
      error: "uid が取得できませんでした。",
    };
  }

  const token =
    await currentUser.getIdToken(true);

  if (!token) {
    return {
      ok: false,
      error:
        "認証トークンが取得できませんでした。再ログインしてください。",
    };
  }

  try {
    const url = buildApiUrl(
      backendUrl,
      "/mall/me/users",
    );

    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        last_name: body.lastName,
        last_name_kana: body.lastNameKana,
        first_name: body.firstName,
        first_name_kana:
          body.firstNameKana,
      }),
    });

    if (!response.ok) {
      return {
        ok: false,
        error:
          await readErrorMessage(response),
      };
    }

    return {
      ok: true,
      message: "保存しました。",
    };
  } catch (error) {
    if (error instanceof Error) {
      return {
        ok: false,
        error: error.message,
      };
    }

    return {
      ok: false,
      error:
        "ユーザー情報の保存に失敗しました。",
    };
  }
}