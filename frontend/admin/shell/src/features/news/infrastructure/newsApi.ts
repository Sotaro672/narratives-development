// frontend/admin/shell/src/features/news/infrastructure/newsApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";

import type {
  CreateNewsInput,
  News,
  NewsPage,
} from "../../../shared/type/news";

const BACKEND_BASE_URL = import.meta.env.VITE_BACKEND_BASE_URL
  ?.trim()
  .replace(/\/+$/, "");

type ListNewsParams = {
  page?: number;
  perPage?: number;
};

export type NewsUploadProgress = {
  loaded: number;
  total: number;
  percentage: number;
};

export type NewsUploadProgressHandler = (
  progress: NewsUploadProgress,
) => void;

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }

  return BACKEND_BASE_URL;
}

async function requireOk(
  response: Response,
  fallbackMessage: string,
): Promise<void> {
  if (response.ok) {
    return;
  }

  let detail = "";

  try {
    const body = (await response.json()) as {
      error?: string;
    };

    detail = body.error ? ` error=${body.error}` : "";
  } catch {
    // Response body may not be JSON.
  }

  throw new Error(
    `${fallbackMessage} status=${response.status}${detail}`,
  );
}

function normalizePositiveInteger(
  value: number | undefined,
): number | undefined {
  if (value === undefined || !Number.isFinite(value)) {
    return undefined;
  }

  const normalized = Math.trunc(value);
  return normalized > 0 ? normalized : undefined;
}

function parseXHRResponse<T>(
  xhr: XMLHttpRequest,
): T | null {
  if (!xhr.responseText) {
    return null;
  }

  try {
    return JSON.parse(xhr.responseText) as T;
  } catch {
    return null;
  }
}

function createXHRHttpError(
  xhr: XMLHttpRequest,
  fallbackMessage: string,
): Error {
  const response = parseXHRResponse<{
    error?: string;
  }>(xhr);

  const detail = response?.error
    ? ` error=${response.error}`
    : "";

  return new Error(
    `${fallbackMessage} status=${xhr.status}${detail}`,
  );
}

export async function listNews(
  params: ListNewsParams = {},
): Promise<NewsPage> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();
  const searchParams = new URLSearchParams();

  const page = normalizePositiveInteger(params.page);
  const perPage = normalizePositiveInteger(params.perPage);

  if (page !== undefined) {
    searchParams.set("page", String(page));
  }

  if (perPage !== undefined) {
    searchParams.set("perPage", String(perPage));
  }

  const query = searchParams.toString();
  const url = query
    ? `${backendBaseUrl}/admin/news?${query}`
    : `${backendBaseUrl}/admin/news`;

  const response = await fetch(url, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  await requireOk(
    response,
    "Failed to load news.",
  );

  return response.json() as Promise<NewsPage>;
}

export async function createNews(
  input: CreateNewsInput,
  onUploadProgress?: NewsUploadProgressHandler,
): Promise<News> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const formData = new FormData();
  formData.set("title", input.title);
  formData.set("body", input.body);

  if (input.image) {
    formData.set("image", input.image);

    if (input.imageAlt !== undefined) {
      formData.set("imageAlt", input.imageAlt);
    }
  }

  return new Promise<News>((resolve, reject) => {
    const xhr = new XMLHttpRequest();

    xhr.open(
      "POST",
      `${backendBaseUrl}/admin/news`,
      true,
    );

    xhr.setRequestHeader(
      "Accept",
      "application/json",
    );

    for (const [name, value] of Object.entries(authHeaders)) {
      xhr.setRequestHeader(name, value);
    }

    if (input.image && onUploadProgress) {
      xhr.upload.addEventListener(
        "progress",
        (event) => {
          if (!event.lengthComputable || event.total <= 0) {
            return;
          }

          onUploadProgress({
            loaded: event.loaded,
            total: event.total,
            percentage: Math.min(
              100,
              Math.max(
                0,
                (event.loaded / event.total) * 100,
              ),
            ),
          });
        },
      );
    }

    xhr.addEventListener("load", () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        const response = parseXHRResponse<News>(xhr);

        if (!response) {
          reject(
            new Error(
              "Failed to publish news. Invalid response.",
            ),
          );
          return;
        }

        resolve(response);
        return;
      }

      reject(
        createXHRHttpError(
          xhr,
          "Failed to publish news.",
        ),
      );
    });

    xhr.addEventListener("error", () => {
      reject(
        new Error(
          "Failed to publish news. Network error.",
        ),
      );
    });

    xhr.addEventListener("abort", () => {
      reject(
        new Error(
          "Failed to publish news. Request aborted.",
        ),
      );
    });

    xhr.addEventListener("timeout", () => {
      reject(
        new Error(
          "Failed to publish news. Request timed out.",
        ),
      );
    });

    xhr.send(formData);
  });
}