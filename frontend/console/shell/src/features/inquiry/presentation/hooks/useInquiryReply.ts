// frontend/console/shell/src/features/inquiry/presentation/hooks/useInquiryReply.ts

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import type {
  ChangeEvent,
} from "react";

import {
  replyInquiryHTTP,
  uploadInquiryReplyImagesToStorage,
} from "../../infrastructure/inquiryRepositoryHTTP";

import {
  MAX_REPLY_IMAGES,
  MAX_REPLY_IMAGE_SIZE_BYTES,
  MAX_REPLY_IMAGE_SIZE_MB,
} from "../../constants/inquiryReply";

export type ReplyUploadImage = {
  id: string;
  file: File;
  previewUrl: string;
};

export type UseInquiryReplyParams = {
  inquiryId: string;
  memberId: string;

  onReloadDetail: () => Promise<unknown>;
  onClearPageError: () => void;
};

export type UseInquiryReplyResult = {
  replyModalOpen: boolean;
  replyContent: string;
  replyImages: ReplyUploadImage[];
  replySubmitting: boolean;
  replyErrorMessage: string | null;

  onOpenReplyModal: () => void;
  onCloseReplyModal: () => void;
  onChangeReplyContent: (value: string) => void;
  onChangeReplyImages: (
    event: ChangeEvent<HTMLInputElement>,
  ) => void;
  onRemoveReplyImage: (id: string) => void;
  onSubmitReply: () => Promise<void>;
};

function createClientID(
  prefix: string,
): string {
  const randomID =
    globalThis.crypto?.randomUUID?.() ??
    `${Date.now()}-${Math.random()
      .toString(36)
      .slice(2)}`;

  return `${prefix}-${randomID}`;
}

function getErrorMessage(
  error: unknown,
  fallbackMessage: string,
): string {
  return error instanceof Error
    ? error.message
    : fallbackMessage;
}

export function useInquiryReply({
  inquiryId,
  memberId,
  onReloadDetail,
  onClearPageError,
}: UseInquiryReplyParams): UseInquiryReplyResult {
  const [
    replyModalOpen,
    setReplyModalOpen,
  ] = useState(false);

  const [
    replyContent,
    setReplyContent,
  ] = useState("");

  const [
    replyImages,
    setReplyImages,
  ] = useState<ReplyUploadImage[]>([]);

  const [
    replySubmitting,
    setReplySubmitting,
  ] = useState(false);

  const [
    replyErrorMessage,
    setReplyErrorMessage,
  ] = useState<string | null>(null);

  const replyImagePreviewUrlsRef =
    useRef<Set<string>>(
      new Set(),
    );

  useEffect(() => {
    return () => {
      for (
        const previewUrl of
        replyImagePreviewUrlsRef.current
      ) {
        URL.revokeObjectURL(
          previewUrl,
        );
      }

      replyImagePreviewUrlsRef.current.clear();
    };
  }, []);

  const revokeReplyImagePreviewUrl =
    useCallback(
      (
        previewUrl: string,
      ): void => {
        URL.revokeObjectURL(
          previewUrl,
        );

        replyImagePreviewUrlsRef.current.delete(
          previewUrl,
        );
      },
      [],
    );

  const clearReplyImages =
    useCallback((): void => {
      setReplyImages(
        (
          currentImages:
            ReplyUploadImage[],
        ) => {
          for (
            const image of currentImages
          ) {
            revokeReplyImagePreviewUrl(
              image.previewUrl,
            );
          }

          return [];
        },
      );
    }, [
      revokeReplyImagePreviewUrl,
    ]);

  const resetReplyForm =
    useCallback((): void => {
      setReplyContent("");
      setReplyErrorMessage(null);

      clearReplyImages();
    }, [
      clearReplyImages,
    ]);

  const onOpenReplyModal =
    useCallback((): void => {
      setReplyErrorMessage(null);
      setReplyModalOpen(true);
    }, []);

  const onCloseReplyModal =
    useCallback((): void => {
      if (replySubmitting) {
        return;
      }

      setReplyModalOpen(false);

      resetReplyForm();
    }, [
      replySubmitting,
      resetReplyForm,
    ]);

  const onChangeReplyContent =
    useCallback(
      (
        value: string,
      ): void => {
        setReplyContent(value);

        if (replyErrorMessage) {
          setReplyErrorMessage(null);
        }
      },
      [
        replyErrorMessage,
      ],
    );

  const onChangeReplyImages =
    useCallback(
      (
        event:
          ChangeEvent<HTMLInputElement>,
      ): void => {
        const files = Array.from(
          event.target.files ?? [],
        );

        event.target.value = "";

        if (
          files.length === 0
        ) {
          return;
        }

        setReplyErrorMessage(null);

        setReplyImages(
          (
            currentImages:
              ReplyUploadImage[],
          ) => {
            const remainingCount =
              MAX_REPLY_IMAGES -
              currentImages.length;

            if (
              remainingCount <= 0
            ) {
              setReplyErrorMessage(
                `添付画像は最大${MAX_REPLY_IMAGES}枚までです。`,
              );

              return currentImages;
            }

            const acceptedFiles: File[] = [];

            for (
              const file of files.slice(
                0,
                remainingCount,
              )
            ) {
              if (
                !file.type.startsWith(
                  "image/",
                )
              ) {
                setReplyErrorMessage(
                  "画像ファイルのみ添付できます。",
                );

                continue;
              }

              if (
                file.size >
                MAX_REPLY_IMAGE_SIZE_BYTES
              ) {
                setReplyErrorMessage(
                  `画像サイズは1枚あたり${MAX_REPLY_IMAGE_SIZE_MB}MB以下にしてください。`,
                );

                continue;
              }

              acceptedFiles.push(
                file,
              );
            }

            if (
              files.length >
              remainingCount
            ) {
              setReplyErrorMessage(
                `添付画像は最大${MAX_REPLY_IMAGES}枚までです。`,
              );
            }

            const nextImages =
              acceptedFiles.map(
                (
                  file: File,
                ): ReplyUploadImage => {
                  const previewUrl =
                    URL.createObjectURL(
                      file,
                    );

                  replyImagePreviewUrlsRef.current.add(
                    previewUrl,
                  );

                  return {
                    id: createClientID(
                      "reply-image",
                    ),
                    file,
                    previewUrl,
                  };
                },
              );

            return [
              ...currentImages,
              ...nextImages,
            ];
          },
        );
      },
      [],
    );

  const onRemoveReplyImage =
    useCallback(
      (
        id: string,
      ): void => {
        setReplyImages(
          (
            currentImages:
              ReplyUploadImage[],
          ) => {
            const target =
              currentImages.find(
                (
                  image:
                    ReplyUploadImage,
                ) =>
                  image.id === id,
              );

            if (target) {
              revokeReplyImagePreviewUrl(
                target.previewUrl,
              );
            }

            return currentImages.filter(
              (
                image:
                  ReplyUploadImage,
              ) =>
                image.id !== id,
            );
          },
        );

        setReplyErrorMessage(null);
      },
      [
        revokeReplyImagePreviewUrl,
      ],
    );

  const onSubmitReply =
    useCallback(
      async (): Promise<void> => {
        if (replySubmitting) {
          return;
        }

        const trimmedContent =
          replyContent.trim();

        if (!inquiryId) {
          setReplyErrorMessage(
            "問い合わせIDが指定されていません。",
          );

          return;
        }

        if (
          !trimmedContent &&
          replyImages.length === 0
        ) {
          setReplyErrorMessage(
            "返信内容または画像を入力してください。",
          );

          return;
        }

        if (
          replyImages.length > 0 &&
          !memberId
        ) {
          setReplyErrorMessage(
            "メンバーIDが取得できません。ログインし直してください。",
          );

          return;
        }

        setReplySubmitting(true);
        setReplyErrorMessage(null);

        onClearPageError();

        try {
          const uploadedImages =
            replyImages.length > 0
              ? await uploadInquiryReplyImagesToStorage(
                  {
                    inquiryId,
                    memberId,
                    files:
                      replyImages.map(
                        (
                          image:
                            ReplyUploadImage,
                        ) =>
                          image.file,
                      ),
                  },
                )
              : [];

          await replyInquiryHTTP(
            inquiryId,
            {
              content:
                trimmedContent,
              images:
                uploadedImages,
            },
          );

          await onReloadDetail();

          setReplyModalOpen(false);

          resetReplyForm();
        } catch (
          error: unknown
        ) {
          setReplyErrorMessage(
            getErrorMessage(
              error,
              "問い合わせ返信の送信に失敗しました",
            ),
          );
        } finally {
          setReplySubmitting(false);
        }
      },
      [
        inquiryId,
        memberId,
        onClearPageError,
        onReloadDetail,
        replyContent,
        replyImages,
        replySubmitting,
        resetReplyForm,
      ],
    );

  return {
    replyModalOpen,
    replyContent,
    replyImages,
    replySubmitting,
    replyErrorMessage,

    onOpenReplyModal,
    onCloseReplyModal,
    onChangeReplyContent,
    onChangeReplyImages,
    onRemoveReplyImage,
    onSubmitReply,
  };
}