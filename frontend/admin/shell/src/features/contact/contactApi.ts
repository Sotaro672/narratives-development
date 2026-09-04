// frontend/admin/shell/src/features/contact/contactApi.ts

import {
  getAuthHeaders,
} from "../../shared/http/authHeaders";

export type ContactStatus = "new";

export type Contact = {
  id: string;
  name: string;
  email: string;
  company: string;
  message: string;
  status: ContactStatus;
  source: string;
  createdAt: string;
};

export type ContactListResponse = {
  items: Contact[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type ContactListParams = {
  page?: number;
  perPage?: number;
  status?: ContactStatus;
  sort?: string;
  order?: "asc" | "desc";
};

const BACKEND_BASE_URL =
  import.meta.env.VITE_BACKEND_BASE_URL
    ?.trim()
    .replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error(
      "VITE_BACKEND_BASE_URL is not configured.",
    );
  }

  return BACKEND_BASE_URL;
}

function appendPositiveInteger(
  params: URLSearchParams,
  key: string,
  value: number | undefined,
): void {
  if (
    value === undefined ||
    !Number.isInteger(value) ||
    value <= 0
  ) {
    return;
  }

  params.set(
    key,
    String(value),
  );
}

export async function listContacts(
  params: ContactListParams = {},
): Promise<ContactListResponse> {
  const backendBaseUrl =
    requireBackendBaseUrl();

  const query =
    new URLSearchParams();

  appendPositiveInteger(
    query,
    "page",
    params.page,
  );

  appendPositiveInteger(
    query,
    "perPage",
    params.perPage,
  );

  if (params.status) {
    query.set(
      "status",
      params.status,
    );
  }

  if (params.sort?.trim()) {
    query.set(
      "sort",
      params.sort.trim(),
    );
  }

  if (params.order) {
    query.set(
      "order",
      params.order,
    );
  }

  const authHeaders =
    await getAuthHeaders();

  const queryString =
    query.toString();

  const url =
    `${backendBaseUrl}/admin/contacts` +
    (queryString
      ? `?${queryString}`
      : "");

  const response =
    await fetch(
      url,
      {
        method: "GET",
        headers: {
          ...authHeaders,
          Accept: "application/json",
        },
      },
    );

  if (!response.ok) {
    throw new Error(
      `Failed to load contacts. status=${response.status}`,
    );
  }

  return response.json() as Promise<ContactListResponse>;
}