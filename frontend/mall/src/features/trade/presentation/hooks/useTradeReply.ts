// frontend/mall/src/features/trade/presentation/hooks/useTradeReply.ts

import {
  type Dispatch,
  type SetStateAction,
  useCallback,
  useState,
} from "react";

import type { TradeDetail } from "../../../shared/types/trade";
import { createTradeMessage } from "../../infrastructure/tradeApi";
import {
  getErrorMessage,
  sortTradeMessages,
} from "../util/tradeChatDetail";

type UseTradeReplyInput = {
  tradeId: string;
  trade: TradeDetail | null;
  setTrade: Dispatch<SetStateAction<TradeDetail | null>>;
  loading: boolean;
  blocked: boolean;
};

export function useTradeReply({
  tradeId,
  trade,
  setTrade,
  loading,
  blocked,
}: UseTradeReplyInput) {
  const [open, setOpen] = useState(false);
  const [content, setContentState] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const canSubmit = /\S/u.test(content);

  const actionDisabled =
    loading ||
    !trade ||
    trade.status !== "active" ||
    trade.isCancelled ||
    submitting ||
    blocked;

  const setContent = useCallback((value: string) => {
    setContentState(value);
    setError("");
  }, []);

  const openModal = useCallback(() => {
    if (actionDisabled) {
      return;
    }

    setError("");
    setOpen(true);
  }, [actionDisabled]);

  const closeModal = useCallback(() => {
    if (submitting) {
      return;
    }

    setOpen(false);
    setContentState("");
    setError("");
  }, [submitting]);

  const submit = useCallback(async (): Promise<void> => {
    if (
      submitting ||
      blocked ||
      !trade ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !tradeId
    ) {
      return;
    }

    if (!/\S/u.test(content)) {
      setError("メッセージを入力してください。");
      return;
    }

    setSubmitting(true);
    setError("");

    try {
      const created = await createTradeMessage({
        tradeId,
        content,
      });

      setTrade((currentTrade) => {
        if (!currentTrade) {
          return currentTrade;
        }

        return {
          ...currentTrade,
          messages: sortTradeMessages([
            ...currentTrade.messages.filter(
              (message) => message.id !== created.id,
            ),
            created,
          ]),
          lastMessageAt: created.createdAt,
          updatedAt: created.createdAt,
        };
      });

      setOpen(false);
      setContentState("");
      setError("");
    } catch (caught) {
      setError(
        getErrorMessage(
          caught,
          "メッセージの送信に失敗しました。",
        ),
      );
    } finally {
      setSubmitting(false);
    }
  }, [
    blocked,
    content,
    setTrade,
    submitting,
    trade,
    tradeId,
  ]);

  return {
    open,
    content,
    error,
    submitting,
    canSubmit,
    actionDisabled,
    openModal,
    closeModal,
    setContent,
    submit,
  };
}

export default useTradeReply;