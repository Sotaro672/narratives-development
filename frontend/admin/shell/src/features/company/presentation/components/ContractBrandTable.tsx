// frontend/admin/shell/src/features/company/presentation/components/ContractBrandTable.tsx

import { useMemo } from "react";

import type { ContractBrandRow } from "../../../../shared/type/contractDetail";
import Table, {
  type TableColumn,
  type TableFilterOption,
} from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ContractBrandTableProps = {
  brands: ContractBrandRow[];
  onBrandClick?: (brand: ContractBrandRow) => void;
};

const STATUS_LABELS: Record<string, string> = {
  active: "有効",
  inactive: "無効",
};

const STATUS_OPTIONS: TableFilterOption[] = Object.entries(
  STATUS_LABELS,
).map(([value, label]) => ({
  value,
  label,
}));

export default function ContractBrandTable({
  brands,
  onBrandClick,
}: ContractBrandTableProps) {
  const columns = useMemo<TableColumn<ContractBrandRow>[]>(
    () => [
      {
        key: "name",
        header: "ブランド名",
        render: (brand) => brand.name || "-",
        sortValue: (brand) => brand.name,
        nowrap: true,
      },
      {
        key: "managerName",
        header: "責任者",
        render: (brand) => brand.managerName || "-",
        sortValue: (brand) => brand.managerName,
        nowrap: true,
      },
      {
        key: "status",
        header: "状態",
        render: (brand) =>
          brand.isActive ? STATUS_LABELS.active : STATUS_LABELS.inactive,
        filter: {
          getValue: (brand) => (brand.isActive ? "active" : "inactive"),
          options: STATUS_OPTIONS,
        },
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "登録日時",
        render: (brand) =>
          brand.createdAt ? formatDateTime(brand.createdAt) : "-",
        sortValue: (brand) =>
          brand.createdAt ? new Date(brand.createdAt).getTime() : 0,
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "更新日時",
        render: (brand) =>
          brand.updatedAt ? formatDateTime(brand.updatedAt) : "-",
        sortValue: (brand) =>
          brand.updatedAt ? new Date(brand.updatedAt).getTime() : 0,
        nowrap: true,
      },
    ],
    [],
  );

  return (
    <Table
      columns={columns}
      rows={brands}
      getRowKey={(brand) => brand.id}
      onRowClick={onBrandClick}
      emptyMessage="ブランドはありません。"
      filteredEmptyMessage="条件に一致するブランドはありません。"
    />
  );
}