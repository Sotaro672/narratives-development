// frontend/amol/src/lib/http/httpClient.ts

import {
  buildBackendUrl,
} from "../apiBaseUrl";

import {
  readJsonDataResponse,
  readJsonResponse,
} from "../apiResponse";

import {
  getAuthHeaders,
  getOptionalAuthHeaders,
} from "../authHeaders";

import type {
  ApiJsonRequestOptions,
  ApiQueryParams,
  ApiRequestOptions,
} from "./apiTypes";

function appendQuery(
  url: string,
  query?: ApiQueryParams,
): string {
  if (!query) {
    return url;
  }

  const result = new URL(url);

  Object.entries(query).forEach(
    ([key, value]) => {
      if (
        value === undefined ||
        value === null
      ) {
        return;
      }

      result.searchParams.set(
        key,
        String(value),
      );
    },
  );

  return result.toString();
}

async function buildHeaders(
  options: ApiRequestOptions,
): Promise<Headers> {
  const headers =
    new Headers(options.headers);

  if (!headers.has("Accept")) {
    headers.set(
      "Accept",
      "application/json",
    );
  }

  if (
    options.json !== undefined &&
    !headers.has("Content-Type")
  ) {
    headers.set(
      "Content-Type",
      "application/json",
    );
  }

  if (options.auth === "required") {
    const authHeaders =
      await getAuthHeaders();

    new Headers(authHeaders).forEach(
      (value, key) => {
        headers.set(key, value);
      },
    );
  }

  if (options.auth === "optional") {
    const authHeaders =
      await getOptionalAuthHeaders();

    if (authHeaders) {
      new Headers(authHeaders).forEach(
        (value, key) => {
          headers.set(key, value);
        },
      );
    }
  }

  return headers;
}

export async function requestRaw(
  path: string,
  options: ApiRequestOptions = {},
): Promise<Response> {
  const {
    auth: _auth,
    query,
    json,
    headers: _headers,
    ...init
  } = options;

  const url = appendQuery(
    buildBackendUrl(path),
    query,
  );

  const headers =
    await buildHeaders(options);

  return fetch(url, {
    ...init,
    headers,
    body:
      json === undefined
        ? undefined
        : JSON.stringify(json),
  });
}

export async function requestJson<T>(
  path: string,
  options:
    ApiJsonRequestOptions<T> = {},
): Promise<T> {
  const response =
    await requestRaw(
      path,
      options,
    );

  const messages = {
    requestErrorMessage:
      options.messages?.requestErrorMessage ??
      "APIリクエストに失敗しました。",

    nonJsonErrorMessage:
      options.messages?.nonJsonErrorMessage ??
      "APIがJSON以外を返しました。",

    invalidJsonErrorMessage:
      options.messages?.invalidJsonErrorMessage ??
      "APIのJSON形式が不正です。",
  };

  const readOptions = {
    ...messages,
    ...(Object.prototype.hasOwnProperty.call(
      options,
      "fallbackValue",
    )
      ? {
          fallbackValue:
            options.fallbackValue,
        }
      : {}),
  };

  if (options.unwrapData) {
    return readJsonDataResponse<T>(
      response,
      readOptions,
    );
  }

  return readJsonResponse<T>(
    response,
    readOptions,
  );
}

export async function requestVoid(
  path: string,
  options: ApiRequestOptions = {},
): Promise<void> {
  const response =
    await requestRaw(
      path,
      options,
    );

  await readJsonResponse<void>(
    response,
    {
      requestErrorMessage:
        "APIリクエストに失敗しました。",

      nonJsonErrorMessage:
        "APIがJSON以外を返しました。",

      invalidJsonErrorMessage:
        "APIのJSON形式が不正です。",

      fallbackValue:
        undefined,
    },
  );
}