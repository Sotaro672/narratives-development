// frontend/console/productBlueprintReview/src/presentation/hook/useProductBlueprintReviewDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  FetchProductBlueprintReviewDetailRows,
  type ProductBlueprintReviewDetailRow,
} from "../../application/productBlueprintReviewDetailService";

import type { ReviewStatus } from "../../domain/entity";

export type UseProductBlueprintReviewDetailResult = {
  ProductBlueprintID: string;

  Status: ReviewStatus;
  Page: number;
  PerPage: number;

  Items: ProductBlueprintReviewDetailRow[];
  TotalCount: number;
  TotalPages: number;

  IsLoading: boolean;
  ErrorMessage: string;

  OnBack: () => void;
  OnReload: () => void;

  SetStatus: (Next: ReviewStatus) => void;
  SetPage: (Next: number) => void;
  SetPerPage: (Next: number) => void;
};

export function useProductBlueprintReviewDetail(): UseProductBlueprintReviewDetailResult {
  const Params = useParams();
  const Navigate = useNavigate();

  // routes.tsx: { path: ":productBlueprintReviewId", element: <ProductBlueprintReviewDetail /> }
  const ProductBlueprintID = String(Params.productBlueprintReviewId ?? "");

  const [Status, SetStatusState] = React.useState<ReviewStatus>("PUBLISHED");
  const [Page, SetPageState] = React.useState<number>(1);
  const [PerPage, SetPerPageState] = React.useState<number>(20);

  const [Items, SetItems] = React.useState<ProductBlueprintReviewDetailRow[]>([]);
  const [TotalCount, SetTotalCount] = React.useState<number>(0);
  const [TotalPages, SetTotalPages] = React.useState<number>(0);

  const [IsLoading, SetIsLoading] = React.useState<boolean>(false);
  const [ErrorMessage, SetErrorMessage] = React.useState<string>("");

  const Load = React.useCallback(async () => {
    if (!ProductBlueprintID) {
      SetItems([]);
      SetTotalCount(0);
      SetTotalPages(0);
      return;
    }

    SetIsLoading(true);
    SetErrorMessage("");

    try {
      const Res = await FetchProductBlueprintReviewDetailRows({
        ProductBlueprintID,
        Status,
        Page,
        PerPage,
      });

      SetItems(Res.Items ?? []);
      SetTotalCount(Res.TotalCount ?? 0);
      SetTotalPages(Res.TotalPages ?? 0);
    } catch (E: any) {
      SetItems([]);
      SetTotalCount(0);
      SetTotalPages(0);
      SetErrorMessage(String(E?.message ?? E ?? "UnknownError"));
    } finally {
      SetIsLoading(false);
    }
  }, [ProductBlueprintID, Status, Page, PerPage]);

  React.useEffect(() => {
    void Load();
  }, [Load]);

  const OnBack = React.useCallback(() => {
    Navigate("..");
  }, [Navigate]);

  const OnReload = React.useCallback(() => {
    void Load();
  }, [Load]);

  const SetStatus = React.useCallback((Next: ReviewStatus) => {
    SetStatusState(Next);
    SetPageState(1);
  }, []);

  const SetPage = React.useCallback((Next: number) => {
    const N = Number(Next);
    SetPageState(Number.isFinite(N) && N > 0 ? N : 1);
  }, []);

  const SetPerPage = React.useCallback((Next: number) => {
    const N = Number(Next);
    const V = Number.isFinite(N) && N > 0 ? N : 20;
    SetPerPageState(V);
    SetPageState(1);
  }, []);

  return {
    ProductBlueprintID,

    Status,
    Page,
    PerPage,

    Items,
    TotalCount,
    TotalPages,

    IsLoading,
    ErrorMessage,

    OnBack,
    OnReload,

    SetStatus,
    SetPage,
    SetPerPage,
  };
}