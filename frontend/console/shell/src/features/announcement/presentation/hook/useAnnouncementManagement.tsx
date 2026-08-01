// frontend/console/shell/src/features/announcement/presentation/hook/useAnnouncementManagement.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  useNavigate,
} from "react-router-dom";

import {
  useAuthContext,
} from "../../../../auth/application/AuthContext";
import {
  fetchAnnouncementManagementRows,
  normalizeAnnouncementManagementSortKey,
  sortAnnouncementManagementRows,
  type AnnouncementManagementRow,
  type AnnouncementManagementSortDir,
  type AnnouncementManagementSortKey,
} from "../../application/announcement_management_service";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 50;

export function useAnnouncementManagement() {
  const navigate = useNavigate();

  const {
    loading,
    currentMember,
    loadingMember,
  } = useAuthContext();

  const [
    sourceRows,
    setSourceRows,
  ] = useState<
    AnnouncementManagementRow[]
  >([]);

  const [
    sortKey,
    setSortKey,
  ] =
    useState<AnnouncementManagementSortKey>(
      "createdAt",
    );

  const [
    sortDir,
    setSortDir,
  ] =
    useState<AnnouncementManagementSortDir>(
      "desc",
    );

  const [
    isResetting,
    setIsResetting,
  ] = useState(false);

  const [
    isLoading,
    setIsLoading,
  ] = useState(false);

  const companyId = useMemo(
    () =>
      String(
        currentMember?.companyId ?? "",
      ).trim(),
    [currentMember?.companyId],
  );

  const isAuthLoading =
    loading || loadingMember;

  const load = useCallback(
    async () => {
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
    },
    [
      companyId,
      isAuthLoading,
    ],
  );

  useEffect(() => {
    void load();
  }, [load]);

  const rows = useMemo(
    () =>
      sortAnnouncementManagementRows(
        sourceRows,
        sortKey,
        sortDir,
      ),
    [
      sourceRows,
      sortKey,
      sortDir,
    ],
  );

  const handleChangeSort =
    useCallback(
      (nextKey: string) => {
        const normalizedKey =
          normalizeAnnouncementManagementSortKey(
            nextKey,
          );

        setSortKey(
          (previousKey) => {
            if (
              previousKey ===
              normalizedKey
            ) {
              setSortDir(
                (previousDirection) =>
                  previousDirection ===
                  "asc"
                    ? "desc"
                    : "asc",
              );

              return previousKey;
            }

            setSortDir("asc");

            return normalizedKey;
          },
        );
      },
      [],
    );

  const handleReset =
    useCallback(async () => {
      setIsResetting(true);

      try {
        setSortKey("createdAt");
        setSortDir("desc");

        await load();
      } finally {
        setIsResetting(false);
      }
    }, [load]);

  const handleCreate =
    useCallback(() => {
      navigate("/sales/create");
    }, [navigate]);

  const handleRowClick =
    useCallback(
      (
        announcementId: string,
      ) => {
        const id = String(
          announcementId ?? "",
        ).trim();

        if (!id) {
          return;
        }

        navigate(
          `/sales/announcements/${encodeURIComponent(
            id,
          )}`,
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