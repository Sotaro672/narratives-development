// frontend/console/shell/src/features/productBlueprintReview/infrastructure/productBlueprintReviewHTTP.tsx

import { API_BASE } from "../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../shared/http/authHeaders";

import type {
  ListCompanyReviewAggregatesParams,
  ListCompanyReviewAggregatesResponse,
  ListProductBlueprintReviewsParams,
  ListProductBlueprintReviewsResponse,
} from "../../../shared/types/productBlueprintReview";
import type {
  ReportProductBlueprintReviewInput,
  ReviewReportRequest,
  ReviewReportResponse,
} from "../../../shared/types/reviewReport";
import { requiresReviewReportDetail } from "../../../shared/types/reviewReport";

// ==============================
// Query builder (PascalCase keys)
// ==============================

function BuildQuery(Params?: Record<string, unknown>): string {
  const Sp = new URLSearchParams();

  if (!Params) {
    return "";
  }

  for (const [K, V] of Object.entries(Params)) {
    if (V === undefined || V === null) {
      continue;
    }

    if (typeof V === "string" && !V.trim()) {
      continue;
    }

    Sp.set(K, String(V));
  }

  const Qs = Sp.toString();
  return Qs ? `?${Qs}` : "";
}

// ==============================
// HTTP core
// ==============================

async function ReadJSONResponse<T>(
  Res: Response,
  Url: string,
): Promise<T> {
  const Ct = Res.headers.get("content-type") || "";
  const Text = await Res.text().catch(() => "");

  if (!Ct.includes("application/json")) {
    if (!Res.ok) {
      throw new Error(Text || `HTTP ${Res.status}`);
    }

    throw new Error(
      `Expected JSON but got "${Ct}". URL=${Url}. Body(head)=${Text.slice(0, 200)}`,
    );
  }

  let Data: unknown = null;

  try {
    Data = Text ? JSON.parse(Text) : null;
  } catch {
    throw new Error(
      `Invalid JSON response. URL=${Url}. Body(head)=${Text.slice(0, 200)}`,
    );
  }

  if (!Res.ok) {
    const ErrorData =
      Data && typeof Data === "object"
        ? (Data as Record<string, unknown>)
        : null;

    const Message =
      typeof ErrorData?.Error === "string"
        ? ErrorData.Error
        : typeof ErrorData?.error === "string"
          ? ErrorData.error
          : typeof ErrorData?.message === "string"
            ? ErrorData.message
            : "";

    throw new Error(Message || `HTTP ${Res.status}`);
  }

  return Data as T;
}

async function HttpGetJSON<T>(Url: string): Promise<T> {
  const Headers = await getAuthHeaders();

  const Res = await fetch(Url, {
    method: "GET",
    headers: {
      ...Headers,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    credentials: "include",
  });

  return ReadJSONResponse<T>(Res, Url);
}

async function HttpPostJSON<T>(
  Url: string,
  Body: unknown,
): Promise<T> {
  const Headers = await getAuthHeaders();

  const Res = await fetch(Url, {
    method: "POST",
    headers: {
      ...Headers,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(Body),
  });

  return ReadJSONResponse<T>(Res, Url);
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
    Params: ListProductBlueprintReviewsParams,
  ): Promise<ListProductBlueprintReviewsResponse> {
    if (!Params.ProductBlueprintID.trim()) {
      throw new Error("ProductBlueprintID is required");
    }

    const Path = `/product-blueprint-reviews${BuildQuery(Params)}`;
    const Url = `${this.BaseURL}${Path}`;

    return HttpGetJSON<ListProductBlueprintReviewsResponse>(Url);
  }

  /**
   * Management: GET /product-blueprint-reviews/aggregates
   * Query: Status, Page, PerPage
   */
  async ListCompanyReviewAggregates(
    Params?: ListCompanyReviewAggregatesParams,
  ): Promise<ListCompanyReviewAggregatesResponse> {
    const Path = `/product-blueprint-reviews/aggregates${BuildQuery(Params)}`;
    const Url = `${this.BaseURL}${Path}`;

    return HttpGetJSON<ListCompanyReviewAggregatesResponse>(Url);
  }

  /**
   * Report:
   * POST /product-blueprints/{productBlueprintId}/reviews/{reviewId}/reports
   */
  async ReportProductBlueprintReview(
    Input: ReportProductBlueprintReviewInput,
  ): Promise<ReviewReportResponse> {
    const ProductBlueprintID = Input.productBlueprintId.trim();
    const ReviewID = Input.reviewId.trim();
    const Detail = Input.detail?.trim() ?? "";

    if (!ProductBlueprintID) {
      throw new Error("productBlueprintId is required");
    }

    if (!ReviewID) {
      throw new Error("reviewId is required");
    }

    if (requiresReviewReportDetail(Input.reason) && !Detail) {
      throw new Error("「その他」を選択した場合は詳細を入力してください。");
    }

    const Request: ReviewReportRequest = {
      reason: Input.reason,
      ...(Detail ? { detail: Detail } : {}),
    };

    const Path =
      `/product-blueprints/${encodeURIComponent(ProductBlueprintID)}` +
      `/reviews/${encodeURIComponent(ReviewID)}/reports`;
    const Url = `${this.BaseURL}${Path}`;

    return HttpPostJSON<ReviewReportResponse>(Url, Request);
  }
}

export const productBlueprintReviewHTTP = new ProductBlueprintReviewHTTP();