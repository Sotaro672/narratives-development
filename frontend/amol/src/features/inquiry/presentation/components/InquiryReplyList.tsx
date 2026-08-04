// frontend/amol/src/features/inquiry/presentation/components/InquiryReplyList.tsx

import { formatDateTime } from "../../../../components/utils/date";

import type {
  InquiryReply,
} from "../../api/inquiryApi";

import InquiryImageGrid from "./InquiryImageGrid";

type InquiryReplyListProps = {
  replies: InquiryReply[];
};

export default function InquiryReplyList({
  replies,
}: InquiryReplyListProps) {
  return (
    <>
      {replies.map((reply, index) => {
        const isAvatarReply =
          reply.senderType === "avatar";

        return (
          <article
            key={
              reply.id ||
              `${reply.inquiryId ?? "inquiry"}-${index}`
            }
            className={
              isAvatarReply
                ? "chat-detail-page__reply chat-detail-page__reply--avatar"
                : "chat-detail-page__reply"
            }
          >
            <div className="chat-detail-page__message-head">
              <div>
                <span className="chat-detail-page__sender">
                  {isAvatarReply
                    ? "あなた"
                    : "テナント"}
                </span>

                {reply.createdAt ? (
                  <time
                    className="chat-detail-page__date"
                    dateTime={reply.createdAt}
                  >
                    {formatDateTime(
                      reply.createdAt,
                    )}
                  </time>
                ) : null}
              </div>
            </div>

            {reply.content ? (
              <p className="chat-detail-page__content">
                {reply.content}
              </p>
            ) : null}

            <InquiryImageGrid
              images={reply.images}
            />
          </article>
        );
      })}
    </>
  );
}