// frontend/admin/shell/src/features/company/presentation/components/CompanyTable.tsx

import { useMemo } from "react";

import type { Company } from "../../../../shared/type/company";
import Table, {
  type TableColumn,
} from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type CompanyTableProps = {
  companies: Company[];
  onCompanyClick?: (company: Company) => void;
};

export default function CompanyTable({
  companies,
  onCompanyClick,
}: CompanyTableProps) {
  const columns = useMemo<TableColumn<Company>[]>(
    () => [
      {
        key: "createdAt",
        header: "登録日時",
        render: (company) => formatDateTime(company.createdAt),
        sortValue: (company) => new Date(company.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "name",
        header: "企業名",
        render: (company) => company.name,
        nowrap: true,
      },
      {
        key: "representativeName",
        header: "代表者",
        render: (company) => company.representativeName || "-",
        nowrap: true,
      },
      {
        key: "isActive",
        header: "契約状態",
        render: (company) => company.isActive ? "契約中" : "停止中",
        sortValue: (company) => company.isActive,
        filter: {
          getValue: (company) => company.isActive ? "契約中" : "停止中",
          options: [
            { value: "契約中", label: "契約中" },
            { value: "停止中", label: "停止中" },
          ],
        },
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "最終更新日時",
        render: (company) => formatDateTime(company.updatedAt),
        sortValue: (company) => new Date(company.updatedAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  return (
    <Table
      columns={columns}
      rows={companies}
      getRowKey={(company) => company.id}
      emptyMessage="登録企業はありません。"
      filteredEmptyMessage="条件に一致する企業はありません。"
      onRowClick={onCompanyClick}
    />
  );
}