// frontend/admin/shell/src/features/company/presentation/components/ContractTokenBlueprintTable.tsx

import { useMemo } from "react";

import type { ContractTokenBlueprintRow } from "../../../../shared/type/contractDetail";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ContractTokenBlueprintTableProps = {
  tokenBlueprints: ContractTokenBlueprintRow[];
  onTokenBlueprintClick?: (tokenBlueprint: ContractTokenBlueprintRow) => void;
};

export default function ContractTokenBlueprintTable({
  tokenBlueprints,
  onTokenBlueprintClick,
}: ContractTokenBlueprintTableProps) {
  const brandOptions = useMemo(
    () =>
      Array.from(
        new Set(
          tokenBlueprints
            .map((tokenBlueprint) => tokenBlueprint.brandName?.trim())
            .filter((name): name is string => Boolean(name)),
        ),
      )
        .sort((a, b) => a.localeCompare(b, "ja"))
        .map((name) => ({
          value: name,
          label: name,
        })),
    [tokenBlueprints],
  );

  const assigneeOptions = useMemo(
    () =>
      Array.from(
        new Set(
          tokenBlueprints
            .map((tokenBlueprint) => tokenBlueprint.assigneeName?.trim())
            .filter((name): name is string => Boolean(name)),
        ),
      )
        .sort((a, b) => a.localeCompare(b, "ja"))
        .map((name) => ({
          value: name,
          label: name,
        })),
    [tokenBlueprints],
  );

  const columns = useMemo<TableColumn<ContractTokenBlueprintRow>[]>(
    () => [
      {
        key: "name",
        header: "トークン",
        render: (tokenBlueprint) => tokenBlueprint.name || tokenBlueprint.id,
        nowrap: true,
      },
      {
        key: "symbol",
        header: "シンボル",
        render: (tokenBlueprint) => tokenBlueprint.symbol || "-",
        nowrap: true,
      },
      {
        key: "brandName",
        header: "ブランド",
        render: (tokenBlueprint) => tokenBlueprint.brandName || "-",
        filter: {
          getValue: (tokenBlueprint) => tokenBlueprint.brandName,
          options: brandOptions,
        },
        nowrap: true,
      },
      {
        key: "assigneeName",
        header: "担当者",
        render: (tokenBlueprint) => tokenBlueprint.assigneeName || "-",
        filter: {
          getValue: (tokenBlueprint) => tokenBlueprint.assigneeName,
          options: assigneeOptions,
        },
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "作成日時",
        render: (tokenBlueprint) => formatDateTime(tokenBlueprint.createdAt),
        sortValue: (tokenBlueprint) =>
          new Date(tokenBlueprint.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "最終更新日時",
        render: (tokenBlueprint) => formatDateTime(tokenBlueprint.updatedAt),
        sortValue: (tokenBlueprint) =>
          new Date(tokenBlueprint.updatedAt).getTime(),
        nowrap: true,
      },
    ],
    [assigneeOptions, brandOptions],
  );

  return (
    <Table
      columns={columns}
      rows={tokenBlueprints}
      getRowKey={(tokenBlueprint) => tokenBlueprint.id}
      onRowClick={onTokenBlueprintClick}
      emptyMessage="トークン設計はありません。"
      filteredEmptyMessage="条件に一致するトークン設計はありません。"
    />
  );
}