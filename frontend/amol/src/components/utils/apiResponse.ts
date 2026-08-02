// frontend/amol/src/components/utils/apiResponse.ts

import { isRecord } from "./typeGuards";

export type ReadJsonResponseOptions<T> = {
  requestErrorMessage: string;
  nonJsonErrorMessage: string;
  invalidJsonErrorMessage?: string;
  fallbackValue?: T;
};

/**
 * APIレスポンスの data ラッパーを展開します。
 *
 * {
 *   data: {...}
 * }
 *
 * の場合は data を返し、それ以外は受け取った値をそのまま返します。
 */
export function unwrapApiData<T>(
  value: unknown,
): T {
  if (
    isRecord(value) &&
    value.data !== undefined &&
    value.data !== null
  ) {
    return value.data as T;
  }

  return value as T;
}

/**
 * APIのエラーレスポンスから表示用メッセージを取得します。
 *
 * error、message の順で確認します。
 */
function extractApiErrorMessage(
  value: unknown,
): string | null {
  const body =
    unwrapApiData<unknown>(value);

  if (!isRecord(body)) {
    return null;
  }

  const error = body.error;

  if (typeof error === "string") {
    const normalizedError =
      error.trim();

    if (normalizedError) {
      return normalizedError;
    }
  }

  const message = body.message;

  if (typeof message === "string") {
    const normalizedMessage =
      message.trim();

    if (normalizedMessage) {
      return normalizedMessage;
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
 * - HTTPエラー時は error / message を抽出
 * - JSON以外の正常レスポンスはエラー
 * - 空レスポンスが許可される場合は fallbackValue を使用
 * - response.text() は一度だけ実行
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
      decoded = JSON.parse(text);
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

    throw new Error(
      apiErrorMessage ||
        text ||
        options.requestErrorMessage,
    );
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
 * JSONレスポンスを読み込み、data ラッパーを展開して返します。
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