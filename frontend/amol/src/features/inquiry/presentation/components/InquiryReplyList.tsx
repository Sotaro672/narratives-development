// frontend/amol/src/features/inquiry/presentation/components/InquiryReplyList.tsx

import { formatDateTime } from "../../../../components/utils/date";

import type { InquiryReply } from "../../api/inquiryApi";

import InquiryImageGrid from "./InquiryImageGrid";

type InquiryReplyListProps = {
  replies: InquiryReply[];
};

function getReplySenderLabel(
  senderType: InquiryReply["senderType"],
): string {
  switch (senderType) {
    case "avatar":
      return "あなた";

    case "member":
      return "テナント";

    case "system":
      return "AMOL";

    default:
      return "-";
  }
}

export default function InquiryReplyList({
  replies,
}: InquiryReplyListProps) {
  return (
    <>
      {replies.map((reply) => {
        const isAvatarReply =
          reply.senderType === "avatar";

        const senderLabel =
          getReplySenderLabel(
            reply.senderType,
          );

        return (
          <article
            key={reply.id}
            className={
              isAvatarReply
                ? "chat-detail-page__reply chat-detail-page__reply--avatar"
                : "chat-detail-page__reply"
            }
          >
            <div className="chat-detail-page__message-head">
              <div>
                <span className="chat-detail-page__sender">
                  {senderLabel}
                </span>

                <time
                  className="chat-detail-page__date"
                  dateTime={reply.createdAt}
                >
                  {formatDateTime(reply.createdAt)}
                </time>
              </div>
            </div>

            {reply.content ? (
              <p className="chat-detail-page__content">
                {reply.content}
              </p>
            ) : null}

            <InquiryImageGrid images={reply.images} />
          </article>
        );
      })}
    </>
  );
}