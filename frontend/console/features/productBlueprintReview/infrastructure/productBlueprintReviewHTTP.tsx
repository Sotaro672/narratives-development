// frontend/console/productBlueprintReview/src/infrastructure/productBlueprintReviewHTTP.tsx

import { API_BASE } from "../../../shell/src/shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../shell/src/shared/http/authHeaders";

import type {
  ListCompanyReviewAggregatesParams,
  ListCompanyReviewAggregatesResponse,
  ListProductBlueprintReviewsParams,
  ListProductBlueprintReviewsResponse,
} from "../domain/entity";

// ==============================
// Query builder (PascalCase keys)
// ==============================

function BuildQuery(Params?: Record<string, unknown>): string {
  const Sp = new URLSearchParams();
  if (!Params) return "";

  for (const [K, V] of Object.entries(Params)) {
    if (V === undefined || V === null) continue;
    if (typeof V === "string" && !V.trim()) continue;
    Sp.set(K, String(V));
  }

  const Qs = Sp.toString();
  return Qs ? `?${Qs}` : "";
}

// ==============================
// HTTP core
// ==============================

async function HttpGetJSON<T>(Url: string): Promise<T> {
  const Headers = await getAuthJsonHeadersOrThrow();

  const Res = await fetch(Url, {
    method: "GET",
    headers: {
      ...Headers,
      Accept: "application/json",
    },
    credentials: "include",
  });

  const Ct = Res.headers.get("content-type") || "";
  if (!Ct.includes("application/json")) {
    const T = await Res.text().catch(() => "");
    throw new Error(
      `Expected JSON but got "${Ct}". URL=${Url}. Body(head)=${T.slice(0, 200)}`
    );
  }

  const Data = (await Res.json().catch(() => null)) as any;

  if (!Res.ok) {
    const Msg = Data?.Error || JSON.stringify(Data);
    throw new Error(Msg || `HTTP ${Res.status}`);
  }

  return Data as T;
}

// ==============================
// Client
// ==============================

export class ProductBlueprintReviewHTTP {
  private readonly BaseURL: string;

  constructor(BaseURL?: string) {
    this.BaseURL = (BaseURL ?? API_BASE).replace(/\/+$/, "");
  }

  /**
   * Detail: GET /product-blueprint-reviews?ProductBlueprintID=...
   * Query: ProductBlueprintID (required), Status, Page, PerPage
   */
  async ListReviewsByProductBlueprintID(
    Params: ListProductBlueprintReviewsParams
  ): Promise<ListProductBlueprintReviewsResponse> {
    if (!Params?.ProductBlueprintID?.trim()) {
      throw new Error("ProductBlueprintID is required");
    }
    const Path = `/product-blueprint-reviews${BuildQuery(Params)}`;
    const Url = `${this.BaseURL}${Path}`;
    return await HttpGetJSON<ListProductBlueprintReviewsResponse>(Url);
  }

  /**
   * Management: GET /product-blueprint-reviews/aggregates
   * Query: Status, Page, PerPage
   */
  async ListCompanyReviewAggregates(
    Params?: ListCompanyReviewAggregatesParams
  ): Promise<ListCompanyReviewAggregatesResponse> {
    const Path = `/product-blueprint-reviews/aggregates${BuildQuery(Params)}`;
    const Url = `${this.BaseURL}${Path}`;
    return await HttpGetJSON<ListCompanyReviewAggregatesResponse>(Url);
  }
}

export const ProductBlueprintReviewHTTPClient = new ProductBlueprintReviewHTTP();
export const productBlueprintReviewHTTP = ProductBlueprintReviewHTTPClient;