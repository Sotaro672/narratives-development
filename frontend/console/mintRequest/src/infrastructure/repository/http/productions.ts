// frontend/console/mintRequest/src/infrastructure/repository/http/productions.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";
import {
  logHttpRequest,
  logHttpResponse,
  safeTokenHint,
} from "../../http/httpLogger";

type ProductionListItemResponse = {
  ID: string;
  ProductBlueprintID: string;
};

type ProductionDetailResponse = {
  ID: string;
  ProductBlueprintID: string;
};

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null;
}

function isNonEmptyString(v: unknown): v is string {
  return typeof v === "string" && v.trim() !== "";
}

function parseProductionListResponse(json: unknown): ProductionListItemResponse[] {
  if (!Array.isArray(json)) {
    throw new Error("Invalid productions response: response is not an array");
  }

  return json.map((item, index) => {
    if (!isRecord(item)) {
      throw new Error(`Invalid productions response: items[${index}] is not an object`);
    }

    if (!isNonEmptyString(item.ID)) {
      throw new Error(`Invalid productions response: items[${index}].ID is missing`);
    }

    if (!isNonEmptyString(item.ProductBlueprintID)) {
      throw new Error(
        `Invalid productions response: items[${index}].ProductBlueprintID is missing`,
      );
    }

    return {
      ID: item.ID.trim(),
      ProductBlueprintID: item.ProductBlueprintID.trim(),
    };
  });
}

function parseProductionDetailResponse(json: unknown): ProductionDetailResponse {
  if (!isRecord(json)) {
    throw new Error("Invalid production response: response is not an object");
  }

  if (!isNonEmptyString(json.ID)) {
    throw new Error("Invalid production response: ID is missing");
  }

  if (!isNonEmptyString(json.ProductBlueprintID)) {
    throw new Error("Invalid production response: ProductBlueprintID is missing");
  }

  return {
    ID: json.ID.trim(),
    ProductBlueprintID: json.ProductBlueprintID.trim(),
  };
}

function getAuthValueOrThrow(authHeaders: Record<string, string>): string {
  const authValue = String((authHeaders as any)?.Authorization ?? "").trim();
  if (!authValue) {
    throw new Error("Authorization header is missing (not logged in or token unavailable)");
  }
  return authValue;
}

function extractIdTokenForLog(authValue: string): string {
  const m = String(authValue ?? "").match(/^Bearer\s+(.+)$/i);
  return String(m?.[1] ?? "").trim();
}

/**
 * productionId から productBlueprintId を解決する
 * - GET /productions/{productionId}
 */
export async function fetchProductBlueprintIdByProductionIdHTTP(
  productionId: string,
): Promise<string | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) throw new Error("productionId が空です");

  const authHeaders = await getAuthHeadersOrThrow();
  const authValue = getAuthValueOrThrow(authHeaders);
  const idToken = extractIdTokenForLog(authValue);

  const url = `${API_BASE}/productions/${encodeURIComponent(pid)}`;

  logHttpRequest("fetchProductBlueprintIdByProductionIdHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
      "Content-Type": "application/json",
    },
  });

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  logHttpResponse("fetchProductBlueprintIdByProductionIdHTTP", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch production: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = await res.json();
  const production = parseProductionDetailResponse(json);

  return production.ProductBlueprintID || null;
}

/**
 * 現在の company の productions を取得し、productionId の配列を返す（重複除去）
 */
export async function fetchProductionIdsForCurrentCompanyHTTP(): Promise<string[]> {
  const authHeaders = await getAuthHeadersOrThrow();
  const authValue = getAuthValueOrThrow(authHeaders);
  const idToken = extractIdTokenForLog(authValue);

  const url = `${API_BASE}/productions`;

  logHttpRequest("fetchProductionIdsForCurrentCompanyHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
      "Content-Type": "application/json",
    },
  });

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  logHttpResponse("fetchProductionIdsForCurrentCompanyHTTP", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch productions: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = await res.json();
  const items = parseProductionListResponse(json);

  const ids: string[] = [];
  const seen = new Set<string>();

  for (const it of items) {
    const pid = it.ID;
    if (seen.has(pid)) continue;

    seen.add(pid);
    ids.push(pid);
  }

  return ids;
}