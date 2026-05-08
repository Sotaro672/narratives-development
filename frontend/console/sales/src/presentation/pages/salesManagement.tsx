// frontend/console/sales/src/presentation/pages/salesManagement.tsx
import React, { useMemo, useState } from "react";
import List, {
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import FilterableTableHeader from "../../../../shell/src/shared/ui/filterable-table-header";
import { useSalesManagement } from "../hook/useSalesManagement";

export default function SalesManagementPage() {
  const {
    rows,
    sortKey,
    sortDir,
    handleChangeSort,
    handleReset,
    handleRowClick,
    isResetting,
  } = useSalesManagement();

  const [selectedBrandNames, setSelectedBrandNames] = useState<string[]>([]);

  const brandOptions = useMemo(() => {
    return Array.from(
      new Set(
        rows
          .map((row) => row.brandName)
          .filter((brandName): brandName is string => Boolean(brandName)),
      ),
    ).map((brandName) => ({
      label: brandName,
      value: brandName,
    }));
  }, [rows]);

  const filteredRows = useMemo(() => {
    if (selectedBrandNames.length === 0) {
      return rows;
    }

    return rows.filter((row) => selectedBrandNames.includes(row.brandName));
  }, [rows, selectedBrandNames]);

  const handleBrandFilterChange = (next: string[]) => {
    setSelectedBrandNames(next);
  };

  const handlePageReset = async () => {
    setSelectedBrandNames([]);
    await handleReset();
  };

  const headers: React.ReactNode[] = [
    <span key="tokenName">トークン名</span>,
    <FilterableTableHeader
      key="brandName"
      label="ブランド名"
      options={brandOptions}
      selected={selectedBrandNames}
      onChange={handleBrandFilterChange}
    />,
    <SortableTableHeader
      key="issueCount"
      label="発行数"
      sortKey="issueCount"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
    <SortableTableHeader
      key="distributionCount"
      label="所有者数"
      sortKey="distributionCount"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="営業"
        headerCells={headers}
        showResetButton
        isResetting={isResetting}
        onReset={handlePageReset}
      >
        {filteredRows.map((row) => (
          <tr
            key={row.tokenBlueprintId}
            role="button"
            tabIndex={0}
            className="cursor-pointer hover:bg-slate-50 transition-colors"
            onClick={() => handleRowClick(row.tokenBlueprintId)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                handleRowClick(row.tokenBlueprintId);
              }
            }}
          >
            <td>{row.tokenName}</td>
            <td>{row.brandName}</td>
            <td>{Array.isArray(row.mintAddresses) ? row.mintAddresses.length : 0}</td>
            <td>{Array.isArray(row.owners) ? row.owners.length : 0}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}