// frontend/admin/shell/src/features/company/presentation/components/TokenBlueprintReviewTable.tsx

import { useMemo } from "react";

import type { ContractTokenBlueprintReview } from "../../../../shared/type/contractTokenBlueprintReview";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import { useContractTokenBlueprintReviews } from "../hooks/useContractTokenBlueprintReviews";

type TokenBlueprintReviewTableProps = {
  companyId: string;
  tokenBlueprintId: string;
  page?: number;
  perPage?: number;
};

const AUTHOR_TYPE_LABELS: Record<string, string> = {
  avatar: "アバター",
  brand: "ブランド",
};

export default function TokenBlueprintReviewTable({
  companyId,
  tokenBlueprintId,
  page = 1,
  perPage = 20,
}: TokenBlueprintReviewTableProps) {
  const { reviews, loading, error, reload } =
    useContractTokenBlueprintReviews(
      companyId,
      tokenBlueprintId,
      page,
      perPage,
    );

  const columns = useMemo<TableColumn<ContractTokenBlueprintReview>[]>(
    () => [
      {
        key: "authorName",
        header: "投稿者",
        render: (review) => review.authorName || review.authorId || "-",
        sortValue: (review) => review.authorName || review.authorId,
        filter: {
          getValue: (review) =>
            [review.authorName, review.authorId].filter(Boolean).join(" "),
          placeholder: "投稿者名・IDで絞り込み",
        },
        nowrap: true,
      },
      {
        key: "authorType",
        header: "種別",
        render: (review) =>
          AUTHOR_TYPE_LABELS[review.authorType] ?? review.authorType,
        sortValue: (review) => review.authorType,
        filter: {
          getValue: (review) => review.authorType,
          options: [
            { value: "avatar", label: "アバター" },
            { value: "brand", label: "ブランド" },
          ],
        },
        nowrap: true,
      },
      {
        key: "body",
        header: "コメント",
        render: (review) => review.body || "-",
        filter: {
          getValue: (review) => review.body,
          placeholder: "コメントで絞り込み",
        },
        minWidth: "320px",
      },
      {
        key: "isOwnerComment",
        header: "オーナー",
        render: (review) => (review.isOwnerComment ? "はい" : "いいえ"),
        sortValue: (review) => review.isOwnerComment,
        filter: {
          getValue: (review) =>
            review.isOwnerComment ? "owner" : "not_owner",
          options: [
            { value: "owner", label: "オーナーコメント" },
            { value: "not_owner", label: "その他" },
          ],
        },
        nowrap: true,
      },
      {
        key: "likeCount",
        header: "いいね",
        render: (review) => review.likeCount,
        sortValue: (review) => review.likeCount,
        nowrap: true,
      },
      {
        key: "dislikeCount",
        header: "低評価",
        render: (review) => review.dislikeCount,
        sortValue: (review) => review.dislikeCount,
        nowrap: true,
      },
      {
        key: "childCount",
        header: "返信",
        render: (review) => review.childCount,
        sortValue: (review) => review.childCount,
        nowrap: true,
      },
      {
        key: "deleted",
        header: "状態",
        render: (review) => (review.deleted ? "削除済み" : "表示中"),
        sortValue: (review) => review.deleted,
        filter: {
          getValue: (review) =>
            review.deleted ? "deleted" : "visible",
          options: [
            { value: "visible", label: "表示中" },
            { value: "deleted", label: "削除済み" },
          ],
        },
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "投稿日時",
        render: (review) => formatDateTime(review.createdAt),
        sortValue: (review) => new Date(review.createdAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  if (loading && !reviews) {
    return <p>トークン設計レビューを取得しています...</p>;
  }

  if (error && !reviews) {
    return (
      <div role="alert">
        <p>トークン設計レビューを取得できませんでした。</p>
        <p>{error}</p>
        <button type="button" onClick={() => void reload()}>
          再読み込み
        </button>
      </div>
    );
  }

  return (
    <Table
      columns={columns}
      rows={reviews?.items ?? []}
      getRowKey={(review) => review.commentId}
      emptyMessage="レビューはありません。"
      filteredEmptyMessage="条件に一致するレビューはありません。"
    />
  );
}