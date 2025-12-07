// frontend/console/production/src/infrastructure/http/productionRepositoryHTTP.ts
/// <reference types="vite/client" />

import type { Production } from "../../application/productionCreateService";
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ----------------------------------------------------------------------
// API ベース URL（productBlueprintRepositoryHTTP と同じ構成）
// ----------------------------------------------------------------------
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)
    ?.replace(/\/+$/g, "") ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// -----------------------------------------------------------
// 共通: Firebase 認証トークン取得
// -----------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");
  return user.getIdToken();
}

// ----------------------------------------------------------------------
// Repository インターフェース
// ----------------------------------------------------------------------
export interface ProductionRepository {
  /** 新規作成 */
  create(payload: Production): Promise<Production>;

  /** 単一取得 */
  getById(id: string): Promise<Production>;

  /** 更新（部分更新） */
  update(id: string, patch: Partial<Production>): Promise<Production>;

  /** 削除 */
  delete(id: string): Promise<void>;

  /**
   * 対応する商品設計(ProductBlueprint)の printed フラグを
   * notYet → printed に更新する
   * - backend: POST /product-blueprints/{id}/mark-printed
   */
  markProductBlueprintPrinted(productBlueprintId: string): Promise<void>;
}

// ----------------------------------------------------------------------
// HTTP 実装
// ----------------------------------------------------------------------
export class ProductionRepositoryHTTP implements ProductionRepository {
  private readonly baseUrl: string;

  constructor(baseUrl: string = API_BASE) {
    this.baseUrl = baseUrl;
  }

  // 共通リクエストラッパー
  private async request<T>(path: string, init: RequestInit): Promise<T> {
    const idToken = await getIdTokenOrThrow();
    const url = `${this.baseUrl}${path}`;

    const res = await fetch(url, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${idToken}`,
        ...(init.headers ?? {}),
      },
    });

    if (!res.ok) {
      let bodyText = "";
      try {
        bodyText = await res.text();
      } catch {
        /* ignore */
      }
      throw new Error(
        `Production API error: ${res.status} ${res.statusText}${
          bodyText ? ` - ${bodyText}` : ""
        }`,
      );
    }

    // DELETE 等の 204 → 空を返す
    if (res.status === 204) {
      return undefined as unknown as T;
    }

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
  // delete: DELETE /productions/{id}
  // --------------------------------------------------------------------
  async delete(id: string): Promise<void> {
    const safeId = encodeURIComponent(id.trim());

    await this.request<void>(`/productions/${safeId}`, {
      method: "DELETE",
    });
  }

  // --------------------------------------------------------------------
  // markProductBlueprintPrinted:
  //   POST /product-blueprints/{productBlueprintId}/mark-printed
  // --------------------------------------------------------------------
  async markProductBlueprintPrinted(productBlueprintId: string): Promise<void> {
    const safeId = encodeURIComponent(productBlueprintId.trim());

    await this.request<void>(
      `/product-blueprints/${safeId}/mark-printed`,
      {
        method: "POST",
      },
    );
  }
}
