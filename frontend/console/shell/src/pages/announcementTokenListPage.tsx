// frontend/console/shell/src/pages/announcementTokenListPage.tsx

import {
  useMemo,
  useState,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { useNavigate } from "react-router-dom";

import { buildAnnouncementTokenListNavigateState } from "../features/announcement/application/announcement_token_list_service";
import { useAnnouncementTokenListPage } from "../features/announcement/presentation/hook/useAnnouncementTokenListPage";
import List, { SortableTableHeader } from "../layout/List/List";
import PageStyle from "../layout/PageStyle/PageStyle";
import FilterableTableHeader from "../shared/ui/filter-able-table-header";
import { TableCell, TableRow } from "../shared/ui/table";

export default function AnnouncementTokenListPage() {
  const navigate = useNavigate();

  const {
    rows,
    sortKey,
    sortDir,
    handleChangeSort,
    handleReset,
    isResetting,
  } = useAnnouncementTokenListPage();

  const [selectedBrandNames, setSelectedBrandNames] = useState<string[]>([]);

  const brandOptions = useMemo(() => {
    const brandNames = rows
      .map((row) => row.brandName)
      .filter(
        (brandName): brandName is string =>
          Boolean(brandName),
      );

    return Array.from(new Set(brandNames)).map((brandName) => ({
      label: brandName,
      value: brandName,
    }));
  }, [rows]);

  const filteredRows = useMemo(() => {
    if (selectedBrandNames.length === 0) {
      return rows;
    }

    return rows.filter((row) =>
      selectedBrandNames.includes(row.brandName),
    );
  }, [rows, selectedBrandNames]);

  const handleBrandFilterChange = (next: string[]) => {
    setSelectedBrandNames(next);
  };

  const handlePageReset = async () => {
    setSelectedBrandNames([]);
    await handleReset();
  };

  const handleBack = () => {
    navigate("/sales", {
      replace: true,
    });
  };

  const handleRowClick = (tokenBlueprintId: string) => {
    const id = String(tokenBlueprintId ?? "").trim();

    if (!id) {
      return;
    }

    const row = rows.find(
      (item) => item.tokenBlueprintId === id,
    );

    navigate(
      `/sales/${encodeURIComponent(id)}/create`,
      {
        state: buildAnnouncementTokenListNavigateState(row),
      },
    );
  };

  const headers: ReactNode[] = [
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
    <PageStyle
      layout="single"
      title="告知対象トークンを選択"
      onBack={handleBack}
      onRefresh={handlePageReset}
      isRefreshing={isResetting}
    >
      <div className="p-0">
        <List
          headerCells={headers}
          showResetButton={false}
        >
          {filteredRows.map((row) => (
            <TableRow
              key={row.tokenBlueprintId}
              role="button"
              tabIndex={0}
              className="cursor-pointer"
              onClick={() =>
                handleRowClick(row.tokenBlueprintId)
              }
              onKeyDown={(
                event: KeyboardEvent<HTMLTableRowElement>,
              ) => {
                if (
                  event.key === "Enter" ||
                  event.key === " "
                ) {
                  event.preventDefault();
                  handleRowClick(row.tokenBlueprintId);
                }
              }}
            >
              <TableCell>{row.tokenName}</TableCell>
              <TableCell>{row.brandName}</TableCell>
              <TableCell>{row.issueCount}</TableCell>
              <TableCell>{row.distributionCount}</TableCell>
            </TableRow>
          ))}
        </List>
      </div>
    </PageStyle>
  );
}