// frontend/amol/src/features/inquiry/presentation/hooks/useInquiryListPage.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import {
  listInquiryReplies,
  listMeInquiries,
  markInquiryAsRead,
  type Inquiry,
  type InquiryReply,
} from "../../api/inquiryApi";

export type InquiryChatListItem = Inquiry & {
  chatKind: "inquiry";
  readAt?: string | null;

  productName?: string | null;
  tokenName?: string | null;
  brandName?: string | null;
  avatarName?: string | null;
  senderName?: string | null;

  latestMessage?: string | null;
  latestReplyContent?: string | null;

  replies: InquiryReply[];
};

function getErrorMessage(
  caught: unknown,
  fallbackMessage: string,
): string {
  return caught instanceof Error
    ? caught.message
    : fallbackMessage;
}

function getComparableTime(
  value?: string | null,
): number {
  if (!value) {
    return 0;
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return 0;
  }

  return date.getTime();
}

function getLatestReply(
  replies: InquiryReply[],
): InquiryReply | null {
  if (
    !Array.isArray(replies) ||
    replies.length === 0
  ) {
    return null;
  }

  return (
    [...replies].sort(
      (firstReply, secondReply) => {
        const firstTime =
          getComparableTime(
            firstReply.updatedAt ??
              firstReply.createdAt,
          );

        const secondTime =
          getComparableTime(
            secondReply.updatedAt ??
              secondReply.createdAt,
          );

        return secondTime - firstTime;
      },
    )[0] ?? null
  );
}

function getLatestActivityAt(
  item: InquiryChatListItem,
): string | null | undefined {
  const latestReply =
    getLatestReply(item.replies);

  return (
    latestReply?.updatedAt ||
    latestReply?.createdAt ||
    item.updatedAt ||
    item.createdAt
  );
}

async function loadInquiryItems(
  signal?: AbortSignal,
): Promise<InquiryChatListItem[]> {
  const result = await listMeInquiries({
    page: 1,
    perPage: 100,
    signal,
  });

  if (signal?.aborted) {
    return [];
  }

  return Promise.all(
    result.items.map(
      async (
        inquiry,
      ): Promise<InquiryChatListItem> => {
        if (!inquiry.id) {
          return {
            ...inquiry,
            chatKind: "inquiry",
            replies: [],
          };
        }

        try {
          const replies =
            await listInquiryReplies(
              inquiry.id,
            );

          return {
            ...inquiry,
            chatKind: "inquiry",
            replies,
          };
        } catch {
          return {
            ...inquiry,
            chatKind: "inquiry",
            replies: [],
          };
        }
      },
    ),
  );
}

export function useInquiryListPage() {
  const navigate = useNavigate();

  const [
    items,
    setItems,
  ] = useState<
    InquiryChatListItem[]
  >([]);

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    navigatingId,
    setNavigatingId,
  ] = useState<string | null>(
    null,
  );

  const [
    error,
    setError,
  ] = useState("");

  const sortedItems = useMemo(() => {
    return [...items].sort(
      (firstItem, secondItem) => {
        const firstTime =
          getComparableTime(
            getLatestActivityAt(
              firstItem,
            ),
          );

        const secondTime =
          getComparableTime(
            getLatestActivityAt(
              secondItem,
            ),
          );

        return secondTime - firstTime;
      },
    );
  }, [items]);

  const loadChats = useCallback(
    async (
      signal?: AbortSignal,
    ) => {
      setLoading(true);
      setError("");

      try {
        const nextItems =
          await loadInquiryItems(
            signal,
          );

        if (signal?.aborted) {
          return;
        }

        setItems(nextItems);
      } catch (caught) {
        if (signal?.aborted) {
          return;
        }

        setItems([]);

        setError(
          getErrorMessage(
            caught,
            "チャット一覧の取得に失敗しました。",
          ),
        );
      } finally {
        if (!signal?.aborted) {
          setLoading(false);
        }
      }
    },
    [],
  );

  useEffect(() => {
    const controller =
      new AbortController();

    void loadChats(
      controller.signal,
    );

    return () => {
      controller.abort();
    };
  }, [loadChats]);

  const handleOpenChat =
    useCallback(
      async (
        item:
          InquiryChatListItem,
      ) => {
        const inquiryId =
          item.id;

        if (
          !inquiryId ||
          navigatingId
        ) {
          return;
        }

        setNavigatingId(
          inquiryId,
        );

        setError("");

        try {
          const now =
            new Date().toISOString();

          let nextItem = item;

          if (
            item.isRead === false
          ) {
            const updatedInquiry =
              await markInquiryAsRead(
                inquiryId,
              );

            nextItem = {
              ...item,
              ...(updatedInquiry ?? {}),
              chatKind:
                "inquiry",
              isRead: true,
              readAt:
                item.readAt ?? now,
              replies:
                item.replies,
            };

            setItems(
              (currentItems) =>
                currentItems.map(
                  (currentItem) =>
                    currentItem.id ===
                    inquiryId
                      ? nextItem
                      : currentItem,
                ),
            );
          }

          navigate(
            `/chats/${encodeURIComponent(
              inquiryId,
            )}`,
            {
              state: {
                inquiry: {
                  ...nextItem,
                  isRead: true,
                  readAt:
                    nextItem.readAt ??
                    now,
                },
                replies:
                  nextItem.replies,
              },
            },
          );
        } catch (caught) {
          setError(
            getErrorMessage(
              caught,
              "チャットを開く処理に失敗しました。",
            ),
          );
        } finally {
          setNavigatingId(
            null,
          );
        }
      },
      [
        navigate,
        navigatingId,
      ],
    );

  return {
    items,
    sortedItems,

    loading,
    navigatingId,
    error,

    loadChats,
    handleOpenChat,
  };
}