// frontend/amol/src/features/announcement/hooks/useAnnouncementsQuery.ts

import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  fetchMeAnnouncements,
  markMeAnnouncementRead,
} from "../api/announcementApi";

import type {
  AnnouncementListItem,
  AnnouncementListResult,
} from "../../shared/types/announcements";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 100;

type UseAnnouncementsQueryParams = {
  page?: number;
  perPage?: number;
  enabled?: boolean;
};

type MarkAnnouncementReadContext = {
  previousQueries: Array<
    readonly [
      readonly unknown[],
      AnnouncementListResult | undefined,
    ]
  >;
};

export const announcementQueryKeys = {
  all: ["announcements"] as const,

  me: () =>
    [...announcementQueryKeys.all, "me"] as const,

  lists: () =>
    [...announcementQueryKeys.me(), "list"] as const,

  list: (page: number, perPage: number) =>
    [
      ...announcementQueryKeys.lists(),
      {
        page,
        perPage,
      },
    ] as const,
};

function normalizePositiveInteger(
  value: number | undefined,
  fallback: number,
): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value)
  ) {
    return fallback;
  }

  const normalized = Math.floor(value);

  return normalized > 0 ? normalized : fallback;
}

function markAnnouncementAsRead(
  current: AnnouncementListResult | undefined,
  announcementId: string,
  readAt: string,
): AnnouncementListResult | undefined {
  if (!current) {
    return current;
  }

  let changed = false;

  const items = current.items.map(
    (
      item: AnnouncementListItem,
    ): AnnouncementListItem => {
      if (
        item.id !== announcementId ||
        item.isRead === true
      ) {
        return item;
      }

      changed = true;

      return {
        ...item,
        isRead: true,
        readAt: item.readAt ?? readAt,
      };
    },
  );

  if (!changed) {
    return current;
  }

  return {
    ...current,
    items,
  };
}

export function useAnnouncementsQuery(
  params: UseAnnouncementsQueryParams = {},
) {
  const page = normalizePositiveInteger(
    params.page,
    DEFAULT_PAGE,
  );

  const perPage = normalizePositiveInteger(
    params.perPage,
    DEFAULT_PER_PAGE,
  );

  const enabled = params.enabled ?? true;

  return useQuery({
    queryKey: announcementQueryKeys.list(
      page,
      perPage,
    ),

    queryFn: ({ signal }) =>
      fetchMeAnnouncements({
        page,
        perPage,
        signal,
      }),

    enabled,
  });
}

export function useMarkAnnouncementReadMutation() {
  const queryClient = useQueryClient();

  return useMutation<
    void,
    Error,
    string,
    MarkAnnouncementReadContext
  >({
    mutationFn: async (
      announcementId: string,
    ): Promise<void> => {
      const normalizedAnnouncementId =
        announcementId.trim();

      if (!normalizedAnnouncementId) {
        throw new Error(
          "announcementId が空のため既読化できません。",
        );
      }

      await markMeAnnouncementRead(
        normalizedAnnouncementId,
      );
    },

    onMutate: async (
      announcementId,
    ): Promise<MarkAnnouncementReadContext> => {
      const normalizedAnnouncementId =
        announcementId.trim();

      await queryClient.cancelQueries({
        queryKey: announcementQueryKeys.lists(),
      });

      const previousQueries =
        queryClient.getQueriesData<AnnouncementListResult>(
          {
            queryKey:
              announcementQueryKeys.lists(),
          },
        );

      if (normalizedAnnouncementId) {
        const readAt = new Date().toISOString();

        queryClient.setQueriesData<AnnouncementListResult>(
          {
            queryKey:
              announcementQueryKeys.lists(),
          },
          (current) =>
            markAnnouncementAsRead(
              current,
              normalizedAnnouncementId,
              readAt,
            ),
        );
      }

      return {
        previousQueries,
      };
    },

    onError: (
      _error,
      _announcementId,
      context,
    ) => {
      context?.previousQueries.forEach(
        ([queryKey, previousData]) => {
          queryClient.setQueryData(
            queryKey,
            previousData,
          );
        },
      );
    },

    onSettled: async () => {
      await queryClient.invalidateQueries({
        queryKey: announcementQueryKeys.lists(),
      });
    },
  });
}