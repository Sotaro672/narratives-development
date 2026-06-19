// frontend/console/sales/src/presentation/hook/useAnnouncementManagement.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  createEmptyAnnouncementManagementListResult,
  fetchAnnouncementManagementRows,
  normalizeAnnouncementManagementSortKey,
  sortAnnouncementManagementRows,
  type AnnouncementManagementRow,
  type AnnouncementManagementSortDir,
  type AnnouncementManagementSortKey,
} from "../../application/announcement_management_service";

export function useAnnouncementManagement() {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  const [sourceRows, setSourceRows] = useState<AnnouncementManagementRow[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(50);
  const [sortKey, setSortKey] =
    useState<AnnouncementManagementSortKey>("createdAt");
  const [sortDir, setSortDir] =
    useState<AnnouncementManagementSortDir>("desc");
  const [isResetting, setIsResetting] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const targetToken = useMemo(() => {
    return String(tokenBlueprintId ?? "").trim();
  }, [tokenBlueprintId]);

  const load = useCallback(async () => {
    if (!targetToken) {
      const empty = createEmptyAnnouncementManagementListResult(page, perPage);
      setSourceRows(empty.rows);
      setTotalCount(empty.totalCount);
      setPage(empty.page);
      setPerPage(empty.perPage);
      return;
    }

    setIsLoading(true);

    try {
      const result = await fetchAnnouncementManagementRows({
        targetToken,
        page,
        perPage,
      });

      setSourceRows(result.rows);
      setTotalCount(result.totalCount);
      setPage(result.page);
      setPerPage(result.perPage);
    } catch {
      const empty = createEmptyAnnouncementManagementListResult(page, perPage);
      setSourceRows(empty.rows);
      setTotalCount(empty.totalCount);
      setPage(empty.page);
      setPerPage(empty.perPage);
    } finally {
      setIsLoading(false);
    }
  }, [page, perPage, targetToken]);

  useEffect(() => {
    void load();
  }, [load]);

  const rows = useMemo(() => {
    return sortAnnouncementManagementRows(sourceRows, sortKey, sortDir);
  }, [sourceRows, sortDir, sortKey]);

  const handleChangeSort = useCallback((nextKey: string) => {
    const normalizedKey = normalizeAnnouncementManagementSortKey(nextKey);

    setSortKey((prevKey) => {
      if (prevKey === normalizedKey) {
        setSortDir((prevDir) => (prevDir === "asc" ? "desc" : "asc"));
        return prevKey;
      }

      setSortDir("asc");
      return normalizedKey;
    });
  }, []);

  const handleReset = useCallback(async () => {
    setIsResetting(true);

    try {
      setSortKey("createdAt");
      setSortDir("desc");
      setPage(1);
      await load();
    } finally {
      setIsResetting(false);
    }
  }, [load]);

  const handleCreate = useCallback(() => {
    if (!targetToken) {
      navigate("/sales/create");
      return;
    }

    navigate(`/sales/${encodeURIComponent(targetToken)}/create`);
  }, [navigate, targetToken]);

  const handleRowClick = useCallback(
    (announcementId: string) => {
      const id = String(announcementId ?? "").trim();
      if (!id) return;

      navigate(`./${encodeURIComponent(id)}`);
    },
    [navigate],
  );

  return {
    rows,
    totalCount,
    page,
    perPage,
    sortKey,
    sortDir,
    handleChangeSort,
    handleReset,
    handleCreate,
    handleRowClick,
    isResetting,
    isLoading,
  };
}

export default useAnnouncementManagement;