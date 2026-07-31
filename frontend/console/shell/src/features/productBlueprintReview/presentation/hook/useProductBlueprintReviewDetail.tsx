// frontend/console/shell/src/features/productBlueprintReview/presentation/hook/useProductBlueprintReviewDetail.tsx

import * as React from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import {
  FetchProductBlueprintReviewDetailRows,
  type ProductBlueprintReviewDetailRow,
} from "../../application/productBlueprintReviewDetailService";

import type { ReviewStatus } from "../../../../shared/types/productBlueprintReview";

const PER_PAGE = 20;

export type UseProductBlueprintReviewDetailResult = {
  ProductBlueprintID: string;

  Status: ReviewStatus;
  Page: number;

  Items: ProductBlueprintReviewDetailRow[];
  TotalPages: number;

  IsLoading: boolean;
  ErrorMessage: string;

  OnBack: () => void;
  OnReload: () => void;

  SetStatus: (Next: ReviewStatus) => void;
  SetPage: (Next: number) => void;
};

export function useProductBlueprintReviewDetail(): UseProductBlueprintReviewDetailResult {
  const Params = useParams();
  const Navigate = useNavigate();

  const ProductBlueprintID = String(
    Params.productBlueprintReviewId ?? "",
  );

  const [Status, SetStatusState] =
    React.useState<ReviewStatus>("PUBLISHED");

  const [Page, SetPageState] =
    React.useState<number>(1);

  const [Items, SetItems] = React.useState<
    ProductBlueprintReviewDetailRow[]
  >([]);

  const [TotalPages, SetTotalPages] =
    React.useState<number>(0);

  const [IsLoading, SetIsLoading] =
    React.useState<boolean>(false);

  const [ErrorMessage, SetErrorMessage] =
    React.useState<string>("");

  const Load = React.useCallback(async () => {
    if (!ProductBlueprintID) {
      SetItems([]);
      SetTotalPages(0);
      return;
    }

    SetIsLoading(true);
    SetErrorMessage("");

    try {
      const Response =
        await FetchProductBlueprintReviewDetailRows({
          ProductBlueprintID,
          Status,
          Page,
          PerPage: PER_PAGE,
        });

      SetItems(Response.Items ?? []);
      SetTotalPages(Response.TotalPages ?? 0);
      } catch (error: unknown) {
        SetItems([]);
        SetTotalPages(0);

        const Message =
          error instanceof Error
            ? error.message
            : String(error ?? "UnknownError");

        SetErrorMessage(Message);
      } finally {
      SetIsLoading(false);
    }
  }, [
    ProductBlueprintID,
    Status,
    Page,
  ]);

  React.useEffect(() => {
    void Load();
  }, [Load]);

  const OnBack = React.useCallback(() => {
    Navigate("..");
  }, [Navigate]);

  const OnReload = React.useCallback(() => {
    void Load();
  }, [Load]);

  const SetStatus = React.useCallback(
    (Next: ReviewStatus) => {
      SetStatusState(Next);
      SetPageState(1);
    },
    [],
  );

  const SetPage = React.useCallback(
    (Next: number) => {
      const NormalizedPage = Number(Next);

      SetPageState(
        Number.isFinite(NormalizedPage) &&
          NormalizedPage > 0
          ? NormalizedPage
          : 1,
      );
    },
    [],
  );

  return {
    ProductBlueprintID,

    Status,
    Page,

    Items,
    TotalPages,

    IsLoading,
    ErrorMessage,

    OnBack,
    OnReload,

    SetStatus,
    SetPage,
  };
}