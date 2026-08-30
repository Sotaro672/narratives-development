// frontend/amol/src/features/inquiry/presentation/hooks/useInquiryListPage.ts

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  listMeInquiries,
  markInquiryAsRead,
  type InquiryListItem,
} from "../../api/inquiryApi";

import {
  fetchMyResaleChats,
  markMyResaleCommentsAsRead,
} from "../../../resale/api/resaleReviewApi";
import {
  updateResaleChatBadgeCount,
} from "../../../resale/presentation/resaleChatBadgeEvents";

import type {
  ResaleChatListItem as ResaleChatListItemDTO,
} from "../../../shared/types/resaleReview";

export type InquiryChatListItem = InquiryListItem & {
  chatKind: "inquiry";
};

export type ResaleChatListItem = ResaleChatListItemDTO & {
  chatKind: "resale";
};

export type ChatListItem =
  | InquiryChatListItem
  | ResaleChatListItem;

function getErrorMessage(caught: unknown, fallbackMessage: string): string {
  return caught instanceof Error ? caught.message : fallbackMessage;
}

function getComparableTime(value: string): number {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? 0 : date.getTime();
}

async function loadInquiryItems(signal?: AbortSignal): Promise<InquiryChatListItem[]> {
  const result = await listMeInquiries({
    page: 1,
    perPage: 100,
    signal,
  });

  if (signal?.aborted) {
    return [];
  }

  return result.items.map((inquiry) => ({
    ...inquiry,
    chatKind: "inquiry",
  }));
}

async function loadResaleItems(): Promise<ResaleChatListItem[]> {
  const result = await fetchMyResaleChats();

  return result.items.map((resale) => ({
    ...resale,
    chatKind: "resale",
  }));
}

export function useInquiryListPage() {
  const navigate = useNavigate();

  const [items, setItems] = useState<ChatListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [navigatingId, setNavigatingId] = useState<string | null>(null);
  const [error, setError] = useState("");

  const sortedItems = useMemo(() => {
    return [...items].sort((firstItem, secondItem) => {
      const firstTime = getComparableTime(firstItem.latestActivityAt);
      const secondTime = getComparableTime(secondItem.latestActivityAt);
      return secondTime - firstTime;
    });
  }, [items]);

  const loadChats = useCallback(async (signal?: AbortSignal) => {
    setLoading(true);
    setError("");

    try {
      const [inquiryItems, resaleItems] = await Promise.all([
        loadInquiryItems(signal),
        loadResaleItems(),
      ]);

      if (signal?.aborted) {
        return;
      }

      setItems([
        ...inquiryItems,
        ...resaleItems,
      ]);
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
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void loadChats(controller.signal);

    return () => {
      controller.abort();
    };
  }, [loadChats]);

  const handleOpenInquiryChat = useCallback(
    async (item: InquiryChatListItem) => {
      const inquiryId = item.id;
      let nextItem: InquiryChatListItem = item;

      if (item.unreadReplyCount > 0) {
        const updatedInquiry = await markInquiryAsRead(inquiryId);

        nextItem = {
          ...item,
          ...updatedInquiry,
          unreadReplyCount: 0,
          chatKind: "inquiry",
        };

        setItems((currentItems) =>
          currentItems.map((currentItem) => {
            if (
              currentItem.chatKind === "inquiry" &&
              currentItem.id === inquiryId
            ) {
              return nextItem;
            }

            return currentItem;
          }),
        );
      }

      navigate(`/chats/${encodeURIComponent(inquiryId)}`, {
        state: {
          inquiry: nextItem,
        },
      });
    },
    [navigate],
  );

  const handleOpenResaleChat = useCallback(
    async (item: ResaleChatListItem) => {
      const resaleId = item.resaleId;
      let nextItem: ResaleChatListItem = item;

      if (item.unreadCommentCount > 0) {
        const result = await markMyResaleCommentsAsRead({
          resaleId,
        });

        nextItem = {
          ...item,
          unreadCommentCount: 0,
          chatKind: "resale",
        };

        setItems((currentItems) =>
          currentItems.map((currentItem) => {
            if (
              currentItem.chatKind === "resale" &&
              currentItem.resaleId === resaleId
            ) {
              return nextItem;
            }

            return currentItem;
          }),
        );

        if (result.markedCount > 0) {
          updateResaleChatBadgeCount(-result.markedCount);
        }
      }

      navigate(`/chats/resales/${encodeURIComponent(resaleId)}`, {
        state: {
          source: "owner",
          resale: nextItem,
        },
      });
    },
    [navigate],
  );

  const handleOpenChat = useCallback(
    async (item: ChatListItem) => {
      if (navigatingId) {
        return;
      }

      const chatId =
        item.chatKind === "inquiry"
          ? item.id
          : item.resaleId;

      if (!chatId) {
        return;
      }

      setNavigatingId(chatId);
      setError("");

      try {
        if (item.chatKind === "inquiry") {
          await handleOpenInquiryChat(item);
          return;
        }

        await handleOpenResaleChat(item);
      } catch (caught) {
        setError(
          getErrorMessage(
            caught,
            "チャットを開く処理に失敗しました。",
          ),
        );
      } finally {
        setNavigatingId(null);
      }
    },
    [
      handleOpenInquiryChat,
      handleOpenResaleChat,
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