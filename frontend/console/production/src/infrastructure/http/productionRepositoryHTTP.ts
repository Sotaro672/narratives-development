// frontend/console/production/src/infrastructure/http/productionRepositoryHTTP.ts
import type { Production } from "../../application/create/ProductionCreateTypes";
import type { ProductionRepository } from "../../application/create/ProductionCreateRepository";

// ✅ shared を single source of truth にする
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";

// ✅ shared auth headers（shell の authService に委譲）
import { getAuthJsonHeadersOrThrow } from "../../../../shell/src/shared/http/authHeaders";

// ----------------------------------------------------------------------
// API ベース URL（末尾スラッシュ除去）
// ----------------------------------------------------------------------
function sanitizeBase(u: string): string {
  return (u || "").replace(/\/+$/g, "");
}

// ----------------------------------------------------------------------
// HTTP 実装（Adapter）
// ----------------------------------------------------------------------
// - Application Port: ProductionRepository を実装する
// - I/O の詳細（HTTP / Auth / BaseURL）をここで吸収する
export class ProductionRepositoryHTTP implements ProductionRepository {
  private readonly baseUrl: string;

  constructor(baseUrl: string = API_BASE) {
    this.baseUrl = sanitizeBase(baseUrl);
  }

  // 共通リクエストラッパー
  private async request<T>(path: string, init: RequestInit): Promise<T> {
    const url = `${this.baseUrl}${path}`;

    const method = (init.method ?? "GET").toUpperCase();

    const authHeaders = await getAuthJsonHeadersOrThrow();

    const res = await fetch(url, {
      ...init,
      headers: {
        ...authHeaders,
        ...(init.headers ?? {}),
      },
    });

    // DELETE 等の 204 → 空を返す
    if (res.status === 204) {
      return undefined as unknown as T;
    }

    if (!res.ok) {
      let bodyText = "";
      try {
        bodyText = await res.text();
      } catch {
        /* ignore */
      }

      // body が JSON なら {"error":"..."} を優先して読みやすくする
      let extracted = "";
      try {
        const obj = bodyText ? JSON.parse(bodyText) : null;
        if (obj && typeof obj.error === "string") extracted = obj.error;
      } catch {
        /* ignore */
      }

      console.error("[ProductionRepositoryHTTP] response error", {
        method,
        url,
        status: res.status,
        statusText: res.statusText,
        extracted,
        bodyHead: bodyText?.slice(0, 200),
      });

      // ★ hook 側で JSON 末尾を拾えるように、可能なら JSON を末尾に残す
      const suffix = bodyText ? `\n${bodyText}` : "";

      throw new Error(
        `Production API error: ${res.status} ${res.statusText}${
          extracted ? ` :: ${extracted}` : ""
        }${suffix}`,
      );
    }

    // 正常系: JSON を返す前提
    const json = (await res.json()) as T;
    return json;
  }

  // --------------------------------------------------------------------
  // create: POST /productions
  // --------------------------------------------------------------------
  async create(payload: Production): Promise<Production> {
    return this.request<Production>("/productions", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  }

  // --------------------------------------------------------------------
  // getById: GET /productions/{id}
  // --------------------------------------------------------------------
  async getById(id: string): Promise<Production> {
    const safeId = encodeURIComponent(id.trim());

    return this.request<Production>(`/productions/${safeId}`, {
      method: "GET",
    });
  }

  // --------------------------------------------------------------------
  // update: PUT /productions/{id}
  // --------------------------------------------------------------------
  async update(id: string, patch: Partial<Production>): Promise<Production> {
    const safeId = encodeURIComponent(id.trim());

    return this.request<Production>(`/productions/${safeId}`, {
      method: "PUT",
      body: JSON.stringify(patch),
    });
  }

  // --------------------------------------------------------------------
  // delete: DELETE /productions/{id}
  // --------------------------------------------------------------------
  async delete(id: string): Promise<void> {
    const safeId = encodeURIComponent(id.trim());

    await this.request<void>(`/productions/${safeId}`, {
      method: "DELETE",
    });
  }
}