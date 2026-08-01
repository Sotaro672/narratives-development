// frontend/console/shell/src/shared/http/fetchJSON.ts

import {
  getAuthHeaders,
  getAuthHeadersOrThrow,
} from "./authHeaders";

/**
 * Shared fetch helpers for module federation remotes.
 *
 * - Supports public, optional-auth and required-auth requests
 * - Adds Firebase Authorization headers through authHeaders.ts
 * - Retries once with a refreshed ID token after a 401 response
 * - Ensures JSON responses unless allowNonJson=true
 * - Throws a structured HttpError on non-2xx responses
 * - Limits the size of error response bodies
 */

export type HttpErrorInit = {
  url: string;
  method?: string;
  status: number;
  statusText?: string;
  contentType?: string;
  bodyText?: string;
};

export class HttpError extends Error {
  name = "HttpError" as const;

  readonly url: string;
  readonly method?: string;
  readonly status: number;
  readonly statusText?: string;
  readonly contentType?: string;
  readonly bodyText?: string;

  constructor(
    init: HttpErrorInit,
  ) {
    const message =
      `${init.method ?? "GET"} ${init.status} ${init.url}`;

    super(message);

    this.url = init.url;
    this.method = init.method;
    this.status = init.status;
    this.statusText = init.statusText;
    this.contentType = init.contentType;
    this.bodyText = init.bodyText;
  }
}

/**
 * リクエストの認証方針。
 *
 * none:
 * - Authorizationヘッダーを自動設定しない
 * - 公開APIで使用する
 *
 * optional:
 * - ログイン中であればAuthorizationを設定する
 * - 未認証でもリクエストを続行する
 *
 * required:
 * - Authorizationを必須とする
 * - 未認証またはtoken取得失敗時はAuthTokenErrorを送出する
 */
export type FetchAuthMode =
  | "none"
  | "optional"
  | "required";

export type FetchJSONOptions =
  RequestInit & {
    /**
     * リクエストの認証方針。
     *
     * デフォルトはnone。
     * 既存の公開APIや、呼出元が独自にAuthorizationを
     * 設定している処理への影響を防ぐ。
     */
    auth?: FetchAuthMode;

    /**
     * 401を受け取った場合にID tokenを強制更新し、
     * リクエストを1回だけ再送するか。
     *
     * authがoptionalまたはrequiredの場合、
     * デフォルトはtrue。
     */
    retryUnauthorized?: boolean;

    /**
     * trueの場合、application/json以外のレスポンスを許可する。
     *
     * Content-TypeがJSONでない場合は、
     * response bodyをstringとして返す。
     */
    allowNonJson?: boolean;

    /**
     * HttpError.bodyTextへ格納する最大文字数。
     *
     * デフォルトは2000文字。
     */
    errorBodyLimit?: number;
  };

type RequestAuthResult = {
  headers: Headers;
  hasGeneratedAuthorization: boolean;
};

/**
 * bodyTextを指定された最大文字数に制限する。
 */
function limitText(
  text: string,
  limit: number,
): string {
  if (!text) {
    return "";
  }

  return text.length > limit
    ? text.slice(0, limit)
    : text;
}

/**
 * Response bodyを安全に文字列として取得する。
 */
async function readTextSafely(
  response: Response,
  limit: number,
): Promise<string> {
  try {
    const text =
      await response.text();

    return limitText(
      text,
      limit,
    );
  } catch {
    return "";
  }
}

/**
 * Content-TypeがJSON形式か判定する。
 *
 * application/jsonだけでなく、
 * application/problem+jsonなども許可する。
 */
function isJsonContentType(
  contentType: string,
): boolean {
  const normalized =
    contentType
      .toLowerCase()
      .split(";")[0]
      ?.trim() ?? "";

  return (
    normalized ===
      "application/json" ||
    normalized.endsWith(
      "+json",
    )
  );
}

/**
 * 認証方針に従ってAuthorizationヘッダーを組み立てる。
 *
 * authHeaders.tsが生成したAuthorizationは、
 * 呼出元が指定した古いAuthorizationより優先する。
 */
async function buildRequestHeaders(
  baseHeaders: Headers,
  authMode: FetchAuthMode,
  forceRefresh: boolean,
): Promise<RequestAuthResult> {
  const headers =
    new Headers(
      baseHeaders,
    );

  if (authMode === "none") {
    return {
      headers,
      hasGeneratedAuthorization:
        false,
    };
  }

  const authHeaders =
    authMode === "required"
      ? await getAuthHeadersOrThrow(
          forceRefresh,
        )
      : await getAuthHeaders(
          forceRefresh,
        );

  const authorization =
    authHeaders.Authorization;

  if (authorization) {
    headers.set(
      "Authorization",
      authorization,
    );
  }

  return {
    headers,
    hasGeneratedAuthorization:
      Boolean(authorization),
  };
}

/**
 * ベースRequestを複製し、認証ヘッダーを設定したRequestを生成する。
 *
 * Requestのbodyを直接再利用せずcloneすることで、
 * 401 retry時に同じリクエストを再送できるようにする。
 */
async function buildRequest(
  baseRequest: Request,
  authMode: FetchAuthMode,
  forceRefresh: boolean,
): Promise<{
  request: Request;
  hasGeneratedAuthorization: boolean;
}> {
  const {
    headers,
    hasGeneratedAuthorization,
  } =
    await buildRequestHeaders(
      baseRequest.headers,
      authMode,
      forceRefresh,
    );

  const request =
    new Request(
      baseRequest.clone(),
      {
        headers,
      },
    );

  return {
    request,
    hasGeneratedAuthorization,
  };
}

/**
 * ResponseをHttpErrorへ変換する。
 */
async function createHttpError(
  response: Response,
  request: Request,
  errorBodyLimit: number,
): Promise<HttpError> {
  const bodyText =
    await readTextSafely(
      response,
      errorBodyLimit,
    );

  return new HttpError({
    url:
      response.url ||
      request.url,
    method:
      request.method,
    status:
      response.status,
    statusText:
      response.statusText,
    contentType:
      response.headers.get(
        "content-type",
      ) ?? "",
    bodyText,
  });
}

/**
 * 正常レスポンスのbodyを解析する。
 */
async function parseSuccessResponse<T>(
  response: Response,
  request: Request,
  allowNonJson: boolean,
  errorBodyLimit: number,
): Promise<T> {
  if (
    response.status === 204 ||
    response.status === 205
  ) {
    return undefined as T;
  }

  const contentType =
    response.headers.get(
      "content-type",
    ) ?? "";

  const jsonResponse =
    isJsonContentType(
      contentType,
    );

  if (
    !jsonResponse &&
    !allowNonJson
  ) {
    const bodyText =
      await readTextSafely(
        response,
        errorBodyLimit,
      );

    throw new HttpError({
      url:
        response.url ||
        request.url,
      method:
        request.method,
      status:
        response.status,
      statusText:
        response.statusText,
      contentType,
      bodyText: limitText(
        [
          `Unexpected content-type: ${contentType || "(empty)"}`,
          bodyText,
        ]
          .filter(Boolean)
          .join("\n"),
        errorBodyLimit,
      ),
    });
  }

  const bodyText =
    await readTextSafely(
      response,
      errorBodyLimit,
    );

  if (!bodyText) {
    return undefined as T;
  }

  if (!jsonResponse) {
    return bodyText as T;
  }

  try {
    return JSON.parse(
      bodyText,
    ) as T;
  } catch (error: unknown) {
    const message =
      error instanceof Error
        ? error.message
        : String(error);

    throw new HttpError({
      url:
        response.url ||
        request.url,
      method:
        request.method,
      status:
        response.status,
      statusText:
        response.statusText,
      contentType,
      bodyText: limitText(
        [
          `Failed to parse JSON: ${message}`,
          bodyText,
        ].join("\n"),
        errorBodyLimit,
      ),
    });
  }
}

/**
 * fetchJSON
 *
 * 認証方針に従ってfetchを実行し、
 * JSONまたは許可された非JSONレスポンスを返す。
 *
 * authがoptionalまたはrequiredで401を受け取った場合、
 * Firebase ID tokenを強制更新して1回だけ再送する。
 *
 * 使用例:
 *
 * 公開API:
 *   await fetchJSON<Response>(url)
 *
 * 認証必須API:
 *   await fetchJSON<Response>(url, {
 *     auth: "required",
 *   })
 *
 * 認証必須JSON POST:
 *   await fetchJSON<Response>(url, {
 *     method: "POST",
 *     auth: "required",
 *     headers: {
 *       "Content-Type": "application/json",
 *     },
 *     body: JSON.stringify(payload),
 *   })
 */
export async function fetchJSON<
  T = unknown,
>(
  input: RequestInfo | URL,
  options: FetchJSONOptions = {},
): Promise<T> {
  const {
    auth: authMode = "none",
    retryUnauthorized,
    allowNonJson = false,
    errorBodyLimit = 2000,
    ...requestInit
  } = options;

  const shouldRetryUnauthorized =
    retryUnauthorized ??
    authMode !== "none";

  /**
   * 再送時にbodyを再利用できるよう、
   * 元のRequestはfetchへ直接渡さず
   * テンプレートとして保持する。
   */
  const baseRequest =
    new Request(
      input,
      requestInit,
    );

  const firstAttempt =
    await buildRequest(
      baseRequest,
      authMode,
      false,
    );

  let request =
    firstAttempt.request;

  let response =
    await fetch(request);

  /**
   * 認証付きリクエストが401になった場合のみ、
   * tokenを強制更新して1回だけ再送する。
   */
  if (
    response.status === 401 &&
    shouldRetryUnauthorized &&
    authMode !== "none"
  ) {
    const retryAttempt =
      await buildRequest(
        baseRequest,
        authMode,
        true,
      );

    /**
     * optional認証でtokenを取得できなかった場合は、
     * 同じ未認証リクエストを再送しても結果が変わらないため
     * retryしない。
     */
    if (
      authMode === "required" ||
      retryAttempt
        .hasGeneratedAuthorization
    ) {
      request =
        retryAttempt.request;

      response =
        await fetch(request);
    }
  }

  if (!response.ok) {
    throw await createHttpError(
      response,
      request,
      errorBodyLimit,
    );
  }

  return parseSuccessResponse<T>(
    response,
    request,
    allowNonJson,
    errorBodyLimit,
  );
}