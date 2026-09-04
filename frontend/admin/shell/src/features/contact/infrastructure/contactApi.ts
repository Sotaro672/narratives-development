// frontend/admin/shell/src/features/contact/infrastructure/contactApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";

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

const BACKEND_BASE_URL = import.meta.env.VITE_BACKEND_BASE_URL?.trim().replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }
  return BACKEND_BASE_URL;
}

function appendPositiveInteger(params: URLSearchParams, key: string, value: number | undefined): void {
  if (value === undefined || !Number.isInteger(value) || value <= 0) {
    return;
  }
  params.set(key, String(value));
}

async function requireOk(response: Response, message: string): Promise<void> {
  if (response.ok) {
    return;
  }

  let detail = "";
  try {
    const body = await response.json() as { error?: string };
    detail = body.error ? ` error=${body.error}` : "";
  } catch {
    // Response body may not be JSON.
  }

  throw new Error(`${message} status=${response.status}${detail}`);
}

export async function listContacts(params: ContactListParams = {}): Promise<ContactListResponse> {
  const backendBaseUrl = requireBackendBaseUrl();
  const query = new URLSearchParams();

  appendPositiveInteger(query, "page", params.page);
  appendPositiveInteger(query, "perPage", params.perPage);

  if (params.status) {
    query.set("status", params.status);
  }
  if (params.sort?.trim()) {
    query.set("sort", params.sort.trim());
  }
  if (params.order) {
    query.set("order", params.order);
  }

  const authHeaders = await getAuthHeaders();
  const queryString = query.toString();
  const url = `${backendBaseUrl}/admin/contacts${queryString ? `?${queryString}` : ""}`;

  const response = await fetch(url, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  await requireOk(response, "Failed to load contacts.");
  return response.json() as Promise<ContactListResponse>;
}

export async function getContact(contactId: string): Promise<Contact> {
  const trimmedContactId = contactId.trim();
  if (!trimmedContactId) {
    throw new Error("contactId is required.");
  }

  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(
    `${backendBaseUrl}/admin/contacts/${encodeURIComponent(trimmedContactId)}`,
    {
      method: "GET",
      headers: {
        ...authHeaders,
        Accept: "application/json",
      },
    },
  );

  await requireOk(response, "Failed to load contact.");
  return response.json() as Promise<Contact>;
}