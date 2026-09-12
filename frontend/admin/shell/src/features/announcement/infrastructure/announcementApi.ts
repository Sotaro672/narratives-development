// frontend/admin/shell/src/features/announcement/infrastructure/announcementApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";

export type AnnouncementAttachmentFile = {
  announcementId: string;
  id: string;
  fileName: string;
  fileUrl: string;
  fileSize: number;
  mimeType: string;
  objectPath: string;
};

export type AnnouncementDetail = {
  id: string;
  title: string;
  content: string;
  targetToken?: string;
  tokenName: string;
  targetAvatars?: string[];
  published: boolean;
  publishedAt?: string;
  attachments?: string[];
  attachmentFiles?: AnnouncementAttachmentFile[];
  createdAt: string;
  createdBy: string;
  createdByName: string;
  updatedAt?: string;
  updatedBy?: string;
  updatedByName?: string;
};

const BACKEND_BASE_URL =
  import.meta.env.VITE_BACKEND_BASE_URL?.trim().replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }

  return BACKEND_BASE_URL;
}

function requireID(value: string, name: string): string {
  const normalized = value.trim();

  if (!normalized) {
    throw new Error(`${name} is required.`);
  }

  return normalized;
}

async function requireOk(response: Response, message: string): Promise<void> {
  if (response.ok) return;

  let detail = "";

  try {
    const body = (await response.json()) as { error?: string };
    detail = body.error ? ` error=${body.error}` : "";
  } catch {
    // Response body may not be JSON.
  }

  throw new Error(`${message} status=${response.status}${detail}`);
}

async function getAdminJSON<T>(
  path: string,
  errorMessage: string,
): Promise<T> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(`${backendBaseUrl}${path}`, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  await requireOk(response, errorMessage);

  return (await response.json()) as T;
}

export async function getAnnouncementDetail(
  companyId: string,
  announcementId: string,
): Promise<AnnouncementDetail> {
  const normalizedCompanyId = requireID(companyId, "companyId");
  const normalizedAnnouncementId = requireID(
    announcementId,
    "announcementId",
  );

  return getAdminJSON<AnnouncementDetail>(
    `/admin/companies/${encodeURIComponent(normalizedCompanyId)}/announcements/${encodeURIComponent(normalizedAnnouncementId)}`,
    "Failed to load announcement detail.",
  );
}