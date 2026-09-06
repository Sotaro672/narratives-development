// frontend/mall/src/features/notification/hooks/useReviewReportDecisionNotificationsQuery.ts

import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  fetchMeReviewReportDecisionNotifications,
  markMeReviewReportDecisionNotificationRead,
  type FetchMeReviewReportDecisionNotificationsParams,
  type ReviewReportDecisionNotification,
  type ReviewReportDecisionNotificationPage,
} from "../infrastructure/reviewReportDecisionNotificationApi";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;
const UNREAD_COUNT_PAGE = 1;
const UNREAD_COUNT_PER_PAGE = 1;

export type UseReviewReportDecisionNotificationsQueryParams = {
  page?: number;
  perPage?: number;
  isRead?: boolean;
  enabled?: boolean;
};

export type UseReviewReportDecisionNotificationUnreadCountQueryParams = {
  enabled?: boolean;
};

type ReviewReportDecisionNotificationListKeyParams = {
  page: number;
  perPage: number;
  isRead?: boolean;
};

export const reviewReportDecisionNotificationQueryKeys = {
  all: ["reviewReportDecisionNotifications"] as const,

  me: () =>
    [
      ...reviewReportDecisionNotificationQueryKeys.all,
      "me",
    ] as const,

  lists: () =>
    [
      ...reviewReportDecisionNotificationQueryKeys.me(),
      "list",
    ] as const,

  list: (
    params: ReviewReportDecisionNotificationListKeyParams,
  ) =>
    [
      ...reviewReportDecisionNotificationQueryKeys.lists(),
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
): FetchMeReviewReportDecisionNotificationsParams {
  return {
    page: params.page,
    perPage: params.perPage,
    ...(params.isRead !== undefined
      ? { isRead: params.isRead }
      : {}),
    ...(signal
      ? { signal }
      : {}),
  };
}

function replaceNotificationInPage(
  current: ReviewReportDecisionNotificationPage | undefined,
  updated: ReviewReportDecisionNotification,
): ReviewReportDecisionNotificationPage | undefined {
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

export function useReviewReportDecisionNotificationsQuery(
  params: UseReviewReportDecisionNotificationsQueryParams = {},
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

  const keyParams: ReviewReportDecisionNotificationListKeyParams = {
    page,
    perPage,
    ...(isRead !== undefined
      ? { isRead }
      : {}),
  };

  return useQuery({
    queryKey:
      reviewReportDecisionNotificationQueryKeys.list(
        keyParams,
      ),

    queryFn: ({ signal }) =>
      fetchMeReviewReportDecisionNotifications(
        createFetchParams(
          keyParams,
          signal,
        ),
      ),

    enabled,
  });
}

export function useReviewReportDecisionNotificationUnreadCountQuery(
  params: UseReviewReportDecisionNotificationUnreadCountQueryParams = {},
) {
  const enabled = params.enabled ?? true;

  const query =
    useReviewReportDecisionNotificationsQuery({
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

export function useMarkReviewReportDecisionNotificationReadMutation() {
  const queryClient = useQueryClient();

  return useMutation<
    ReviewReportDecisionNotification,
    Error,
    string
  >({
    mutationFn: async (
      notificationId: string,
    ): Promise<ReviewReportDecisionNotification> => {
      if (!notificationId) {
        throw new Error(
          "notificationId が空のため既読化できません。",
        );
      }

      return markMeReviewReportDecisionNotificationRead(
        notificationId,
      );
    },

    onSuccess: (updated) => {
      queryClient.setQueriesData<
        ReviewReportDecisionNotificationPage
      >(
        {
          queryKey:
            reviewReportDecisionNotificationQueryKeys.lists(),
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
          reviewReportDecisionNotificationQueryKeys.lists(),
      });
    },
  });
}

export default useReviewReportDecisionNotificationsQuery;