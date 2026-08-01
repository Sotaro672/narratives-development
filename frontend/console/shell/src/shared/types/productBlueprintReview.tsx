// frontend\console\shell\src\shared\types\productBluleprintReview.tsx

import type {
  PageResult,
} from "./common/common";

// ==============================
// Backend-aligned domain entities (PascalCase only)
// ==============================

export type ReviewStatus =
  | "PUBLISHED"
  | "HIDDEN"
  | "REMOVED";

export type Review = {
  ID: string;

  ProductBlueprintID: string;
  AvatarID: string;

  // ✅ backend(usecase) が同梱する追加フィールド
  AvatarName?: string;
  AvatarIcon?: string;

  Rating: number;
  Title: string;
  Body: string;

  HelpfulVotes: number;
  TotalVotes: number;

  ReviewedAt: string;

  Status: ReviewStatus;

  CreatedAt: string;
  CreatedBy: string;
  UpdatedAt: string;
  UpdatedBy: string;

  ModerationReason?: string | null;
};

// ==============================
// PascalCase page result
// ==============================

type PascalCasePageResult<T> = {
  [K in keyof PageResult<T> as Capitalize<
    K & string
  >]: PageResult<T>[K];
};

// Detail page: GET /product-blueprint-reviews?ProductBlueprintID=...
export type ListProductBlueprintReviewsParams = {
  ProductBlueprintID: string; // required
  Status?: ReviewStatus;
  Page?: number;
  PerPage?: number;
};

export type ListProductBlueprintReviewsResponse =
  PascalCasePageResult<Review> & {
    ProductBlueprintID: string;
    Status: ReviewStatus;
  };

// Management page aggregates: GET /product-blueprint-reviews/aggregates
export type ProductBlueprintReviewAggregate = {
  ID: string;
  ProductBlueprintID: string;

  ProductName: string;

  BrandID: string;
  BrandName: string;

  AssigneeID: string;
  AssigneeName: string;

  Rating1Count: number;
  Rating2Count: number;
  Rating3Count: number;
  Rating4Count: number;
  Rating5Count: number;

  TotalCount: number;
  AverageRating: number;
};

export type ListCompanyReviewAggregatesParams = {
  Status?: ReviewStatus;
  Page?: number;
  PerPage?: number;
};

type CompanyReviewAggregatesPageResult =
  Omit<
    PascalCasePageResult<
      ProductBlueprintReviewAggregate
    >,
    "TotalCount" | "TotalPages"
  > & {
    TotalCount?: number;
    TotalPages?: number;
  };

export type ListCompanyReviewAggregatesResponse =
  CompanyReviewAggregatesPageResult & {
    CompanyID: string;
    Status: ReviewStatus;
  };