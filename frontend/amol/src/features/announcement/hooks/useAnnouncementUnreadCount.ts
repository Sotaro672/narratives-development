import { useEffect, useMemo, useState } from "react";

import { fetchMeAnnouncements } from "../api/announcementApi";
import type { AnnouncementListItem } from "../types";

type UseAnnouncementUnreadCountParams = {
  enabled?: boolean;
};

type UseAnnouncementUnreadCountResult = {
  unreadCount: number;
  loading: boolean;
  error: Error | null;
};

export function useAnnouncementUnreadCount(
  params: UseAnnouncementUnreadCountParams = {},
): UseAnnouncementUnreadCountResult {
  const enabled = params.enabled ?? true;

  const [items, setItems] = useState<AnnouncementListItem[]>([]);
  const [loading, setLoading] = useState<boolean>(enabled);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!enabled) {
      setItems([]);
      setLoading(false);
      setError(null);
      return;
    }

    const controller = new AbortController();

    setLoading(true);
    setError(null);

    fetchMeAnnouncements({
      page: 1,
      perPage: 100,
      signal: controller.signal,
    })
      .then((result) => {
        setItems(result.items);
      })
      .catch((caught) => {
        if (controller.signal.aborted) {
          return;
        }

        setError(
          caught instanceof Error
            ? caught
            : new Error("failed to fetch announcement unread count"),
        );
        setItems([]);
      })
      .finally(() => {
        if (controller.signal.aborted) {
          return;
        }

        setLoading(false);
      });

    return () => {
      controller.abort();
    };
  }, [enabled]);

  const unreadCount = useMemo(() => {
    return items.filter((item) => item.isRead === false).length;
  }, [items]);

  return {
    unreadCount,
    loading,
    error,
  };
}