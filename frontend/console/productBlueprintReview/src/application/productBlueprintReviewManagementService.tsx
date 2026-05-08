// frontend/console/productBlueprintReview/src/application/productBlueprintReviewManagementService.tsx

import { productBlueprintReviewHTTP } from "../infrastructure/productBlueprintReviewHTTP";

import type {
  ListCompanyReviewAggregatesParams,
  ListCompanyReviewAggregatesResponse,
  ProductBlueprintReviewAggregate,
  ReviewStatus,
} from "../domain/entity";

// 画面側で使う行（Aggregate をそのまま返す / PascalCase）
export type UiRow = ProductBlueprintReviewAggregate;

// PascalCase で一貫（必要なら拡張）
export type ProductBlueprintReviewSortKey =
  | "ProductName"
  | "TotalCount"
  | "AverageRating"
  | null;

export type SortDirection = "Asc" | "Desc" | null;

// ★ management 側の一覧取得（Aggregates）
export async function FetchProductBlueprintReviewManagementRows(Params: {
  Status?: ReviewStatus;
  Page?: number;
  PerPage?: number;
}): Promise<UiRow[]> {
  const { Status, Page, PerPage } = Params;

  const Q: ListCompanyReviewAggregatesParams = {
    Status,
    Page,
    PerPage,
  };

  const Res: ListCompanyReviewAggregatesResponse =
    await productBlueprintReviewHTTP.ListCompanyReviewAggregates(Q);

  return (Res.Items ?? []) as UiRow[];
}

const ToNum = (V: any) => {
  const N = typeof V === "number" ? V : Number(V);
  return Number.isFinite(N) ? N : 0;
};

const ToStr = (V: any) => String(V ?? "");

export function FilterAndSortProductBlueprintReviewRows(Params: {
  AllRows: UiRow[];
  BrandFilter: string[];
  AssigneeFilter: string[];
  SortedKey: ProductBlueprintReviewSortKey;
  SortedDir: SortDirection;
}): UiRow[] {
  const { AllRows, BrandFilter, AssigneeFilter, SortedKey, SortedDir } = Params;

  let Work = AllRows;

  // ✅ Name 解決済みのみを使う方針でも、フィルタ値は "name" で渡ってくる可能性があるので、
  // BrandName/AssigneeName で filter する（ID filter はしない）
  if (BrandFilter?.length) {
    const BrandNameSet = new Set<string>(BrandFilter);
    Work = Work.filter((R) => BrandNameSet.has(ToStr(R.BrandName)));
  }

  if (AssigneeFilter?.length) {
    const AssigneeNameSet = new Set<string>(AssigneeFilter);
    Work = Work.filter((R) => AssigneeNameSet.has(ToStr(R.AssigneeName)));
  }

  if (SortedKey && SortedDir) {
    Work = [...Work].sort((A: any, B: any) => {
      const Av =
        SortedKey === "ProductName"
          ? ToStr(A[SortedKey]).localeCompare(ToStr(B[SortedKey]))
          : ToNum(A[SortedKey]) - ToNum(B[SortedKey]);

      return SortedDir === "Asc" ? Av : -Av;
    });
  }

  return Work;
}