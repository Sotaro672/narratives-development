// frontend/admin/shell/src/features/company/presentation/components/ContractProductBlueprintTable.tsx

import { useMemo } from "react";

import type { ContractProductBlueprintRow } from "../../../../shared/type/contractDetail";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ContractProductBlueprintTableProps = {
  productBlueprints: ContractProductBlueprintRow[];
  onProductBlueprintClick?: (
    productBlueprint: ContractProductBlueprintRow,
  ) => void;
};

export default function ContractProductBlueprintTable({
  productBlueprints,
  onProductBlueprintClick,
}: ContractProductBlueprintTableProps) {
  const brandOptions = useMemo(
    () =>
      Array.from(
        new Set(
          productBlueprints
            .map((productBlueprint) => productBlueprint.brandName?.trim())
            .filter((name): name is string => Boolean(name)),
        ),
      )
        .sort((a, b) => a.localeCompare(b, "ja"))
        .map((name) => ({
          value: name,
          label: name,
        })),
    [productBlueprints],
  );

  const assigneeOptions = useMemo(
    () =>
      Array.from(
        new Set(
          productBlueprints
            .map((productBlueprint) => productBlueprint.assigneeName?.trim())
            .filter((name): name is string => Boolean(name)),
        ),
      )
        .sort((a, b) => a.localeCompare(b, "ja"))
        .map((name) => ({
          value: name,
          label: name,
        })),
    [productBlueprints],
  );

  const columns = useMemo<TableColumn<ContractProductBlueprintRow>[]>(
    () => [
      {
        key: "productName",
        header: "プロダクト",
        render: (productBlueprint) =>
          productBlueprint.productName || productBlueprint.id,
        nowrap: true,
      },
      {
        key: "brandName",
        header: "ブランド",
        render: (productBlueprint) => productBlueprint.brandName || "-",
        filter: {
          getValue: (productBlueprint) => productBlueprint.brandName,
          options: brandOptions,
        },
        nowrap: true,
      },
      {
        key: "assigneeName",
        header: "担当者",
        render: (productBlueprint) => productBlueprint.assigneeName || "-",
        filter: {
          getValue: (productBlueprint) => productBlueprint.assigneeName,
          options: assigneeOptions,
        },
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "作成日時",
        render: (productBlueprint) =>
          formatDateTime(productBlueprint.createdAt),
        sortValue: (productBlueprint) =>
          new Date(productBlueprint.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "最終更新日時",
        render: (productBlueprint) =>
          formatDateTime(productBlueprint.updatedAt),
        sortValue: (productBlueprint) =>
          new Date(productBlueprint.updatedAt).getTime(),
        nowrap: true,
      },
    ],
    [assigneeOptions, brandOptions],
  );

  return (
    <Table
      columns={columns}
      rows={productBlueprints}
      getRowKey={(productBlueprint) => productBlueprint.id}
      onRowClick={onProductBlueprintClick}
      emptyMessage="商品設計はありません。"
      filteredEmptyMessage="条件に一致する商品設計はありません。"
    />
  );
}