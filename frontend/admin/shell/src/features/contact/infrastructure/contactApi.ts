// frontend/admin/shell/src/features/contact/infrastructure/contactApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type {
  Contact,
  ContactListParams,
  ContactListResponse,
} from "../../../shared/type/contact";
import { appendPaginationParams } from "../../../shared/util/pagination";

const BACKEND_BASE_URL = import.meta.env.VITE_BACKEND_BASE_URL?.trim().replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }

  return BACKEND_BASE_URL;
}

async function requireOk(
  response: Response,
  message: string,
): Promise<void> {
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

export async function listContacts(
  params: ContactListParams = {},
): Promise<ContactListResponse> {
  const backendBaseUrl = requireBackendBaseUrl();
  const query = new URLSearchParams();

  appendPaginationParams(query, params.page, params.perPage);

  if (params.isRead !== undefined) {
    query.set("isRead", String(params.isRead));
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

export async function getContact(
  contactId: string,
): Promise<Contact> {
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