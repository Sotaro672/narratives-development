// frontend/amol/src/features/inquiry/presentation/components/InquiryReplyList.tsx

import ChatMessageBubble from "../../../shared/presentation/components/ChatMessageBubble";
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

        return (
          <ChatMessageBubble
            key={reply.id}
            senderName={sender.name}
            senderIcon={sender.icon}
            createdAt={reply.createdAt}
            content={reply.content}
            isMine={isAvatarReply}
            isSystem={isSystemReply}
            afterContent={<InquiryImageGrid images={reply.images} />}
          />
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