// frontend/admin/shell/src/features/announcement/presentation/hooks/useAnnouncementDetail.ts

import { useCallback, useEffect, useState } from "react";

import {
  getAnnouncementDetail,
  type AnnouncementDetail,
} from "../../infrastructure/announcementApi";

export function useAnnouncementDetail(
  companyId: string | undefined,
  announcementId: string | undefined,
) {
  const [announcement, setAnnouncement] =
    useState<AnnouncementDetail | null>(null);
  const [loading, setLoading] = useState(
    Boolean(companyId?.trim() && announcementId?.trim()),
  );
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedCompanyId = companyId?.trim() ?? "";
    const normalizedAnnouncementId = announcementId?.trim() ?? "";

    if (!normalizedCompanyId || !normalizedAnnouncementId) {
      setAnnouncement(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await getAnnouncementDetail(
        normalizedCompanyId,
        normalizedAnnouncementId,
      );
      setAnnouncement(result);
    } catch (cause) {
      setAnnouncement(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "告知詳細の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [companyId, announcementId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    announcement,
    loading,
    error,
    reload,
  };
}