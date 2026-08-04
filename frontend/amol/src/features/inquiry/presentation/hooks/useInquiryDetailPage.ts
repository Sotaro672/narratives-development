// frontend/amol/src/features/inquiry/presentation/hooks/useInquiryDetailPage.ts

import {
  type ChangeEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  useLocation,
  useParams,
} from "react-router-dom";

import {
  closeInquiry,
  getInquiry,
  listInquiryReplies,
  markInquiryAsRead,
  replyInquiry,
  uploadReplyImage,
  type Inquiry,
  type InquiryReply,
} from "../../api/inquiryApi";

type InquiryDetailLocationState = {
  inquiry?: Inquiry | null;
  replies?: InquiryReply[] | null;
};

type InquiryDetailRouteParams = {
  inquiryId?: string;
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

function getInquiryTitle(
  inquiry: Inquiry | null,
): string {
  const subject = inquiry?.subject?.trim() ?? "";

  return subject || "チャット詳細";
}

export function useInquiryDetailPage() {
  const { inquiryId } =
    useParams<InquiryDetailRouteParams>();

  const location = useLocation();

  const locationState =
    location.state as InquiryDetailLocationState | null;

  const [inquiry, setInquiry] =
    useState<Inquiry | null>(
      locationState?.inquiry ?? null,
    );

  const [replies, setReplies] =
    useState<InquiryReply[]>(
      Array.isArray(locationState?.replies)
        ? locationState.replies
        : [],
    );

  const [loading, setLoading] =
    useState<boolean>(
      !locationState?.inquiry,
    );

  const [error, setError] =
    useState("");

  const [
    isReplyModalOpen,
    setIsReplyModalOpen,
  ] = useState(false);

  const [
    replyContent,
    setReplyContent,
  ] = useState("");

  const [
    replyFiles,
    setReplyFiles,
  ] = useState<File[]>([]);

  const [
    replyError,
    setReplyError,
  ] = useState("");

  const [
    postingReply,
    setPostingReply,
  ] = useState(false);

  const [
    closingInquiry,
    setClosingInquiry,
  ] = useState(false);

  const [
    closeError,
    setCloseError,
  ] = useState("");

  const canSubmitReply =
    replyContent.trim() !== "" ||
    replyFiles.length > 0;

  const sortedReplies = useMemo(() => {
    return [...replies].sort(
      (firstReply, secondReply) => {
        const firstTime = getComparableTime(
          firstReply.createdAt ??
            firstReply.updatedAt,
        );

        const secondTime = getComparableTime(
          secondReply.createdAt ??
            secondReply.updatedAt,
        );

        return firstTime - secondTime;
      },
    );
  }, [replies]);

  const loadThread = useCallback(async () => {
    if (!inquiryId) {
      setInquiry(null);
      setReplies([]);
      setError(
        "問い合わせIDが見つかりません。",
      );
      setLoading(false);

      return;
    }

    setLoading(true);
    setError("");
    setCloseError("");

    try {
      const [
        nextInquiry,
        nextReplies,
        updatedInquiry,
      ] = await Promise.all([
        getInquiry(inquiryId),
        listInquiryReplies(inquiryId),
        markInquiryAsRead(inquiryId),
      ]);

      setInquiry(
        updatedInquiry ?? nextInquiry,
      );

      setReplies(nextReplies);
    } catch (caught) {
      setInquiry(null);
      setReplies([]);

      setError(
        getErrorMessage(
          caught,
          "チャット内容の取得に失敗しました。",
        ),
      );
    } finally {
      setLoading(false);
    }
  }, [inquiryId]);

  useEffect(() => {
    void loadThread();
  }, [loadThread]);

  useEffect(() => {
    if (!isReplyModalOpen) {
      return;
    }

    const previousOverflow =
      document.body.style.overflow;

    const previousTouchAction =
      document.body.style.touchAction;

    document.body.style.overflow =
      "hidden";

    document.body.style.touchAction =
      "none";

    return () => {
      document.body.style.overflow =
        previousOverflow;

      document.body.style.touchAction =
        previousTouchAction;
    };
  }, [isReplyModalOpen]);

  const openReplyModal =
    useCallback(() => {
      setReplyError("");
      setIsReplyModalOpen(true);
    }, []);

  const closeReplyModal =
    useCallback(() => {
      if (postingReply) {
        return;
      }

      setIsReplyModalOpen(false);
      setReplyContent("");
      setReplyFiles([]);
      setReplyError("");
    }, [postingReply]);

  const handleReplyFilesChange =
    useCallback(
      (
        event:
          ChangeEvent<HTMLInputElement>,
      ) => {
        const selectedFiles =
          Array.from(
            event.target.files ?? [],
          ).filter((file) =>
            file.type.startsWith(
              "image/",
            ),
          );

        if (
          selectedFiles.length > 0
        ) {
          setReplyFiles(
            (currentFiles) => [
              ...currentFiles,
              ...selectedFiles,
            ],
          );
        }

        event.target.value = "";
      },
      [],
    );

  const removeReplyFile =
    useCallback(
      (index: number) => {
        setReplyFiles(
          (currentFiles) =>
            currentFiles.filter(
              (
                _file,
                currentIndex,
              ) =>
                currentIndex !==
                index,
            ),
        );
      },
      [],
    );

  const submitReply =
    useCallback(async () => {
      if (postingReply) {
        return;
      }

      const content =
        replyContent.trim();

      if (
        !content &&
        replyFiles.length === 0
      ) {
        setReplyError(
          "本文または画像を入力してください。",
        );

        return;
      }

      if (!inquiryId) {
        setReplyError(
          "問い合わせIDが見つかりません。",
        );

        return;
      }

      setPostingReply(true);
      setReplyError("");

      try {
        const images =
          await Promise.all(
            replyFiles.map(
              (file) =>
                uploadReplyImage({
                  inquiryId,
                  file,
                }),
            ),
          );

        const createdReply =
          await replyInquiry(
            inquiryId,
            {
              content,
              images,
            },
          );

        if (createdReply) {
          setReplies(
            (currentReplies) => [
              ...currentReplies,
              createdReply,
            ],
          );
        } else {
          const nextReplies =
            await listInquiryReplies(
              inquiryId,
            );

          setReplies(nextReplies);
        }

        setIsReplyModalOpen(false);
        setReplyContent("");
        setReplyFiles([]);
        setReplyError("");
      } catch (caught) {
        setReplyError(
          getErrorMessage(
            caught,
            "送信に失敗しました。",
          ),
        );
      } finally {
        setPostingReply(false);
      }
    }, [
      inquiryId,
      postingReply,
      replyContent,
      replyFiles,
    ]);

  const handleCloseInquiry =
    useCallback(async () => {
      if (
        !inquiryId ||
        closingInquiry
      ) {
        return;
      }

      setClosingInquiry(true);
      setCloseError("");

      try {
        const closedInquiry =
          await closeInquiry(inquiryId);

        setInquiry(
          (currentInquiry) =>
            closedInquiry ??
            (currentInquiry
              ? {
                  ...currentInquiry,
                  status: "closed",
                }
              : currentInquiry),
        );
      } catch (caught) {
        setCloseError(
          getErrorMessage(
            caught,
            "クローズに失敗しました。",
          ),
        );
      } finally {
        setClosingInquiry(false);
      }
    }, [
      inquiryId,
      closingInquiry,
    ]);

  const title =
    getInquiryTitle(inquiry);

  const shouldShowClosePrompt =
    inquiry?.status === "resolved";

  const replyActionDisabled =
    !inquiryId ||
    loading ||
    !inquiry ||
    postingReply ||
    inquiry.status === "closed";

  return {
    inquiryId,

    title,
    inquiry,
    replies,
    sortedReplies,

    loading,
    error,

    isReplyModalOpen,
    replyContent,
    replyFiles,
    replyError,
    postingReply,
    canSubmitReply,

    closingInquiry,
    closeError,

    shouldShowClosePrompt,
    replyActionDisabled,

    setReplyContent,

    loadThread,
    openReplyModal,
    closeReplyModal,
    handleReplyFilesChange,
    removeReplyFile,
    submitReply,
    handleCloseInquiry,
  };
}