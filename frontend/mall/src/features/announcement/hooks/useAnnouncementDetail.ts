// frontend/mall/src/features/announcement/hooks/useAnnouncementDetail.ts

import { useEffect, useMemo, useRef } from "react";

import type { AnnouncementListItem } from "../../shared/types/announcements";
import {
  useAnnouncementsQuery,
  useMarkAnnouncementReadMutation,
} from "./useAnnouncementsQuery";

type UseAnnouncementDetailParams = {
  announcementId?: string;
  initialAnnouncement?: AnnouncementListItem | null;
  enabled?: boolean;
};

export type UseAnnouncementDetailResult = {
  announcementId: string;
  announcement: AnnouncementListItem | null;
  loading: boolean;
  error: string;
  notFound: boolean;
};

export function useAnnouncementDetail({
  announcementId = "",
  initialAnnouncement = null,
  enabled = true,
}: UseAnnouncementDetailParams): UseAnnouncementDetailResult {
  const effectiveAnnouncementId =
    announcementId ||
    initialAnnouncement?.id ||
    "";

  const validInitialAnnouncement = useMemo(() => {
    if (
      !initialAnnouncement ||
      initialAnnouncement.id !== effectiveAnnouncementId
    ) {
      return null;
    }

    return initialAnnouncement;
  }, [
    effectiveAnnouncementId,
    initialAnnouncement,
  ]);

  const announcementsQuery = useAnnouncementsQuery({
    page: 1,
    perPage: 100,
    enabled:
      enabled &&
      Boolean(effectiveAnnouncementId),
  });

  const markReadMutation =
    useMarkAnnouncementReadMutation();

  const announcementFromQuery = useMemo(() => {
    if (!announcementsQuery.data) {
      return null;
    }

    return (
      announcementsQuery.data.items.find(
        (item) =>
          item.id === effectiveAnnouncementId,
      ) ?? null
    );
  }, [
    announcementsQuery.data,
    effectiveAnnouncementId,
  ]);

  const announcement =
    announcementsQuery.data !== undefined
      ? announcementFromQuery
      : validInitialAnnouncement;

  const markedReadRef = useRef<string>("");

  useEffect(() => {
    if (
      !enabled ||
      !effectiveAnnouncementId ||
      !announcement
    ) {
      return;
    }

    if (announcement.isRead === true) {
      markedReadRef.current =
        effectiveAnnouncementId;
      return;
    }

    if (
      markedReadRef.current ===
      effectiveAnnouncementId
    ) {
      return;
    }

    markedReadRef.current =
      effectiveAnnouncementId;

    markReadMutation.mutate(
      effectiveAnnouncementId,
    );
  }, [
    announcement,
    effectiveAnnouncementId,
    enabled,
    markReadMutation,
  ]);

  const queryError =
    enabled &&
    announcementsQuery.error instanceof Error
      ? announcementsQuery.error.message
      : enabled &&
          announcementsQuery.error
        ? "お知らせの取得に失敗しました"
        : "";

  const mutationError =
    enabled &&
    markReadMutation.error instanceof Error
      ? markReadMutation.error.message
      : enabled &&
          markReadMutation.error
        ? "お知らせの既読化に失敗しました"
        : "";

  const loading =
    enabled &&
    Boolean(effectiveAnnouncementId) &&
    announcementsQuery.isPending &&
    !validInitialAnnouncement;

  const notFound =
    enabled &&
    (
      !effectiveAnnouncementId ||
      (
        announcementsQuery.isSuccess &&
        !announcement
      )
    );

  const error =
    mutationError ||
    queryError ||
    (
      notFound
        ? "お知らせが見つかりません。"
        : ""
    );

  return {
    announcementId: effectiveAnnouncementId,
    announcement,
    loading,
    error,
    notFound,
  };
}

export default useAnnouncementDetail;