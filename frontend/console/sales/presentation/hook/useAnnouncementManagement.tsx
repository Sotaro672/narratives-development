// frontend/console/sales/src/presentation/hook/useAnnouncementManagement.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  createEmptyAnnouncementManagementListResult,
  fetchAnnouncementManagementRows,
  normalizeAnnouncementManagementSortKey,
  sortAnnouncementManagementRows,
  type AnnouncementManagementRow,
  type AnnouncementManagementSortDir,
  type AnnouncementManagementSortKey,
} from "../../application/announcement_management_service";
import { useAuth } from "../../../shell/src/auth/presentation/hook/useCurrentMember";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 50;

export function useAnnouncementManagement() {
  const navigate = useNavigate();
  const { user, loading, currentMember, loadingMember } = useAuth();

  const [sourceRows, setSourceRows] = useState<AnnouncementManagementRow[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [page, setPage] = useState(DEFAULT_PAGE);
  const [perPage, setPerPage] = useState(DEFAULT_PER_PAGE);
  const [sortKey, setSortKey] =
    useState<AnnouncementManagementSortKey>("createdAt");
  const [sortDir, setSortDir] =
    useState<AnnouncementManagementSortDir>("desc");
  const [isResetting, setIsResetting] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const companyId = useMemo(() => {
    return String(currentMember?.companyId ?? user?.companyId ?? "").trim();
  }, [currentMember, user]);

  const isAuthLoading = loading || loadingMember;

  const load = useCallback(
    async (nextPage = page, nextPerPage = perPage) => {
      if (isAuthLoading) {
        return;
      }

      if (!companyId) {
        const empty = createEmptyAnnouncementManagementListResult(
          nextPage,
          nextPerPage,
        );

        setSourceRows(empty.rows);
        setTotalCount(empty.totalCount);
        setPage(empty.page);
        setPerPage(empty.perPage);
        return;
      }

      setIsLoading(true);

      try {
        const result = await fetchAnnouncementManagementRows({
          companyId,
          page: nextPage,
          perPage: nextPerPage,
        });

        setSourceRows(result.rows);
        setTotalCount(result.totalCount);
        setPage(result.page || nextPage);
        setPerPage(result.perPage || nextPerPage);
      } catch {
        const empty = createEmptyAnnouncementManagementListResult(
          nextPage,
          nextPerPage,
        );

        setSourceRows(empty.rows);
        setTotalCount(empty.totalCount);
        setPage(empty.page);
        setPerPage(empty.perPage);
      } finally {
        setIsLoading(false);
      }
    },
    [companyId, isAuthLoading, page, perPage],
  );

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
      await load(DEFAULT_PAGE, DEFAULT_PER_PAGE);
    } finally {
      setIsResetting(false);
    }
  }, [load]);

  const handleCreate = useCallback(() => {
    navigate("/sales/create");
  }, [navigate]);

  const handleRowClick = useCallback(
    (announcementId: string) => {
      const id = String(announcementId ?? "").trim();
      if (!id) return;

      navigate(`/sales/announcements/${encodeURIComponent(id)}`);
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