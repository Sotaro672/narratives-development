// frontend/amol/src/features/announcement/api/announcementApi.ts
import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";

import type { AnnouncementListResult } from "../types";

const ANNOUNCEMENTS_ENDPOINT = "/mall/me/announcement";

type FetchAnnouncementsParams = {
  page?: number;
  perPage?: number;
  signal?: AbortSignal;
};

export async function fetchMeAnnouncements(
  params: FetchAnnouncementsParams = {},
): Promise<AnnouncementListResult> {
  const page = params.page ?? 1;
  const perPage = params.perPage ?? 100;

  const token = await getOptionalFirebaseIdToken();

  if (!token) {
    return {
      items: [],
      totalCount: 0,
      page,
      perPage,
    };
  }

  const searchParams = new URLSearchParams({
    page: String(page),
    perPage: String(perPage),
  });

  const response = await fetch(
    `${apiUrl(ANNOUNCEMENTS_ENDPOINT)}?${searchParams}`,
    {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: "application/json",
      },
      signal: params.signal,
      cache: "no-store",
    },
  );

  if (response.status === 401 || response.status === 403) {
    return {
      items: [],
      totalCount: 0,
      page,
      perPage,
    };
  }

  if (!response.ok) {
    throw new Error(`failed to fetch announcements: ${response.status}`);
  }

  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.includes("application/json")) {
    throw new Error("failed to fetch announcements: response is not json");
  }

  const json = (await response.json()) as Partial<AnnouncementListResult>;

  return {
    items: Array.isArray(json.items) ? json.items : [],
    totalCount:
      typeof json.totalCount === "number" && Number.isFinite(json.totalCount)
        ? json.totalCount
        : 0,
    page:
      typeof json.page === "number" && Number.isFinite(json.page)
        ? json.page
        : page,
    perPage:
      typeof json.perPage === "number" && Number.isFinite(json.perPage)
        ? json.perPage
        : perPage,
  };
}

export async function markMeAnnouncementRead(
  announcementId: string,
): Promise<void> {
  if (!announcementId) {
    return;
  }

  const token = await getOptionalFirebaseIdToken();

  if (!token) {
    return;
  }

  const response = await fetch(
    apiUrl(`${ANNOUNCEMENTS_ENDPOINT}/${announcementId}/read`),
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: "application/json",
      },
      cache: "no-store",
    },
  );

  if (response.status === 401 || response.status === 403) {
    return;
  }

  if (!response.ok) {
    throw new Error(`failed to mark announcement read: ${response.status}`);
  }
}

function apiUrl(path: string): string {
  const baseUrl = getApiBaseUrl();

  if (!baseUrl) {
    return path;
  }

  return `${baseUrl}${path}`;
}

async function getOptionalFirebaseIdToken(): Promise<string | null> {
  try {
    return await getFirebaseIdToken();
  } catch {
    return null;
  }
}