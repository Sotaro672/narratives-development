// frontend/console/shell/src/features/productBlueprintReview/presentation/hook/useProductBlueprintReviewDetail.tsx

import * as React from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import {
  FetchProductBlueprintReviewDetailRows,
} from "../../application/productBlueprintReviewDetailService";
import {
  productBlueprintReviewHTTP,
} from "../../infrastructure/productBlueprintReviewHTTP";

import type {
  Review,
  ReviewStatus,
} from "../../../../shared/types/productBlueprintReview";
import type {
  ReviewReportReason,
  ReviewReportResponse,
} from "../../../../shared/types/reviewReport";
import {
  requiresReviewReportDetail,
} from "../../../../shared/types/reviewReport";

const PER_PAGE = 20;
const DEFAULT_REPORT_REASON: ReviewReportReason = "INAPPROPRIATE";

export type UseProductBlueprintReviewDetailResult = {
  ProductBlueprintID: string;

  Status: ReviewStatus;
  Page: number;

  Items: Review[];
  TotalPages: number;

  IsLoading: boolean;
  ErrorMessage: string;

  IsReportOpen: boolean;
  ReportTargetReviewID: string;
  ReportReason: ReviewReportReason;
  ReportDetail: string;
  ReportSubmitting: boolean;
  ReportErrorMessage: string;
  ReportResult: ReviewReportResponse | null;
  CanSubmitReport: boolean;

  OnBack: () => void;
  OnReload: () => void;

  SetStatus: (Next: ReviewStatus) => void;
  SetPage: (Next: number) => void;

  OpenReport: (ReviewID: string) => void;
  CloseReport: () => void;
  SetReportReason: (Next: ReviewReportReason) => void;
  SetReportDetail: (Next: string) => void;
  SubmitReport: () => Promise<void>;
};

export function useProductBlueprintReviewDetail(): UseProductBlueprintReviewDetailResult {
  const Params = useParams();
  const Navigate = useNavigate();
  const ReportSubmittingRef = React.useRef(false);

  const ProductBlueprintID = String(
    Params.productBlueprintReviewId ?? "",
  ).trim();

  const [Status, SetStatusState] =
    React.useState<ReviewStatus>("PUBLISHED");

  const [Page, SetPageState] =
    React.useState<number>(1);

  const [Items, SetItems] =
    React.useState<Review[]>([]);

  const [TotalPages, SetTotalPages] =
    React.useState<number>(0);

  const [IsLoading, SetIsLoading] =
    React.useState<boolean>(false);

  const [ErrorMessage, SetErrorMessage] =
    React.useState<string>("");

  const [IsReportOpen, SetIsReportOpen] =
    React.useState<boolean>(false);

  const [ReportTargetReviewID, SetReportTargetReviewID] =
    React.useState<string>("");

  const [ReportReason, SetReportReasonState] =
    React.useState<ReviewReportReason>(DEFAULT_REPORT_REASON);

  const [ReportDetail, SetReportDetailState] =
    React.useState<string>("");

  const [ReportSubmitting, SetReportSubmitting] =
    React.useState<boolean>(false);

  const [ReportErrorMessage, SetReportErrorMessage] =
    React.useState<string>("");

  const [ReportResult, SetReportResult] =
    React.useState<ReviewReportResponse | null>(null);

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

      SetItems(Response.items ?? []);
      SetTotalPages(Response.totalPages ?? 0);
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
          ? Math.trunc(NormalizedPage)
          : 1,
      );
    },
    [],
  );

  const OpenReport = React.useCallback(
    (ReviewID: string) => {
      const NormalizedReviewID = ReviewID.trim();

      if (
        !ProductBlueprintID ||
        !NormalizedReviewID ||
        ReportSubmittingRef.current
      ) {
        return;
      }

      SetReportTargetReviewID(NormalizedReviewID);
      SetReportReasonState(DEFAULT_REPORT_REASON);
      SetReportDetailState("");
      SetReportErrorMessage("");
      SetReportResult(null);
      SetIsReportOpen(true);
    },
    [ProductBlueprintID],
  );

  const CloseReport = React.useCallback(() => {
    if (ReportSubmittingRef.current) {
      return;
    }

    SetIsReportOpen(false);
    SetReportTargetReviewID("");
    SetReportReasonState(DEFAULT_REPORT_REASON);
    SetReportDetailState("");
    SetReportErrorMessage("");
    SetReportResult(null);
  }, []);

  const SetReportReason = React.useCallback(
    (Next: ReviewReportReason) => {
      if (ReportSubmittingRef.current) {
        return;
      }

      SetReportReasonState(Next);
      SetReportErrorMessage("");

      if (!requiresReviewReportDetail(Next)) {
        SetReportDetailState("");
      }
    },
    [],
  );

  const SetReportDetail = React.useCallback(
    (Next: string) => {
      if (ReportSubmittingRef.current) {
        return;
      }

      SetReportDetailState(Next);
      SetReportErrorMessage("");
    },
    [],
  );

  const CanSubmitReport = Boolean(
    IsReportOpen &&
      ProductBlueprintID &&
      ReportTargetReviewID &&
      !ReportSubmitting &&
      !ReportResult &&
      (
        !requiresReviewReportDetail(ReportReason) ||
        ReportDetail.trim()
      ),
  );

  const SubmitReport = React.useCallback(async (): Promise<void> => {
    const NormalizedProductBlueprintID =
      ProductBlueprintID.trim();

    const NormalizedReviewID =
      ReportTargetReviewID.trim();

    const NormalizedDetail =
      ReportDetail.trim();

    if (
      ReportSubmittingRef.current ||
      !IsReportOpen ||
      !NormalizedProductBlueprintID ||
      !NormalizedReviewID ||
      ReportResult
    ) {
      return;
    }

    if (
      requiresReviewReportDetail(ReportReason) &&
      !NormalizedDetail
    ) {
      SetReportErrorMessage(
        "「その他」を選択した場合は詳細を入力してください。",
      );
      return;
    }

    ReportSubmittingRef.current = true;
    SetReportSubmitting(true);
    SetReportErrorMessage("");

    try {
      const Result =
        await productBlueprintReviewHTTP.ReportProductBlueprintReview({
          productBlueprintId: NormalizedProductBlueprintID,
          reviewId: NormalizedReviewID,
          reason: ReportReason,
          ...(NormalizedDetail
            ? { detail: NormalizedDetail }
            : {}),
        });

      SetReportResult(Result);
    } catch (error: unknown) {
      const Message =
        error instanceof Error
          ? error.message
          : String(
              error ??
                "レビューの通報に失敗しました。",
            );

      SetReportErrorMessage(
        Message || "レビューの通報に失敗しました。",
      );
    } finally {
      ReportSubmittingRef.current = false;
      SetReportSubmitting(false);
    }
  }, [
    ProductBlueprintID,
    ReportTargetReviewID,
    ReportReason,
    ReportDetail,
    ReportResult,
    IsReportOpen,
  ]);

  return {
    ProductBlueprintID,

    Status,
    Page,

    Items,
    TotalPages,

    IsLoading,
    ErrorMessage,

    IsReportOpen,
    ReportTargetReviewID,
    ReportReason,
    ReportDetail,
    ReportSubmitting,
    ReportErrorMessage,
    ReportResult,
    CanSubmitReport,

    OnBack,
    OnReload,

    SetStatus,
    SetPage,

    OpenReport,
    CloseReport,
    SetReportReason,
    SetReportDetail,
    SubmitReport,
  };
}