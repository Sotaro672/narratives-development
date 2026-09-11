// frontend/admin/shell/src/features/company/presentation/components/ProductBlueprintReviewTable.tsx

import { useMemo } from "react";

import type {
  ContractProductBlueprintReview,
  ContractProductBlueprintReviewStatus,
} from "../../model/ProductBlueprintReview";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import { useContractProductBlueprintReviews } from "../hooks/useProductBlueprintReviews";

type ProductBlueprintReviewTableProps = {
  companyId: string;
  productBlueprintId: string;
  status?: ContractProductBlueprintReviewStatus;
  page?: number;
  perPage?: number;
};

const REVIEW_STATUS_LABELS: Record<
  ContractProductBlueprintReviewStatus,
  string
> = {
  PUBLISHED: "公開",
  HIDDEN: "非公開",
  REMOVED: "削除済み",
};

export default function ProductBlueprintReviewTable({
  companyId,
  productBlueprintId,
  status = "PUBLISHED",
  page = 1,
  perPage = 20,
}: ProductBlueprintReviewTableProps) {
  const { reviews, loading, error, reload } =
    useContractProductBlueprintReviews(
      companyId,
      productBlueprintId,
      status,
      page,
      perPage,
    );

  const columns = useMemo<TableColumn<ContractProductBlueprintReview>[]>(
    () => [
      {
        key: "avatarName",
        header: "投稿者",
        render: (review) => review.avatarName || review.avatarId || "-",
        nowrap: true,
      },
      {
        key: "rating",
        header: "評価",
        render: (review) => `${review.rating} / 5`,
        sortValue: (review) => review.rating,
        filter: {
          getValue: (review) => String(review.rating),
          options: [
            { value: "5", label: "5" },
            { value: "4", label: "4" },
            { value: "3", label: "3" },
            { value: "2", label: "2" },
            { value: "1", label: "1" },
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
        key: "status",
        header: "状態",
        render: (review) => REVIEW_STATUS_LABELS[review.status] ?? review.status,
        filter: {
          getValue: (review) => review.status,
          options: [
            { value: "PUBLISHED", label: "公開" },
            { value: "HIDDEN", label: "非公開" },
            { value: "REMOVED", label: "削除済み" },
          ],
        },
        nowrap: true,
      },
      {
        key: "helpfulVotes",
        header: "役に立った",
        render: (review) =>
          review.totalVotes > 0
            ? `${review.helpfulVotes} / ${review.totalVotes}`
            : "0",
        sortValue: (review) => review.helpfulVotes,
        nowrap: true,
      },
      {
        key: "reviewedAt",
        header: "投稿日",
        render: (review) => formatDateTime(review.reviewedAt),
        sortValue: (review) => new Date(review.reviewedAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  if (loading && !reviews) {
    return <p>商品設計レビューを取得しています...</p>;
  }

  if (error && !reviews) {
    return (
      <div role="alert">
        <p>商品設計レビューを取得できませんでした。</p>
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
      getRowKey={(review) => review.id}
      emptyMessage="レビューはありません。"
      filteredEmptyMessage="条件に一致するレビューはありません。"
    />
  );
}