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
): Promise<News> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(
    `${backendBaseUrl}/admin/news`,
    {
      method: "POST",
      headers: {
        ...authHeaders,
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        title: input.title.trim(),
        body: input.body.trim(),
      }),
    },
  );

  await requireOk(
    response,
    "Failed to publish news.",
  );

  return response.json() as Promise<News>;
}