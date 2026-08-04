// frontend/amol/src/features/announcement/api/announcementApi.ts

import {
  HttpError,
  requestJson,
  requestVoid,
} from "../../../lib/http";
import { getOptionalAuthHeaders } from "../../../lib/authHeaders";

import type { AnnouncementListResult } from "../../shared/types/announcements";

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

  const headers = await getOptionalAuthHeaders();

  if (!headers) {
    return {
      items: [],
      totalCount: 0,
      page,
      perPage,
    };
  }

  try {
    const json = await requestJson<Partial<AnnouncementListResult>>(
      ANNOUNCEMENTS_ENDPOINT,
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
            "failed to fetch announcements",
          nonJsonErrorMessage:
            "failed to fetch announcements: response is not json",
        },
      },
    );

    return {
      items: Array.isArray(json.items) ? json.items : [],
      totalCount:
        typeof json.totalCount === "number" &&
        Number.isFinite(json.totalCount)
          ? json.totalCount
          : 0,
      page:
        typeof json.page === "number" &&
        Number.isFinite(json.page)
          ? json.page
          : page,
      perPage:
        typeof json.perPage === "number" &&
        Number.isFinite(json.perPage)
          ? json.perPage
          : perPage,
    };
  } catch (error) {
    if (
      error instanceof HttpError &&
      (error.status === 401 || error.status === 403)
    ) {
      return {
        items: [],
        totalCount: 0,
        page,
        perPage,
      };
    }

    throw error;
  }
}

export async function markMeAnnouncementRead(
  announcementId: string,
): Promise<void> {
  if (!announcementId) {
    return;
  }

  const headers = await getOptionalAuthHeaders();

  if (!headers) {
    return;
  }

  try {
    await requestVoid(
      `${ANNOUNCEMENTS_ENDPOINT}/${encodeURIComponent(
        announcementId,
      )}/read`,
      {
        method: "POST",
        headers,
        cache: "no-store",
      },
    );
  } catch (error) {
    if (
      error instanceof HttpError &&
      (error.status === 401 || error.status === 403)
    ) {
      return;
    }

    throw error;
  }
}