// frontend/mall/src/features/report/hooks/useReport.ts

import { useCallback, useRef, useState } from "react";

import type {
  ReportReason,
  ReportResponse,
} from "../../shared/types/report";
import {
  reportAvatar,
  reportProductBlueprintReview,
  reportTokenBlueprintComment,
} from "../api/reportApi";

export type ReportTarget =
  | {
      type: "PRODUCT_BLUEPRINT_REVIEW";
      productBlueprintId: string;
      reviewId: string;
    }
  | {
      type: "TOKEN_BLUEPRINT_COMMENT";
      tokenBlueprintId: string;
      commentId: string;
    }
  | {
      type: "AVATAR";
      avatarId: string;
    };

type OpenProductBlueprintReviewReportInput = {
  productBlueprintId: string;
  reviewId: string;
};

type OpenTokenBlueprintCommentReportInput = {
  tokenBlueprintId: string;
  commentId: string;
};

type OpenAvatarReportInput = {
  avatarId: string;
};

const DEFAULT_REASON: ReportReason = "SPAM";

function normalizeId(
  value: string,
  fieldName: string,
): string {
  const normalized = value.trim();

  if (!normalized) {
    throw new Error(`${fieldName}が指定されていません。`);
  }

  return normalized;
}

export function useReport() {
  const submittingRef = useRef(false);

  const [target, setTarget] = useState<ReportTarget | null>(null);
  const [reason, setReasonState] = useState<ReportReason>(DEFAULT_REASON);
  const [detail, setDetailState] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<ReportResponse | null>(null);

  const resetForm = useCallback(() => {
    setReasonState(DEFAULT_REASON);
    setDetailState("");
    setError(null);
    setResult(null);
  }, []);

  const openProductBlueprintReviewReport = useCallback(
    (input: OpenProductBlueprintReviewReportInput) => {
      const productBlueprintId = normalizeId(
        input.productBlueprintId,
        "productBlueprintId",
      );
      const reviewId = normalizeId(input.reviewId, "reviewId");

      resetForm();
      setTarget({
        type: "PRODUCT_BLUEPRINT_REVIEW",
        productBlueprintId,
        reviewId,
      });
    },
    [resetForm],
  );

  const openTokenBlueprintCommentReport = useCallback(
    (input: OpenTokenBlueprintCommentReportInput) => {
      const tokenBlueprintId = normalizeId(
        input.tokenBlueprintId,
        "tokenBlueprintId",
      );
      const commentId = normalizeId(input.commentId, "commentId");

      resetForm();
      setTarget({
        type: "TOKEN_BLUEPRINT_COMMENT",
        tokenBlueprintId,
        commentId,
      });
    },
    [resetForm],
  );

  const openAvatarReport = useCallback(
    (input: OpenAvatarReportInput) => {
      const avatarId = normalizeId(input.avatarId, "avatarId");

      resetForm();
      setTarget({
        type: "AVATAR",
        avatarId,
      });
    },
    [resetForm],
  );

  const close = useCallback(() => {
    if (submittingRef.current) {
      return;
    }

    setTarget(null);
    resetForm();
  }, [resetForm]);

  const setReason = useCallback((value: ReportReason) => {
    setReasonState(value);
    setError(null);

    if (value !== "OTHER") {
      setDetailState("");
    }
  }, []);

  const setDetail = useCallback((value: string) => {
    setDetailState(value);
    setError(null);
  }, []);

  const submit = useCallback(async (): Promise<ReportResponse | null> => {
    if (submittingRef.current) {
      return null;
    }

    if (!target) {
      setError("通報対象が指定されていません。");
      return null;
    }

    const normalizedDetail = detail.trim();

    if (reason === "OTHER" && !normalizedDetail) {
      setError("「その他」を選択した場合は詳細を入力してください。");
      return null;
    }

    submittingRef.current = true;
    setSubmitting(true);
    setError(null);
    setResult(null);

    try {
      let response: ReportResponse;

      switch (target.type) {
        case "PRODUCT_BLUEPRINT_REVIEW":
          response = await reportProductBlueprintReview({
            productBlueprintId: target.productBlueprintId,
            reviewId: target.reviewId,
            reason,
            detail: normalizedDetail || undefined,
          });
          break;

        case "TOKEN_BLUEPRINT_COMMENT":
          response = await reportTokenBlueprintComment({
            tokenBlueprintId: target.tokenBlueprintId,
            commentId: target.commentId,
            reason,
            detail: normalizedDetail || undefined,
          });
          break;

        case "AVATAR":
          response = await reportAvatar({
            avatarId: target.avatarId,
            reason,
            detail: normalizedDetail || undefined,
          });
          break;
      }

      setResult(response);
      return response;
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "通報の送信に失敗しました。",
      );
      return null;
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
  }, [target, reason, detail]);

  const isOpen = target !== null;
  const requiresDetail = reason === "OTHER";
  const canSubmit =
    isOpen &&
    !submitting &&
    (!requiresDetail || detail.trim().length > 0);
  const submitted = result !== null;
  const alreadyReported = result !== null && !result.reportCreated;

  return {
    target,
    isOpen,
    reason,
    detail,
    submitting,
    error,
    result,
    submitted,
    alreadyReported,
    requiresDetail,
    canSubmit,
    openProductBlueprintReviewReport,
    openTokenBlueprintCommentReport,
    openAvatarReport,
    close,
    setReason,
    setDetail,
    submit,
  };
}

export default useReport;