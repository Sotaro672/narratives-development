// frontend/amol/src/features/inquiry/presentation/components/InquiryReplyList.tsx

import { formatDateTime } from "../../../../components/utils/date";

import type { InquiryReply } from "../../../shared/types/inquiryTypes";

import InquiryImageGrid from "./InquiryImageGrid";

type InquiryReplyListProps = {
  replies: InquiryReply[];
  brandName: string;
  brandIcon: string;
  avatarName: string;
  avatarIcon: string;
};

type ReplySenderDisplay = {
  name: string;
  icon: string;
};

export default function InquiryReplyList({
  replies,
  brandName,
  brandIcon,
  avatarName,
  avatarIcon,
}: InquiryReplyListProps) {
  return (
    <>
      {replies.map((reply) => {
        const isAvatarReply = reply.senderType === "avatar";
        const isSystemReply = reply.senderType === "system";
        const sender = getReplySenderDisplay(
          reply.senderType,
          brandName,
          brandIcon,
          avatarName,
          avatarIcon,
        );
        const senderInitial = getInitial(sender.name);

        const className = isAvatarReply
          ? "chat-detail-page__reply chat-detail-page__reply--avatar"
          : isSystemReply
            ? "chat-detail-page__reply chat-detail-page__reply--system"
            : "chat-detail-page__reply";

        return (
          <article
            key={reply.id}
            className={className}
          >
            <div className="chat-detail-page__message-head">
              <div className="chat-detail-page__sender-profile">
                {!isSystemReply ? (
                  <div
                    className="chat-detail-page__sender-icon"
                    aria-hidden="true"
                  >
                    {sender.icon ? (
                      <img
                        src={sender.icon}
                        alt=""
                        className="chat-detail-page__sender-icon-image"
                      />
                    ) : (
                      <span>{senderInitial}</span>
                    )}
                  </div>
                ) : null}

                <div>
                  <span className="chat-detail-page__sender">
                    {sender.name}
                  </span>

                  <time
                    className="chat-detail-page__date"
                    dateTime={reply.createdAt}
                  >
                    {formatDateTime(reply.createdAt)}
                  </time>
                </div>
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

function getReplySenderDisplay(
  senderType: InquiryReply["senderType"],
  brandName: string,
  brandIcon: string,
  avatarName: string,
  avatarIcon: string,
): ReplySenderDisplay {
  switch (senderType) {
    case "avatar":
      return {
        name: avatarName,
        icon: avatarIcon,
      };

    case "member":
      return {
        name: brandName,
        icon: brandIcon,
      };

    case "system":
      return {
        name: "AMOL",
        icon: "",
      };
  }
}

function getInitial(value: string): string {
  return Array.from(value)[0] ?? "？";
}