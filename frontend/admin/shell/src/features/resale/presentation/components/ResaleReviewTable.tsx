// frontend/admin/shell/src/features/resale/presentation/components/ResaleReviewTable.tsx

import { useMemo } from "react";

import type { ResaleReviewComment } from "../../../../shared/type/resaleReview";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import { useResaleReviews } from "../hooks/useResaleReviews";

type ResaleReviewTableProps = {
  avatarId: string;
  resaleId: string;
  page?: number;
  perPage?: number;
};

export default function ResaleReviewTable({
  avatarId,
  resaleId,
  page = 1,
  perPage = 20,
}: ResaleReviewTableProps) {
  const { reviews, loading, error, reload } = useResaleReviews(
    avatarId,
    resaleId,
    page,
    perPage,
  );

  const columns = useMemo<TableColumn<ResaleReviewComment>[]>(
    () => [
      {
        key: "avatarName",
        header: "投稿者",
        render: (review) => review.avatarName || review.avatarId || "-",
        nowrap: true,
      },
      {
        key: "body",
        header: "本文",
        render: (review) => review.body || "-",
        minWidth: "280px",
      },
      {
        key: "reportCount",
        header: "通報数",
        render: (review) => review.reportCount,
        sortValue: (review) => review.reportCount,
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
    return <p>リセールレビューを取得しています...</p>;
  }

  if (error && !reviews) {
    return (
      <div role="alert">
        <p>リセールレビューを取得できませんでした。</p>
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