// frontend/console/shell/src/features/production/infrastructure/http/productionRepositoryHTTP.ts

import type { Production } from "../../../../shared/types/production";
import { API_BASE } from "../../../../shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../shared/http/authHeaders";
import type {
  CreateProductionRequest,
  ProductionRepository,
} from "../../application/create/ProductionCreateService";

export class ProductionRepositoryHTTP implements ProductionRepository {
  private readonly baseUrl: string;

  constructor(baseUrl: string = API_BASE) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(
    path: string,
    init: RequestInit,
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const authHeaders = await getAuthJsonHeadersOrThrow();

    const response = await fetch(url, {
      ...init,
      headers: {
        ...authHeaders,
        ...(init.headers ?? {}),
      },
    });

    if (response.status === 204) {
      return undefined as T;
    }

    if (!response.ok) {
      const bodyText = await response.text().catch(() => "");
      let extracted = "";

      try {
        const body: unknown = bodyText ? JSON.parse(bodyText) : null;

        if (
          body !== null &&
          typeof body === "object" &&
          !Array.isArray(body) &&
          "error" in body &&
          typeof body.error === "string"
        ) {
          extracted = body.error;
        }
      } catch {
        // JSON以外のレスポンス本文はbodyTextをそのままエラーへ含める。
      }

      const suffix = bodyText ? `\n${bodyText}` : "";

      throw new Error(
        `Production API error: ${response.status} ${response.statusText}${
          extracted ? ` :: ${extracted}` : ""
        }${suffix}`,
      );
    }

    return (await response.json()) as T;
  }

  // --------------------------------------------------------------------
  // POST /productions
  // --------------------------------------------------------------------

  async create(
    payload: CreateProductionRequest,
  ): Promise<unknown> {
    return this.request<unknown>("/productions", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  }

  // --------------------------------------------------------------------
  // GET /productions/{id}
  // --------------------------------------------------------------------

  async getById(
    id: string,
  ): Promise<Production> {
    const safeId = encodeURIComponent(id.trim());

    return this.request<Production>(`/productions/${safeId}`, {
      method: "GET",
    });
  }

  // --------------------------------------------------------------------
  // PUT /productions/{id}
  // --------------------------------------------------------------------

  async update(
    id: string,
    patch: Partial<Production>,
  ): Promise<Production> {
    const safeId = encodeURIComponent(id.trim());

    return this.request<Production>(`/productions/${safeId}`, {
      method: "PUT",
      body: JSON.stringify(patch),
    });
  }

  // --------------------------------------------------------------------
  // DELETE /productions/{id}
  // --------------------------------------------------------------------

  async delete(
    id: string,
  ): Promise<void> {
    const safeId = encodeURIComponent(id.trim());

    await this.request<void>(`/productions/${safeId}`, {
      method: "DELETE",
    });
  }
}