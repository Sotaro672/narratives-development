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

const COMMENT_KIND_LABELS: Record<string, string> = {
  user: "コメント",
  purchase: "購入",
};

const READ_STATUS_LABELS: Record<string, string> = {
  read: "既読",
  unread: "未読",
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
        key: "kind",
        header: "種別",
        render: (review) => COMMENT_KIND_LABELS[review.kind] ?? review.kind,
        filter: {
          getValue: (review) => review.kind,
          options: [
            { value: "user", label: "コメント" },
            { value: "purchase", label: "購入" },
          ],
        },
        nowrap: true,
      },
      {
        key: "body",
        header: "本文",
        render: (review) => review.body || "-",
        minWidth: "280px",
      },
      {
        key: "isRead",
        header: "確認状況",
        render: (review) =>
          READ_STATUS_LABELS[review.isRead ? "read" : "unread"],
        filter: {
          getValue: (review) => (review.isRead ? "read" : "unread"),
          options: [
            { value: "read", label: "既読" },
            { value: "unread", label: "未読" },
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