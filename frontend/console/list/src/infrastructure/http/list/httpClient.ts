// frontend/console/list/src/infrastructure/http/list/httpClient.ts
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
    body?: unknown; // 呼び出し側が入れても良いが、送信bodyとズレることがある
  };
}): Promise<T> {
  const token = await getIdToken();
  const url = `${API_BASE}${args.path.startsWith("/") ? "" : "/"}${args.path}`;

  let bodyText: string | undefined = undefined;
  if (args.body !== undefined) {
    try {
      bodyText = JSON.stringify(args.body);
    } catch {
      throw new Error("invalid_json_stringify");
    }
  }

  // ✅ debug: “実際に送る args.body / bodyText” を必ずログする
  if (args.debug) {
    try {
      const actualBody = args.body !== undefined ? args.body : args.debug.body;
      const actualBodyStr = actualBody === undefined ? undefined : JSON.stringify(actualBody);

      // objectPath の存在チェック（ネストは追わずトップだけ）
      const bodyObj = actualBody && typeof actualBody === "object" ? (actualBody as any) : null;
      const hasObjectPath =
        !!bodyObj && Object.prototype.hasOwnProperty.call(bodyObj, "objectPath");
      const hasObjectPathSnake =
        !!bodyObj && Object.prototype.hasOwnProperty.call(bodyObj, "object_path");

      console.log(`[list/listRepositoryHTTP] ${args.debug.tag}`, {
        method: args.debug.method,
        url: args.debug.url,

        // ✅ 実際に送るbodyを表示
        body: actualBody,
        bodyJSON: actualBodyStr,

        // ✅ fetchに渡す“文字列”も表示（長いので先頭だけ）
        bodyTextLen: bodyText ? bodyText.length : 0,
        bodyTextHead: bodyText ? bodyText.slice(0, 200) : undefined,

        hasObjectPath,
        hasObjectPathSnake,
        objectPath: bodyObj ? bodyObj.objectPath : undefined,
        object_path: bodyObj ? bodyObj.object_path : undefined,
      });
    } catch (e) {
      console.log(`[list/listRepositoryHTTP] ${args.debug.tag} (stringify_failed)`, {
        method: args.debug.method,
        url: args.debug.url,
        err: String(e),
      });
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
