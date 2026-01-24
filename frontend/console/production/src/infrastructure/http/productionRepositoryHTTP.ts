// frontend/console/production/src/infrastructure/http/productionRepositoryHTTP.ts
/// <reference types="vite/client" />

import type { Production } from "../../application/create/ProductionCreateTypes";
import type { ProductionRepository } from "../../application/create/ProductionCreateRepository";
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ----------------------------------------------------------------------
// API ベース URL（末尾スラッシュ除去）
// ----------------------------------------------------------------------
const RAW_ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

function sanitizeBase(u: string): string {
  return (u || "").replace(/\/+$/g, "");
}

const ENV_BASE = sanitizeBase(RAW_ENV_BASE);
export const API_BASE = sanitizeBase(ENV_BASE || FALLBACK_BASE);

// -----------------------------------------------------------
// 共通: Firebase 認証トークン取得
// -----------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");

  // ★ companyId などの claims 更新が絡む場合に備えて、必要に応じて強制リフレッシュ
  // （毎回 true だと少し重いので、まずは false。問題が続くなら true にして確認）
  return user.getIdToken(false);
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
    const idToken = await getIdTokenOrThrow();
    const url = `${this.baseUrl}${path}`;

    const method = (init.method ?? "GET").toUpperCase();
    console.log("[ProductionRepositoryHTTP] request", { method, url });

    const res = await fetch(url, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${idToken}`,
        ...(init.headers ?? {}),
      },
    });

    // DELETE 等の 204 → 空を返す
    if (res.status === 204) {
      console.log("[ProductionRepositoryHTTP] response", {
        method,
        url,
        status: res.status,
      });
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
    console.log("[ProductionRepositoryHTTP] response", {
      method,
      url,
      status: res.status,
    });
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

  // --------------------------------------------------------------------
  // markProductBlueprintPrinted:
  //   POST /product-blueprints/{productBlueprintId}/mark-printed
  // --------------------------------------------------------------------
  async markProductBlueprintPrinted(productBlueprintId: string): Promise<void> {
    const safeId = encodeURIComponent(productBlueprintId.trim());

    await this.request<void>(`/product-blueprints/${safeId}/mark-printed`, {
      method: "POST",
    });
  }
}
