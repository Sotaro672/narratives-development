// frontend/amol/src/lib/apiResponse.ts

import {
  HttpError,
} from "./http/httpError";

export type ReadJsonResponseOptions<T> = {
  requestErrorMessage: string;
  nonJsonErrorMessage: string;
  invalidJsonErrorMessage?: string;
  fallbackValue?: T;
};

function isRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value)
  );
}

/**
 * APIレスポンスのdataラッパーを展開します。
 *
 * {
 *   data: {...}
 * }
 *
 * の場合はdataを返し、それ以外は受け取った値をそのまま返します。
 *
 * dataがnullまたはundefinedの場合も、
 * dataプロパティが存在すればその値を返します。
 */
export function unwrapApiData<T>(
  value: unknown,
): T {
  if (
    isRecord(value) &&
    Object.prototype.hasOwnProperty.call(
      value,
      "data",
    )
  ) {
    return value.data as T;
  }

  return value as T;
}

/**
 * APIのエラーレスポンスから表示用メッセージを取得します。
 *
 * errorMessage、detail、message、errorの順で確認します。
 */
function extractApiErrorMessage(
  value: unknown,
): string | null {
  const body =
    unwrapApiData<unknown>(value);

  if (!isRecord(body)) {
    return null;
  }

  const candidates = [
    body.errorMessage,
    body.detail,
    body.message,
    body.error,
  ];

  for (const candidate of candidates) {
    if (typeof candidate !== "string") {
      continue;
    }

    const normalizedCandidate =
      candidate.trim();

    if (normalizedCandidate) {
      return normalizedCandidate;
    }
  }

  return null;
}

function hasFallbackValue<T>(
  options: ReadJsonResponseOptions<T>,
): boolean {
  return Object.prototype.hasOwnProperty.call(
    options,
    "fallbackValue",
  );
}

/**
 * ResponseをJSONとして読み込みます。
 *
 * - HTTPエラー時はAPIレスポンスからメッセージを抽出
 * - JSON以外の正常レスポンスはエラー
 * - 空レスポンスが許可される場合はfallbackValueを使用
 * - response.text()は一度だけ実行
 * - HTTPエラーはHttpErrorとして返す
 */
export async function readJsonResponse<T>(
  response: Response,
  options: ReadJsonResponseOptions<T>,
): Promise<T> {
  const contentType = (
    response.headers.get(
      "content-type",
    ) || ""
  ).toLowerCase();

  const isJsonResponse =
    contentType.includes(
      "application/json",
    );

  const text = await response
    .text()
    .catch(() => "");

  let decoded: unknown;
  let jsonParsed = false;

  if (isJsonResponse && text) {
    try {
      decoded =
        JSON.parse(text) as unknown;

      jsonParsed = true;
    } catch {
      decoded = undefined;
    }
  }

  if (!response.ok) {
    const apiErrorMessage =
      jsonParsed
        ? extractApiErrorMessage(
            decoded,
          )
        : null;

    throw new HttpError({
      message:
        apiErrorMessage ||
        text.trim() ||
        options.requestErrorMessage,

      status:
        response.status,

      url:
        response.url,

      body:
        jsonParsed
          ? decoded
          : text || undefined,
    });
  }

  if (!isJsonResponse) {
    if (hasFallbackValue(options)) {
      return options.fallbackValue as T;
    }

    throw new Error(
      options.nonJsonErrorMessage,
    );
  }

  if (!text) {
    if (hasFallbackValue(options)) {
      return options.fallbackValue as T;
    }

    throw new Error(
      options.invalidJsonErrorMessage ||
        options.nonJsonErrorMessage,
    );
  }

  if (!jsonParsed) {
    if (hasFallbackValue(options)) {
      return options.fallbackValue as T;
    }

    throw new Error(
      options.invalidJsonErrorMessage ||
        options.nonJsonErrorMessage,
    );
  }

  return decoded as T;
}

/**
 * JSONレスポンスを読み込み、
 * dataラッパーを展開して返します。
 */
export async function readJsonDataResponse<T>(
  response: Response,
  options: ReadJsonResponseOptions<T>,
): Promise<T> {
  const readOptions:
    ReadJsonResponseOptions<unknown> = {
      requestErrorMessage:
        options.requestErrorMessage,

      nonJsonErrorMessage:
        options.nonJsonErrorMessage,

      invalidJsonErrorMessage:
        options.invalidJsonErrorMessage,
    };

  if (hasFallbackValue(options)) {
    readOptions.fallbackValue =
      options.fallbackValue;
  }

  const body =
    await readJsonResponse<unknown>(
      response,
      readOptions,
    );

  return unwrapApiData<T>(body);
}