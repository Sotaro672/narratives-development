//frontend\console\list\src\infrastructure\http\list\httpClient.ts
import { API_BASE } from "./config";
import { getIdToken } from "./authToken";

export async function requestJSON<T>(args: {
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  path: string;
  body?: unknown;

  // ✅ debug log
  debug?: {
    tag: string;
    url: string;
    method: string;
    body?: unknown;
  };
}): Promise<T> {
  const token = await getIdToken();
  const url = `${API_BASE}${args.path.startsWith("/") ? "" : "/"}${args.path}`;

  if (args.debug) {
    try {
      const bodyStr =
        args.debug.body === undefined ? undefined : JSON.stringify(args.debug.body);
      console.log(`[list/listRepositoryHTTP] ${args.debug.tag}`, {
        method: args.debug.method,
        url: args.debug.url,
        body: args.debug.body,
        bodyJSON: bodyStr,
      });
    } catch (e) {
      console.log(`[list/listRepositoryHTTP] ${args.debug.tag} (stringify_failed)`, {
        method: args.debug.method,
        url: args.debug.url,
        body: args.debug.body,
        err: String(e),
      });
    }
  }

  let bodyText: string | undefined = undefined;
  if (args.body !== undefined) {
    try {
      bodyText = JSON.stringify(args.body);
    } catch {
      throw new Error("invalid_json_stringify");
    }
  }

  const res = await fetch(url, {
    method: args.method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: bodyText,
  });

  const text = await res.text();
  let json: any = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = { raw: text };
  }

  if (!res.ok) {
    const msg =
      (json && typeof json === "object" && (json.error || json.message)) ||
      `http_error_${res.status}`;

    console.log(`[list/listRepositoryHTTP] response error`, {
      method: args.method,
      url,
      status: res.status,
      raw: text,
      json,
    });

    throw new Error(String(msg));
  }

  return json as T;
}
