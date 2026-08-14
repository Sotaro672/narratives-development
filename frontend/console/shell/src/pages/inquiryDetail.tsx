// frontend/console/shell/src/pages/inquiryDetail.tsx

import * as React from "react";

import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import { safeDateTimeLabelJa } from "../../../shell/src/shared/util/dateJa";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../shell/src/shared/ui/card";

import {
  replyInquiryHTTP,
  uploadInquiryReplyImagesToStorage,
} from "../features/inquiry/infrastructure/inquiryRepositoryHTTP";

import ReplyModal, {
  type ReplyUploadImage,
} from "../features/inquiry/presentation/components/replyModal";

import { useInquiryDetailPage } from "../features/inquiry/presentation/hooks/useInquiryDetailPage";

import {
  MAX_REPLY_IMAGES,
  MAX_REPLY_IMAGE_SIZE_BYTES,
  MAX_REPLY_IMAGE_SIZE_MB,
} from "../features/inquiry/constants/inquiryReply";

import {
  getInquiryStatusButtonVariant,
  getInquiryStatusLabel,
  isClosedStatus,
} from "../features/inquiry/presentation/utils/inquiryStatus";

import type {
  InquiryDetail as InquiryDetailDTO,
  InquiryImageFile,
  InquiryOrderItemSummary,
  InquiryOrderSummary,
} from "../shared/types/inquiry";

import type {
  ShippingAddress,
} from "../shared/types/shippingAddress";

import "../styles/inquiry-page.css";

type InquiryImageView = {
  id: string;
  fileName: string;
  fileUrl: string;
  mimeType: string;
};

type InquiryReplyView =
  InquiryDetailDTO["replies"][number];

function textOrDash(
  value: string | null | undefined,
): string {
  const normalized = String(value ?? "").trim();

  return normalized || "-";
}

function normalizeText(value: unknown): string {
  return String(value ?? "").trim();
}

function typeLabel(
  value: string | null | undefined,
): string {
  const inquiryType = normalizeText(value);

  switch (inquiryType) {
    case "product_description":
      return "商品説明";

    case "exchange":
      return "交換";

    case "shipping":
      return "配送";

    case "payment":
      return "決済";

    case "other":
      return "その他";

    case "product":
      return "商品";

    default:
      return inquiryType || "-";
  }
}

function uniqueTextValues(
  values: Array<string | null | undefined>,
): string[] {
  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const normalized = normalizeText(value);

    if (!normalized || normalized === "-") {
      continue;
    }

    if (seen.has(normalized)) {
      continue;
    }

    seen.add(normalized);
    result.push(normalized);
  }

  return result;
}

function createClientID(prefix: string): string {
  const randomID =
    globalThis.crypto?.randomUUID?.() ??
    `${Date.now()}-${Math.random().toString(36).slice(2)}`;

  return `${prefix}-${randomID}`;
}

function getShippingAddressLine(
  address: ShippingAddress,
): string {
  const zipCode = normalizeText(
    address.zipCode,
  );

  const state = normalizeText(
    address.state,
  );

  const city = normalizeText(
    address.city,
  );

  const street = normalizeText(
    address.street,
  );

  const parts = [
    zipCode ? `〒${zipCode}` : "",
    state,
    city,
    street,
  ].filter(Boolean);

  return parts.length > 0
    ? parts.join(" ")
    : "-";
}

function getShippingAddressStreet2(
  address: ShippingAddress,
): string {
  return normalizeText(
    address.street2,
  );
}

function getOrderItemsLabel(
  order: InquiryOrderSummary,
): string {
  if (order.items.length === 0) {
    return "-";
  }

  return order.items
    .map((item: InquiryOrderItemSummary) => {
      const tokenName = textOrDash(item.tokenName);
      const quantity = item.qty;

      return quantity > 0
        ? `${tokenName} × ${quantity}`
        : tokenName;
    })
    .join(" / ");
}

function getOrderTransferredAtLabel(
  order: InquiryOrderSummary,
): string {
  if (order.items.length === 0) {
    return "-";
  }

  const transferredAtValues = uniqueTextValues(
    order.items.map(
      (item: InquiryOrderItemSummary) =>
        item.transferredAt ?? null,
    ),
  );

  if (transferredAtValues.length === 0) {
    return "-";
  }

  return transferredAtValues
    .map((transferredAt) =>
      safeDateTimeLabelJa(transferredAt, "-"),
    )
    .join(" / ");
}

function normalizeImages(
  images: InquiryImageFile[] | undefined,
): InquiryImageView[] {
  if (!images) {
    return [];
  }

  return images.map(
    (
      image: InquiryImageFile,
      index: number,
    ): InquiryImageView => ({
      id:
        image.objectPath ||
        `${image.fileUrl}-${index}`,
      fileName: image.fileName,
      fileUrl: image.fileUrl,
      mimeType: image.mimeType,
    }),
  );
}

function getInquiryImages(
  inquiry:
    | InquiryDetailDTO["inquiry"]
    | null
    | undefined,
): InquiryImageView[] {
  return normalizeImages(
    inquiry?.images,
  );
}

function getReplyImages(
  reply:
    | InquiryReplyView
    | null
    | undefined,
): InquiryImageView[] {
  return normalizeImages(
    reply?.images,
  );
}

function replySenderLabel(
  reply: InquiryReplyView,
  params: {
    memberId: string;
    avatarName: string;
  },
): string {
  if (reply.senderType === "member") {
    return reply.senderId === params.memberId
      ? "自分"
      : "担当者";
  }

  return params.avatarName !== "-"
    ? params.avatarName
    : "アバター";
}

export default function InquiryDetail() {
  const {
    inquiryId,
    memberId,
    detail,
    loading,
    statusUpdating,
    errorMessage,
    onBack,
    reloadDetail,
    clearErrorMessage,
    onToggleStatus,
  } = useInquiryDetailPage();

  const [
    replyModalOpen,
    setReplyModalOpen,
  ] = React.useState(false);

  const [
    replyContent,
    setReplyContent,
  ] = React.useState("");

  const [
    replyImages,
    setReplyImages,
  ] = React.useState<ReplyUploadImage[]>([]);

  const [
    replySubmitting,
    setReplySubmitting,
  ] = React.useState(false);

  const [
    replyErrorMessage,
    setReplyErrorMessage,
  ] = React.useState<string | null>(null);

  const replyImagePreviewUrlsRef =
    React.useRef<Set<string>>(
      new Set(),
    );

  React.useEffect(() => {
    return () => {
      for (
        const previewUrl of
        replyImagePreviewUrlsRef.current
      ) {
        URL.revokeObjectURL(previewUrl);
      }

      replyImagePreviewUrlsRef.current.clear();
    };
  }, []);

  const inquiry =
    detail?.inquiry ?? null;

  const title =
    textOrDash(inquiry?.subject);

  const body =
    textOrDash(inquiry?.content);

  const avatarName =
    textOrDash(detail?.avatarName);

  const userFullName =
    textOrDash(detail?.userFullName);

  const status =
    getInquiryStatusLabel(inquiry?.status);

  const inquiryType =
    typeLabel(inquiry?.inquiryType);

  const productName =
    textOrDash(detail?.productName);

  const brandName =
    textOrDash(detail?.brandName);

  const inquiredAt =
    safeDateTimeLabelJa(
      inquiry?.createdAt,
      "-",
    );

  const updatedAt =
    safeDateTimeLabelJa(
      inquiry?.updatedAt,
      "-",
    );

  const inquiryImages =
    getInquiryImages(inquiry);

  const replies: InquiryReplyView[] =
    detail?.replies ?? [];

  const shippingAddresses =
    detail?.shippingAddresses ?? [];

  const orders: InquiryOrderSummary[] =
    detail?.orders ?? [];

  const pageTitle = (
    <div className="inq-detail__page-title">
      <span className="inq__chip">
        {inquiryType}
      </span>

      <span className="inq-detail__page-title-text">
        {title}
      </span>
    </div>
  );

  const statusButtonVariant =
    getInquiryStatusButtonVariant(
      inquiry?.status,
    );

  const statusButtonDisabled =
    !detail ||
    isClosedStatus(inquiry?.status);

  const revokeReplyImagePreviewUrl =
    React.useCallback(
      (previewUrl: string) => {
        URL.revokeObjectURL(previewUrl);

        replyImagePreviewUrlsRef.current.delete(
          previewUrl,
        );
      },
      [],
    );

  const clearReplyImages =
    React.useCallback(() => {
      setReplyImages(
        (
          currentImages: ReplyUploadImage[],
        ) => {
          for (const image of currentImages) {
            revokeReplyImagePreviewUrl(
              image.previewUrl,
            );
          }

          return [];
        },
      );
    }, [revokeReplyImagePreviewUrl]);

  const onChangeReplyImages =
    React.useCallback(
      (
        event:
          React.ChangeEvent<HTMLInputElement>,
      ) => {
        const files = Array.from(
          event.target.files ?? [],
        );

        event.target.value = "";

        if (files.length === 0) {
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

            if (remainingCount <= 0) {
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

              acceptedFiles.push(file);
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
                    URL.createObjectURL(file);

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
    React.useCallback(
      (id: string) => {
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
      },
      [revokeReplyImagePreviewUrl],
    );

  const onOpenReplyModal =
    React.useCallback(() => {
      setReplyErrorMessage(null);
      setReplyModalOpen(true);
    }, []);

  const onCloseReplyModal =
    React.useCallback(() => {
      if (replySubmitting) {
        return;
      }

      setReplyModalOpen(false);
      setReplyContent("");
      setReplyErrorMessage(null);
      clearReplyImages();
    }, [
      clearReplyImages,
      replySubmitting,
    ]);

  const onSubmitReply =
    React.useCallback(
      async () => {
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
        clearErrorMessage();

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
              content: trimmedContent,
              images: uploadedImages,
            },
          );

          await reloadDetail();

          setReplyModalOpen(false);
          setReplyContent("");
          setReplyErrorMessage(null);
          clearReplyImages();
        } catch (error) {
          const message =
            error instanceof Error
              ? error.message
              : "問い合わせ返信の送信に失敗しました";

          setReplyErrorMessage(message);
        } finally {
          setReplySubmitting(false);
        }
      },
      [
        clearErrorMessage,
        clearReplyImages,
        inquiryId,
        memberId,
        reloadDetail,
        replyContent,
        replyImages,
      ],
    );

  if (loading) {
    return (
      <>
        <PageStyle
          layout="grid-2"
          title="問い合わせ詳細"
          onBack={onBack}
          onSave={undefined}
        >
          <Card>
            <CardHeader>
              <CardTitle>
                問い合わせ内容
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq__empty">
                問い合わせ詳細を読み込み中です。
              </div>
            </CardContent>
          </Card>

          <div>
            <Card>
              <CardHeader>
                <CardTitle>
                  問い合わせ情報
                </CardTitle>
              </CardHeader>

              <CardContent>
                <div className="inq__empty">
                  問い合わせ情報を読み込み中です。
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>
                  商品・注文情報
                </CardTitle>
              </CardHeader>

              <CardContent>
                <div className="inq__empty">
                  商品・注文情報を読み込み中です。
                </div>
              </CardContent>
            </Card>
          </div>
        </PageStyle>
      </>
    );
  }

  if (
    errorMessage &&
    !detail
  ) {
    return (
      <>
        <PageStyle
          layout="grid-2"
          title="問い合わせ詳細"
          onBack={onBack}
          onSave={undefined}
        >
          <Card>
            <CardHeader>
              <CardTitle>
                問い合わせ内容
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq__empty">
                {errorMessage}
              </div>
            </CardContent>
          </Card>

          <div>
            <Card>
              <CardHeader>
                <CardTitle>
                  問い合わせ情報
                </CardTitle>
              </CardHeader>

              <CardContent>
                <div className="inq__empty">
                  問い合わせ情報を表示できません。
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>
                  商品・注文情報
                </CardTitle>
              </CardHeader>

              <CardContent>
                <div className="inq__empty">
                  商品・注文情報を表示できません。
                </div>
              </CardContent>
            </Card>
          </div>
        </PageStyle>
      </>
    );
  }

  return (
    <>
      <PageStyle
        layout="grid-2"
        title={pageTitle}
        onBack={onBack}
        onSave={undefined}
        onReply={onOpenReplyModal}
        statusButtonLabel={status}
        statusButtonBusyLabel="更新中"
        statusButtonVariant={
          statusButtonVariant
        }
        onStatusButtonClick={
          onToggleStatus
        }
        isStatusButtonLoading={
          statusUpdating
        }
        statusButtonDisabled={
          statusButtonDisabled
        }
      >
        <div>
          <Card>
            <CardHeader>
              <CardTitle>
                問い合わせ内容
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq-detail">
                {errorMessage ? (
                  <div className="inq__empty">
                    {errorMessage}
                  </div>
                ) : null}

                <div className="inq-detail__body">
                  <div className="inq-detail__label">
                    問い合わせ本文
                  </div>

                  <p className="inq-detail__text">
                    {body}
                  </p>
                </div>

                <div className="inq-detail__body">
                  <div className="inq-detail__label">
                    添付画像
                  </div>

                  {inquiryImages.length >
                  0 ? (
                    <div className="inq-detail__image-grid">
                      {inquiryImages.map(
                        (
                          image:
                            InquiryImageView,
                        ) => (
                          <a
                            key={image.id}
                            href={
                              image.fileUrl
                            }
                            target="_blank"
                            rel="noreferrer"
                            className="inq-detail__image-link"
                            aria-label={`${image.fileName}を開く`}
                          >
                            <img
                              src={
                                image.fileUrl
                              }
                              alt={
                                image.fileName
                              }
                              className="inq-detail__image"
                              loading="lazy"
                            />
                          </a>
                        ),
                      )}
                    </div>
                  ) : (
                    <div className="inq__empty">
                      添付画像はありません。
                    </div>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>
                返信一覧
              </CardTitle>
            </CardHeader>

            <CardContent>
              {replies.length > 0 ? (
                <div className="inq-reply-list">
                  {replies.map(
                    (
                      reply:
                        InquiryReplyView,
                    ) => {
                      const replyImagesView =
                        getReplyImages(
                          reply,
                        );

                      const senderLabel =
                        replySenderLabel(
                          reply,
                          {
                            memberId,
                            avatarName,
                          },
                        );

                      const createdAtLabel =
                        safeDateTimeLabelJa(
                          reply.createdAt,
                          "-",
                        );

                      return (
                        <article
                          key={reply.id}
                          className="inq-reply-item"
                        >
                          <div className="inq-reply-item__header">
                            <span className="inq-reply-item__sender">
                              {senderLabel}
                            </span>

                            <span className="inq-reply-item__date">
                              {
                                createdAtLabel
                              }
                            </span>
                          </div>

                          <p className="inq-reply-item__content">
                            {textOrDash(
                              reply.content,
                            )}
                          </p>

                          {replyImagesView.length >
                          0 ? (
                            <div className="inq-detail__image-grid">
                              {replyImagesView.map(
                                (
                                  image:
                                    InquiryImageView,
                                ) => (
                                  <a
                                    key={
                                      image.id
                                    }
                                    href={
                                      image.fileUrl
                                    }
                                    target="_blank"
                                    rel="noreferrer"
                                    className="inq-detail__image-link"
                                    aria-label={`${image.fileName}を開く`}
                                  >
                                    <img
                                      src={
                                        image.fileUrl
                                      }
                                      alt={
                                        image.fileName
                                      }
                                      className="inq-detail__image"
                                      loading="lazy"
                                    />
                                  </a>
                                ),
                              )}
                            </div>
                          ) : null}
                        </article>
                      );
                    },
                  )}
                </div>
              ) : (
                <div className="inq__empty">
                  返信はありません。
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <div>
          <Card>
            <CardHeader>
              <CardTitle>
                問い合わせ情報
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq-detail">
                <div className="inq-detail__meta">
                  <div>
                    <span className="inq-detail__label">
                      アバター名
                    </span>

                    <span className="inq-detail__value">
                      {avatarName}
                    </span>
                  </div>

                  <div>
                    <span className="inq-detail__label">
                      ユーザー名
                    </span>

                    <span className="inq-detail__value">
                      {userFullName}
                    </span>
                  </div>

                  <div>
                    <span className="inq-detail__label">
                      配送先情報
                    </span>

                    {shippingAddresses.length >
                    0 ? (
                      <div className="inq-detail__value">
                        {shippingAddresses.map(
                          (
                            address:
                              ShippingAddress,
                          ) => {
                            const addressLine =
                              getShippingAddressLine(
                                address,
                              );

                            const street2 =
                              getShippingAddressStreet2(
                                address,
                              );

                            return (
                              <span
                                key={address.id}
                              >
                                {
                                  addressLine
                                }

                                {street2
                                  ? ` ${street2}`
                                  : ""}
                              </span>
                            );
                          },
                        )}
                      </div>
                    ) : (
                      <span className="inq-detail__value">
                        -
                      </span>
                    )}
                  </div>

                  <div>
                    <span className="inq-detail__label">
                      問い合わせ日
                    </span>

                    <span className="inq-detail__value">
                      {inquiredAt}
                    </span>
                  </div>

                  <div>
                    <span className="inq-detail__label">
                      最終更新日
                    </span>

                    <span className="inq-detail__value">
                      {updatedAt}
                    </span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>
                商品・注文情報
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq-detail">
                <div className="inq-detail__meta">
                  <div>
                    <span className="inq-detail__label">
                      商品名
                    </span>

                    <span className="inq-detail__value">
                      {productName}
                    </span>
                  </div>

                  <div>
                    <span className="inq-detail__label">
                      ブランド
                    </span>

                    <span className="inq-detail__value">
                      {brandName}
                    </span>
                  </div>

                  {orders.length > 0 ? (
                    orders.flatMap(
                      (
                        order:
                          InquiryOrderSummary,
                        index: number,
                      ) => [
                        <div
                          key={`${order.id}-id-${index}`}
                        >
                          <span className="inq-detail__label">
                            注文ID
                          </span>

                          <span className="inq-detail__value">
                            {textOrDash(
                              order.id,
                            )}
                          </span>
                        </div>,

                        <div
                          key={`${order.id}-created-at-${index}`}
                        >
                          <span className="inq-detail__label">
                            発注日時
                          </span>

                          <span className="inq-detail__value">
                            {safeDateTimeLabelJa(
                              order.createdAt,
                              "-",
                            )}
                          </span>
                        </div>,

                        <div
                          key={`${order.id}-transferred-at-${index}`}
                        >
                          <span className="inq-detail__label">
                            移譲日
                          </span>

                          <span className="inq-detail__value">
                            {getOrderTransferredAtLabel(
                              order,
                            )}
                          </span>
                        </div>,

                        <div
                          key={`${order.id}-items-${index}`}
                        >
                          <span className="inq-detail__label">
                            注文内容
                          </span>

                          <span className="inq-detail__value">
                            {getOrderItemsLabel(
                              order,
                            )}
                          </span>
                        </div>,
                      ],
                    )
                  ) : (
                    <div className="inq__empty">
                      注文情報はありません。
                    </div>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </PageStyle>

      <ReplyModal
        open={replyModalOpen}
        content={replyContent}
        images={replyImages}
        submitting={replySubmitting}
        errorMessage={replyErrorMessage}
        onClose={onCloseReplyModal}
        onChangeContent={setReplyContent}
        onChangeImages={
          onChangeReplyImages
        }
        onRemoveImage={
          onRemoveReplyImage
        }
        onSubmit={() =>
          void onSubmitReply()
        }
      />
    </>
  );
}