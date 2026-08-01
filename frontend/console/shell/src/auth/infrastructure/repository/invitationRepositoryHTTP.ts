// frontend/console/shell/src/auth/infrastructure/repository/invitationRepositoryHTTP.ts

import {
  buildConsoleUrl,
} from "../../../shared/http/apiBase";

import {
  fetchJSON,
  HttpError,
} from "../../../shared/http/fetchJSON";

// ------------------------------
// 型定義
// ------------------------------

export type InvitationInfo = {
  companyName?: string;
  brandNames?: string[];
};

export type ValidateResponse = {
  companyName?: string;
  brandNames?: string[];
};

export type CompleteInvitationBackendPayload = {
  token: string;
  lastName: string;
  lastNameKana: string;
  firstName: string;
  firstNameKana: string;
};

type ErrorResponse = {
  error?: string;
  message?: string;
};

// ------------------------------
// Helpers
// ------------------------------

function safeTrim(
  value: unknown,
): string {
  return typeof value === "string"
    ? value.trim()
    : "";
}

function normalizeCompleteInvitationPayload(
  payload: CompleteInvitationBackendPayload,
): CompleteInvitationBackendPayload {
  const normalized:
    CompleteInvitationBackendPayload = {
      token:
        safeTrim(
          payload.token,
        ),
      lastName:
        safeTrim(
          payload.lastName,
        ),
      lastNameKana:
        safeTrim(
          payload.lastNameKana,
        ),
      firstName:
        safeTrim(
          payload.firstName,
        ),
      firstNameKana:
        safeTrim(
          payload.firstNameKana,
        ),
    };

  if (!normalized.token) {
    throw new Error(
      "token が指定されていません。",
    );
  }

  if (!normalized.lastName) {
    throw new Error(
      "lastName が指定されていません。",
    );
  }

  if (!normalized.lastNameKana) {
    throw new Error(
      "lastNameKana が指定されていません。",
    );
  }

  if (!normalized.firstName) {
    throw new Error(
      "firstName が指定されていません。",
    );
  }

  if (!normalized.firstNameKana) {
    throw new Error(
      "firstNameKana が指定されていません。",
    );
  }

  return normalized;
}

function normalizeStringArray(
  value: unknown,
): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  const normalized =
    value
      .map((item) =>
        safeTrim(
          item,
        ),
      )
      .filter(
        (item) =>
          item.length > 0,
      );

  return Array.from(
    new Set(
      normalized,
    ),
  );
}

function normalizeValidateResponse(
  data: unknown,
): ValidateResponse {
  if (
    typeof data !== "object" ||
    data === null
  ) {
    return {};
  }

  const record =
    data as Record<
      string,
      unknown
    >;

  const companyName =
    safeTrim(
      record.companyName,
    );

  const brandNames =
    normalizeStringArray(
      record.brandNames,
    );

  return {
    companyName:
      companyName ||
      undefined,
    brandNames:
      brandNames.length > 0
        ? brandNames
        : undefined,
  };
}

function validateResponseToInvitationInfo(
  data: ValidateResponse,
): InvitationInfo {
  return {
    companyName:
      data.companyName,
    brandNames:
      data.brandNames,
  };
}

/**
 * Backendのエラーレスポンスから、
 * 画面表示用のメッセージを取得する。
 */
function parseBackendErrorBody(
  bodyText: string | undefined,
): string | null {
  if (!bodyText) {
    return null;
  }

  try {
    const parsed =
      JSON.parse(
        bodyText,
      ) as ErrorResponse;

    const backendMessage =
      safeTrim(
        parsed.error,
      ) ||
      safeTrim(
        parsed.message,
      );

    return (
      backendMessage ||
      null
    );
  } catch {
    return null;
  }
}

/**
 * fetchJSONから送出されたエラーを、
 * Repository用のメッセージへ変換する。
 */
function toInvitationRepositoryError(
  error: unknown,
  fallback: string,
): Error {
  if (
    error instanceof HttpError
  ) {
    const backendMessage =
      parseBackendErrorBody(
        error.bodyText,
      );

    if (backendMessage) {
      return new Error(
        backendMessage,
      );
    }

    return new Error(
      `${fallback} (status ${error.status})`,
    );
  }

  if (
    error instanceof Error
  ) {
    return error;
  }

  return new Error(
    fallback,
  );
}

// ------------------------------
// 招待情報取得
// - POST /invitations/validate
// ------------------------------

export async function fetchInvitationInfo(
  token: string,
): Promise<InvitationInfo> {
  const data =
    await validateInvitation(
      token,
    );

  return validateResponseToInvitationInfo(
    data,
  );
}

// ------------------------------
// validateInvitation
// - POST /invitations/validate
// - 招待受諾前の公開APIのためAuthorizationは付与しない
// - 会社名とブランド名以外の機微情報は取得しない
// ------------------------------

export async function validateInvitation(
  token: string,
): Promise<ValidateResponse> {
  const normalizedToken =
    safeTrim(
      token,
    );

  if (!normalizedToken) {
    throw new Error(
      "token が指定されていません。",
    );
  }

  const url =
    buildConsoleUrl(
      "/invitations/validate",
    );

  try {
    const data =
      await fetchJSON<unknown>(
        url,
        {
          method: "POST",
          auth: "none",
          headers: {
            "Content-Type":
              "application/json",
          },
          body:
            JSON.stringify({
              token:
                normalizedToken,
            }),
        },
      );

    return normalizeValidateResponse(
      data,
    );
  } catch (error: unknown) {
    throw toInvitationRepositoryError(
      error,
      "招待の検証に失敗しました",
    );
  }
}

// ------------------------------
// completeInvitationOnBackend
// - POST /invitations/complete
// - UIDとemailはbodyへ送信しない
// - Firebase ID tokenはfetchJSONがAuthorizationへ付与する
// - 401時はID tokenを強制更新して1回だけ再送する
// - Backend側でID tokenからUIDとemailを取得する
// ------------------------------

export async function completeInvitationOnBackend(
  payload: CompleteInvitationBackendPayload,
): Promise<void> {
  const url =
    buildConsoleUrl(
      "/invitations/complete",
    );

  const body =
    normalizeCompleteInvitationPayload(
      payload,
    );

  try {
    await fetchJSON<void>(
      url,
      {
        method: "POST",
        auth: "required",
        headers: {
          "Content-Type":
            "application/json",
        },
        body:
          JSON.stringify(
            body,
          ),

        /**
         * complete APIが204、空body、
         * またはJSON以外を返す場合も成功として扱う。
         */
        allowNonJson: true,
      },
    );
  } catch (error: unknown) {
    throw toInvitationRepositoryError(
      error,
      "招待の完了処理に失敗しました",
    );
  }
}