// frontend/mall/src/features/news/api/newsApi.ts

import { HttpError, requestJson } from "../../../lib/http";
import { getOptionalAuthHeaders } from "../../../lib/authHeaders";

const NEWS_ENDPOINT = "/mall/me/news";

export type News = {
  id: string;
  title: string;
  body: string;
  publishedAt: string;
  createdAt: string;
  isRead: boolean;
  readAt: string | null;
};

export type NewsPage = {
  items: News[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type NewsUnreadCountResponse = {
  unreadCount: number;
};

export type NewsReadResponse = {
  id: string;
  newsId: string;
  isRead: boolean;
  readAt: string;
};

export type FetchMeNewsParams = {
  page?: number;
  perPage?: number;
  signal?: AbortSignal;
};

function createEmptyPage(
  page: number,
  perPage: number,
): NewsPage {
  return {
    items: [],
    totalCount: 0,
    totalPages: 0,
    page,
    perPage,
  };
}

function normalizeFiniteNumber(
  value: unknown,
  fallback: number,
): number {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : fallback;
}

export async function fetchMeNews(
  params: FetchMeNewsParams = {},
): Promise<NewsPage> {
  const page = params.page ?? 1;
  const perPage = params.perPage ?? 20;

  const headers = await getOptionalAuthHeaders();

  if (!headers) {
    return createEmptyPage(page, perPage);
  }

  try {
    const json = await requestJson<Partial<NewsPage>>(
      NEWS_ENDPOINT,
      {
        method: "GET",
        headers,
        query: {
          page,
          perPage,
        },
        signal: params.signal,
        cache: "no-store",
        messages: {
          requestErrorMessage:
            "failed to fetch system news",
          nonJsonErrorMessage:
            "failed to fetch system news: response is not json",
          invalidJsonErrorMessage:
            "failed to fetch system news: invalid json",
        },
      },
    );

    return {
      items: Array.isArray(json.items) ? json.items : [],
      totalCount: Math.max(
        0,
        normalizeFiniteNumber(json.totalCount, 0),
      ),
      totalPages: Math.max(
        0,
        normalizeFiniteNumber(json.totalPages, 0),
      ),
      page: Math.max(
        1,
        normalizeFiniteNumber(json.page, page),
      ),
      perPage: Math.max(
        1,
        normalizeFiniteNumber(json.perPage, perPage),
      ),
    };
  } catch (error) {
    if (
      error instanceof HttpError &&
      (error.status === 401 || error.status === 403)
    ) {
      return createEmptyPage(page, perPage);
    }

    throw error;
  }
}

export async function fetchMeNewsUnreadCount(
  signal?: AbortSignal,
): Promise<NewsUnreadCountResponse> {
  const headers = await getOptionalAuthHeaders();

  if (!headers) {
    return {
      unreadCount: 0,
    };
  }

  try {
    const json = await requestJson<
      Partial<NewsUnreadCountResponse>
    >(
      `${NEWS_ENDPOINT}/unread-count`,
      {
        method: "GET",
        headers,
        signal,
        cache: "no-store",
        messages: {
          requestErrorMessage:
            "failed to fetch system news unread count",
          nonJsonErrorMessage:
            "failed to fetch system news unread count: response is not json",
          invalidJsonErrorMessage:
            "failed to fetch system news unread count: invalid json",
        },
      },
    );

    return {
      unreadCount: Math.max(
        0,
        normalizeFiniteNumber(json.unreadCount, 0),
      ),
    };
  } catch (error) {
    if (
      error instanceof HttpError &&
      (error.status === 401 || error.status === 403)
    ) {
      return {
        unreadCount: 0,
      };
    }

    throw error;
  }
}

export async function markMeNewsRead(
  newsId: string,
): Promise<NewsReadResponse> {
  const normalizedNewsId = newsId.trim();

  if (!normalizedNewsId) {
    throw new Error("newsId is required");
  }

  const headers = await getOptionalAuthHeaders();

  if (!headers) {
    throw new Error("authentication is required");
  }

  return requestJson<NewsReadResponse>(
    `${NEWS_ENDPOINT}/${encodeURIComponent(
      normalizedNewsId,
    )}/read`,
    {
      method: "POST",
      headers,
      cache: "no-store",
      messages: {
        requestErrorMessage:
          "failed to mark system news as read",
        nonJsonErrorMessage:
          "failed to mark system news as read: response is not json",
        invalidJsonErrorMessage:
          "failed to mark system news as read: invalid json",
      },
    },
  );
}

export const newsApi = {
  list: fetchMeNews,
  unreadCount: fetchMeNewsUnreadCount,
  markRead: markMeNewsRead,
};