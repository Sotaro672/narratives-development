// frontend/console/sales/src/presentation/pages/announcementManagement.tsx
import React, { useMemo, useState } from "react";
import List, {
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import FilterableTableHeader from "../../../shell/src/shared/ui/filterable-table-header";
import { useAnnouncementManagement } from "../hook/useAnnouncementManagement";

export default function AnnouncementManagementPage() {
  const {
    rows,
    sortKey,
    sortDir,
    handleChangeSort,
    handleReset,
    handleCreate,
    handleRowClick,
    isResetting,
    isLoading,
  } = useAnnouncementManagement();

  const [selectedPublishedValues, setSelectedPublishedValues] = useState<
    string[]
  >([]);

  const publishedOptions = useMemo(
    () => [
      {
        label: "公開済み",
        value: "published",
      },
      {
        label: "下書き",
        value: "draft",
      },
    ],
    [],
  );

  const filteredRows = useMemo(() => {
    if (selectedPublishedValues.length === 0) {
      return rows;
    }

    return rows.filter((row) => {
      const statusValue = row.published ? "published" : "draft";
      return selectedPublishedValues.includes(statusValue);
    });
  }, [rows, selectedPublishedValues]);

  const handlePublishedFilterChange = (next: string[]) => {
    setSelectedPublishedValues(next);
  };

  const handlePageReset = async () => {
    setSelectedPublishedValues([]);
    await handleReset();
  };

  const headers: React.ReactNode[] = [
    <SortableTableHeader
      key="title"
      label="タイトル"
      sortKey="title"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
    <FilterableTableHeader
      key="published"
      label="状態"
      options={publishedOptions}
      selected={selectedPublishedValues}
      onChange={handlePublishedFilterChange}
    />,
    <SortableTableHeader
      key="targetAvatarCount"
      label="送信対象数"
      sortKey="targetAvatarCount"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
    <SortableTableHeader
      key="createdAt"
      label="作成日時"
      sortKey="createdAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
    <SortableTableHeader
      key="updatedAt"
      label="更新日時"
      sortKey="updatedAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="告知"
        headerCells={headers}
        showCreateButton
        createLabel="告知を作成"
        showResetButton
        isResetting={isResetting || isLoading}
        onCreate={handleCreate}
        onReset={handlePageReset}
      >
        {filteredRows.map((row) => (
          <tr
            key={row.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer hover:bg-slate-50 transition-colors"
            onClick={() => handleRowClick(row.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                handleRowClick(row.id);
              }
            }}
          >
            <td>{row.title}</td>
            <td>{row.published ? "公開済み" : "下書き"}</td>
            <td>{row.targetAvatarCount}</td>
            <td>{row.createdAt}</td>
            <td>{row.updatedAt ?? ""}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}