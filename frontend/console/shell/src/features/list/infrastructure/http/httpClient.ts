// frontend/console/shell/src/features/list/infrastructure/http/httpClient.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { getAuthJsonHeaders } from "../../../../shared/http/authHeaders";
import { fetchJSON, HttpError } from "../../../../shared/http/fetchJSON";

export async function requestJSON<T>(args: {
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  path: string;
  body?: unknown;
}): Promise<T> {
  const url = `${API_BASE}${args.path.startsWith("/") ? "" : "/"}${args.path}`;

  let bodyText: string | undefined = undefined;
  if (args.body !== undefined) {
    try {
      bodyText = JSON.stringify(args.body);
    } catch {
      throw new Error("invalid_json_stringify");
    }
  }

  const headers = await getAuthJsonHeaders();

  try {
    return await fetchJSON<T>(url, {
      method: args.method,
      headers,
      body: bodyText,
    });
  } catch (e) {
    if (e instanceof HttpError) {
      let json: any = null;
      try {
        json = e.bodyText ? JSON.parse(e.bodyText) : null;
      } catch {
        json = { raw: e.bodyText ?? "" };
      }

      const msg =
        (json && typeof json === "object" && (json.error || json.message)) ||
        `http_error_${e.status}`;

      throw new Error(String(msg));
    }

    throw e;
  }
}