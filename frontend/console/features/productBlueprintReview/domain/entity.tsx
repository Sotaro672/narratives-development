// frontend/console/productBlueprintReview/src/domain/entity.tsx

// ==============================
// Backend-aligned domain entities (PascalCase only)
// ==============================

export type ReviewStatus = "PUBLISHED" | "HIDDEN" | "REMOVED";

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

// Detail page: GET /product-blueprint-reviews?ProductBlueprintID=...
export type ListProductBlueprintReviewsParams = {
  ProductBlueprintID: string; // required
  Status?: ReviewStatus;
  Page?: number;
  PerPage?: number;
};

export type ListProductBlueprintReviewsResponse = {
  ProductBlueprintID: string;
  Status: ReviewStatus;
  Page: number;
  PerPage: number;
  Items: Review[]; // ✅ Review に AvatarName/Icon が含まれるようになった
  TotalCount: number;
  TotalPages: number;
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

export type ListCompanyReviewAggregatesResponse = {
  CompanyID: string;
  Status: ReviewStatus;
  Page: number;
  PerPage: number;
  Items: ProductBlueprintReviewAggregate[];
  TotalCount?: number;
  TotalPages?: number;
};