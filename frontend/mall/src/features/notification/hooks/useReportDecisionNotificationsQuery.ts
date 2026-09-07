// frontend/mall/src/features/notification/hooks/useReportDecisionNotificationsQuery.ts

import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  fetchMeReportDecisionNotifications,
  markMeReportDecisionNotificationRead,
  type FetchMeReportDecisionNotificationsParams,
  type ReportDecisionNotification,
  type ReportDecisionNotificationPage,
} from "../infrastructure/reportDecisionNotificationApi";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;
const UNREAD_COUNT_PAGE = 1;
const UNREAD_COUNT_PER_PAGE = 1;

export type UseReportDecisionNotificationsQueryParams = {
  page?: number;
  perPage?: number;
  isRead?: boolean;
  enabled?: boolean;
};

export type UseReportDecisionNotificationUnreadCountQueryParams = {
  enabled?: boolean;
};

type ReportDecisionNotificationListKeyParams = {
  page: number;
  perPage: number;
  isRead?: boolean;
};

export const reportDecisionNotificationQueryKeys = {
  all: ["reportDecisionNotifications"] as const,

  me: () => [
    ...reportDecisionNotificationQueryKeys.all,
    "me",
  ] as const,

  lists: () => [
    ...reportDecisionNotificationQueryKeys.me(),
    "list",
  ] as const,

  list: (
    params: ReportDecisionNotificationListKeyParams,
  ) => [
    ...reportDecisionNotificationQueryKeys.lists(),
    params,
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

function createFetchParams(
  params: {
    page: number;
    perPage: number;
    isRead?: boolean;
  },
  signal?: AbortSignal,
): FetchMeReportDecisionNotificationsParams {
  return {
    page: params.page,
    perPage: params.perPage,
    ...(params.isRead !== undefined
      ? { isRead: params.isRead }
      : {}),
    ...(signal ? { signal } : {}),
  };
}

function replaceNotificationInPage(
  current: ReportDecisionNotificationPage | undefined,
  updated: ReportDecisionNotification,
): ReportDecisionNotificationPage | undefined {
  if (!current) {
    return current;
  }

  let changed = false;

  const items = current.items.map((notification) => {
    if (notification.id !== updated.id) {
      return notification;
    }

    changed = true;
    return updated;
  });

  if (!changed) {
    return current;
  }

  return {
    ...current,
    items,
  };
}

export function useReportDecisionNotificationsQuery(
  params: UseReportDecisionNotificationsQueryParams = {},
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
  const isRead = params.isRead;

  const keyParams: ReportDecisionNotificationListKeyParams = {
    page,
    perPage,
    ...(isRead !== undefined
      ? { isRead }
      : {}),
  };

  return useQuery({
    queryKey:
      reportDecisionNotificationQueryKeys.list(
        keyParams,
      ),

    queryFn: ({ signal }) =>
      fetchMeReportDecisionNotifications(
        createFetchParams(
          keyParams,
          signal,
        ),
      ),

    enabled,
  });
}

export function useReportDecisionNotificationUnreadCountQuery(
  params: UseReportDecisionNotificationUnreadCountQueryParams = {},
) {
  const enabled = params.enabled ?? true;

  const query =
    useReportDecisionNotificationsQuery({
      page: UNREAD_COUNT_PAGE,
      perPage: UNREAD_COUNT_PER_PAGE,
      isRead: false,
      enabled,
    });

  const unreadCount =
    typeof query.data?.totalCount === "number" &&
    Number.isFinite(query.data.totalCount)
      ? Math.max(
          0,
          Math.floor(query.data.totalCount),
        )
      : 0;

  return {
    ...query,
    unreadCount,
  };
}

export function useMarkReportDecisionNotificationReadMutation() {
  const queryClient = useQueryClient();

  return useMutation<
    ReportDecisionNotification,
    Error,
    string
  >({
    mutationFn: async (
      notificationId: string,
    ): Promise<ReportDecisionNotification> => {
      if (!notificationId) {
        throw new Error(
          "notificationId が空のため既読化できません。",
        );
      }

      return markMeReportDecisionNotificationRead(
        notificationId,
      );
    },

    onSuccess: (updated) => {
      queryClient.setQueriesData<
        ReportDecisionNotificationPage
      >(
        {
          queryKey:
            reportDecisionNotificationQueryKeys.lists(),
        },
        (current) =>
          replaceNotificationInPage(
            current,
            updated,
          ),
      );
    },

    onSettled: async () => {
      await queryClient.invalidateQueries({
        queryKey:
          reportDecisionNotificationQueryKeys.lists(),
      });
    },
  });
}

export default useReportDecisionNotificationsQuery;