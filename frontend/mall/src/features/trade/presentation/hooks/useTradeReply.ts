// frontend/mall/src/features/trade/presentation/hooks/useTradeReply.ts

import {
  type Dispatch,
  type SetStateAction,
  useCallback,
  useState,
} from "react";

import type { TradeDetail } from "../../../shared/types/trade";
import { createTradeMessage } from "../../infrastructure/tradeApi";
import { uploadTradeMessageImage } from "../../infrastructure/tradeMessageImageApi";
import {
  getErrorMessage,
  sortTradeMessages,
} from "../util/tradeChatDetail";

const MAX_MESSAGE_IMAGES = 10;
const MAX_MESSAGE_IMAGE_SIZE_BYTES = 20 * 1024 * 1024;

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
  const [content, setContentState] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const canSubmit =
    /\S/u.test(content) ||
    files.length > 0;

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

  const addFiles = useCallback((nextFiles: File[]) => {
    const imageFiles = nextFiles.filter((file) =>
      file.type.startsWith("image/"),
    );

    if (imageFiles.length !== nextFiles.length) {
      setError("画像ファイルのみ添付できます。");
      return;
    }

    const oversizedFile = imageFiles.find(
      (file) =>
        file.size > MAX_MESSAGE_IMAGE_SIZE_BYTES,
    );

    if (oversizedFile) {
      setError("画像は1枚20MB以下にしてください。");
      return;
    }

    setFiles((currentFiles) => {
      const remainingCount = Math.max(
        MAX_MESSAGE_IMAGES - currentFiles.length,
        0,
      );

      if (remainingCount === 0) {
        setError(
          `画像は最大${MAX_MESSAGE_IMAGES}枚まで添付できます。`,
        );
        return currentFiles;
      }

      const filesToAdd = imageFiles.slice(
        0,
        remainingCount,
      );

      if (imageFiles.length > remainingCount) {
        setError(
          `画像は最大${MAX_MESSAGE_IMAGES}枚まで添付できます。`,
        );
      } else {
        setError("");
      }

      return [
        ...currentFiles,
        ...filesToAdd,
      ];
    });
  }, []);

  const removeFile = useCallback((index: number) => {
    setFiles((currentFiles) =>
      currentFiles.filter(
        (_file, currentIndex) =>
          currentIndex !== index,
      ),
    );
    setError("");
  }, []);

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

    const normalizedContent = content.trim();

    if (
      !normalizedContent &&
      files.length === 0
    ) {
      setError(
        "メッセージまたは画像を入力してください。",
      );
      return;
    }

    setSubmitting(true);
    setError("");

    try {
      const images = await Promise.all(
        files.map((file) =>
          uploadTradeMessageImage({
            tradeId,
            file,
          }),
        ),
      );

      const created = await createTradeMessage({
        tradeId,
        content: normalizedContent || undefined,
        images,
      });

      setTrade((currentTrade) => {
        if (!currentTrade) {
          return currentTrade;
        }

        return {
          ...currentTrade,
          messages: sortTradeMessages([
            ...currentTrade.messages.filter(
              (message) =>
                message.id !== created.id,
            ),
            created,
          ]),
          lastMessageAt: created.createdAt,
          updatedAt: created.createdAt,
        };
      });

      setContentState("");
      setFiles([]);
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
    files,
    setTrade,
    submitting,
    trade,
    tradeId,
  ]);

  return {
    content,
    files,
    error,
    submitting,
    canSubmit,
    actionDisabled,
    setContent,
    addFiles,
    removeFile,
    submit,
  };
}

export default useTradeReply;