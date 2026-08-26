// frontend/console/shell/src/features/inquiry/presentation/components/inquiryReplyListCard.tsx

import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

import type {
  InquiryDetail,
  InquiryImageFile,
} from "../../../../shared/types/inquiry";

type InquiryReplyView =
  InquiryDetail["replies"][number];

type InquiryImageView = {
  id: string;
  fileName: string;
  fileUrl: string;
};

export type InquiryReplyListCardProps = {
  replies: InquiryReplyView[];
  memberId: string;
  onOpenReplyModal: () => void;
};

function textOrDash(
  value: string | null | undefined,
): string {
  const normalized = String(value ?? "").trim();
  return normalized || "-";
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
    }),
  );
}

function replySenderLabel(
  reply: InquiryReplyView,
  memberId: string,
): string {
  if (reply.senderType === "member") {
    return reply.senderId === memberId
      ? "自分"
      : "担当者";
  }

  return "お客様";
}

export default function InquiryReplyListCard({
  replies,
  memberId,
  onOpenReplyModal,
}: InquiryReplyListCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-3">
        <CardTitle>
          返信一覧
        </CardTitle>

        <Button
          type="button"
          onClick={onOpenReplyModal}
        >
          返信
        </Button>
      </CardHeader>

      <CardContent>
        {replies.length > 0 ? (
          <div className="inq-reply-list">
            {replies.map((reply) => {
              const replyImages =
                normalizeImages(
                  reply.images,
                );

              const senderLabel =
                replySenderLabel(
                  reply,
                  memberId,
                );

              const createdAtLabel =
                safeDateTimeLabelJa(
                  reply.createdAt,
                  "-",
                );

              const isSelf =
                reply.senderType ===
                  "member" &&
                reply.senderId ===
                  memberId;

              return (
                <article
                  key={reply.id}
                  className={
                    isSelf
                      ? "inq-reply-item inq-reply-item--self"
                      : "inq-reply-item inq-reply-item--other"
                  }
                >
                  <div className="inq-reply-item__header">
                    <span className="inq-reply-item__sender">
                      {senderLabel}
                    </span>

                    <span className="inq-reply-item__date">
                      {createdAtLabel}
                    </span>
                  </div>

                  <p className="inq-reply-item__content">
                    {textOrDash(
                      reply.content,
                    )}
                  </p>

                  {replyImages.length > 0 ? (
                    <div className="inq-detail__image-grid">
                      {replyImages.map(
                        (image) => (
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
                  ) : null}
                </article>
              );
            })}
          </div>
        ) : (
          <div className="inq__empty">
            返信はありません。
          </div>
        )}
      </CardContent>
    </Card>
  );
}