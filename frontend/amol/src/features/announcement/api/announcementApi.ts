// frontend/amol/src/features/announcement/api/announcementApi.ts
import { getAuth } from "firebase/auth";

import type { AnnouncementListResult } from "../types";

const ANNOUNCEMENTS_ENDPOINT = "/mall/me/announcements";

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

  const token = await getCurrentUserIdToken();

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

  const response = await fetch(`${ANNOUNCEMENTS_ENDPOINT}?${searchParams}`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
    signal: params.signal,
  });

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

  const token = await getCurrentUserIdToken();

  if (!token) {
    return;
  }

  const response = await fetch(
    `${ANNOUNCEMENTS_ENDPOINT}/${announcementId}/read`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  );

  if (response.status === 401 || response.status === 403) {
    return;
  }

  if (!response.ok) {
    throw new Error(`failed to mark announcement read: ${response.status}`);
  }
}

async function getCurrentUserIdToken(): Promise<string | null> {
  const auth = getAuth();
  const user = auth.currentUser;

  if (!user) {
    return null;
  }

  return user.getIdToken();
}