// frontend/console/shell/src/features/announcement/presentation/hook/useAnnouncementManagement.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  fetchAnnouncementManagementRows,
  normalizeAnnouncementManagementSortKey,
  sortAnnouncementManagementRows,
  type AnnouncementManagementRow,
  type AnnouncementManagementSortDir,
  type AnnouncementManagementSortKey,
} from "../../application/announcement_management_service";
import { useAuth } from "../../../../auth/presentation/hook/useCurrentMember";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 50;

export function useAnnouncementManagement() {
  const navigate = useNavigate();
  const { user, loading, currentMember, loadingMember } = useAuth();

  const [sourceRows, setSourceRows] = useState<AnnouncementManagementRow[]>([]);
  const [sortKey, setSortKey] =
    useState<AnnouncementManagementSortKey>("createdAt");
  const [sortDir, setSortDir] =
    useState<AnnouncementManagementSortDir>("desc");
  const [isResetting, setIsResetting] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const companyId = useMemo(() => {
    return String(
      currentMember?.companyId ??
        user?.companyId ??
        "",
    ).trim();
  }, [currentMember, user]);

  const isAuthLoading = loading || loadingMember;

  const load = useCallback(async () => {
    if (isAuthLoading) {
      return;
    }

    if (!companyId) {
      setSourceRows([]);
      return;
    }

    setIsLoading(true);

    try {
      const result =
        await fetchAnnouncementManagementRows({
          companyId,
          page: DEFAULT_PAGE,
          perPage: DEFAULT_PER_PAGE,
        });

      setSourceRows(result.rows);
    } catch {
      setSourceRows([]);
    } finally {
      setIsLoading(false);
    }
  }, [companyId, isAuthLoading]);

  useEffect(() => {
    void load();
  }, [load]);

  const rows = useMemo(() => {
    return sortAnnouncementManagementRows(
      sourceRows,
      sortKey,
      sortDir,
    );
  }, [sourceRows, sortDir, sortKey]);

  const handleChangeSort = useCallback(
    (nextKey: string) => {
      const normalizedKey =
        normalizeAnnouncementManagementSortKey(
          nextKey,
        );

      setSortKey((prevKey) => {
        if (prevKey === normalizedKey) {
          setSortDir((prevDir) =>
            prevDir === "asc" ? "desc" : "asc",
          );

          return prevKey;
        }

        setSortDir("asc");

        return normalizedKey;
      });
    },
    [],
  );

  const handleReset = useCallback(async () => {
    setIsResetting(true);

    try {
      setSortKey("createdAt");
      setSortDir("desc");

      await load();
    } finally {
      setIsResetting(false);
    }
  }, [load]);

  const handleCreate = useCallback(() => {
    navigate("/sales/create");
  }, [navigate]);

  const handleRowClick = useCallback(
    (announcementId: string) => {
      const id = String(
        announcementId ?? "",
      ).trim();

      if (!id) {
        return;
      }

      navigate(
        `/sales/announcements/${encodeURIComponent(id)}`,
      );
    },
    [navigate],
  );

  return {
    rows,
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