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
  brandName: string;
  brandIcon: string;
  userName: string;
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
  brandName: string,
  userName: string,
): string {
  switch (reply.senderType) {
    case "member":
      return textOrDash(brandName);

    case "system":
      return "AMOL";

    case "avatar":
      return textOrDash(userName);

    default:
      return "-";
  }
}

export default function InquiryReplyListCard({
  replies,
  memberId,
  brandName,
  brandIcon,
  userName,
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
                  brandName,
                  userName,
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

              const showBrandIcon =
                reply.senderType ===
                  "member" &&
                Boolean(brandIcon);

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
                    <div className="inq-reply-item__sender-profile">
                      {showBrandIcon ? (
                        <img
                          src={brandIcon}
                          alt=""
                          className="inq-reply-item__sender-icon"
                        />
                      ) : null}

                      <span className="inq-reply-item__sender">
                        {senderLabel}
                      </span>
                    </div>

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