// frontend/console/mintRequest/src/infrastructure/repository/http/productions.ts

import { API_BASE } from "../../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../../shared/http/authHeaders";

type ProductionListItemResponse = {
  ID: string;
  ProductBlueprintID: string;
};

type ProductionDetailResponse = {
  ID: string;
  ProductBlueprintID: string;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === "string" && value !== "";
}

function parseProductionListResponse(
  json: unknown,
): ProductionListItemResponse[] {
  if (!Array.isArray(json)) {
    throw new Error(
      "Invalid productions response: response is not an array",
    );
  }

  return json.map((item, index) => {
    if (!isRecord(item)) {
      throw new Error(
        `Invalid productions response: items[${index}] is not an object`,
      );
    }

    if (!isNonEmptyString(item.ID)) {
      throw new Error(
        `Invalid productions response: items[${index}].ID is missing`,
      );
    }

    if (!isNonEmptyString(item.ProductBlueprintID)) {
      throw new Error(
        `Invalid productions response: items[${index}].ProductBlueprintID is missing`,
      );
    }

    return {
      ID: item.ID,
      ProductBlueprintID: item.ProductBlueprintID,
    };
  });
}

function parseProductionDetailResponse(
  json: unknown,
): ProductionDetailResponse {
  if (!isRecord(json)) {
    throw new Error(
      "Invalid production response: response is not an object",
    );
  }

  if (!isNonEmptyString(json.ID)) {
    throw new Error(
      "Invalid production response: ID is missing",
    );
  }

  if (!isNonEmptyString(json.ProductBlueprintID)) {
    throw new Error(
      "Invalid production response: ProductBlueprintID is missing",
    );
  }

  return {
    ID: json.ID,
    ProductBlueprintID: json.ProductBlueprintID,
  };
}

/**
 * productionIdからproductBlueprintIdを解決する。
 *
 * GET /productions/{productionId}
 */
export async function fetchProductBlueprintIdByProductionIdHTTP(
  productionId: string,
): Promise<string | null> {
  const pid = String(productionId ?? "");

  if (!pid) {
    throw new Error("productionId が空です");
  }

  const authHeaders = await getAuthHeaders();
  const url = `${API_BASE}/productions/${encodeURIComponent(pid)}`;

  const response = await fetch(url, {
    method: "GET",
    headers: authHeaders,
  });

  if (!response.ok) {
    const body = await response.text().catch(() => "");

    throw new Error(
      `Failed to fetch production: ` +
        `${response.status} ${response.statusText}` +
        (body ? ` body=${body.slice(0, 400)}` : ""),
    );
  }

  const json = await response.json();
  const production = parseProductionDetailResponse(json);

  return production.ProductBlueprintID || null;
}

/**
 * 現在のcompanyのproductionsを取得し、
 * productionIdの配列を返す。
 *
 * 重複するproductionIdは除外する。
 */
export async function fetchProductionIdsForCurrentCompanyHTTP(): Promise<
  string[]
> {
  const authHeaders = await getAuthHeaders();
  const url = `${API_BASE}/productions`;

  const response = await fetch(url, {
    method: "GET",
    headers: authHeaders,
  });

  if (!response.ok) {
    const body = await response.text().catch(() => "");

    throw new Error(
      `Failed to fetch productions: ` +
        `${response.status} ${response.statusText}` +
        (body ? ` body=${body.slice(0, 400)}` : ""),
    );
  }

  const json = await response.json();
  const items = parseProductionListResponse(json);

  const ids: string[] = [];
  const seen = new Set<string>();

  for (const item of items) {
    const productionId = item.ID;

    if (seen.has(productionId)) {
      continue;
    }

    seen.add(productionId);
    ids.push(productionId);
  }

  return ids;
}