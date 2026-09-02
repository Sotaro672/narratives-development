// frontend/amol/src/features/shared/presentation/components/ChatMessageBubble.tsx

import type { ReactNode } from "react";

import ChatMessageHeader from "./ChatMessageHeader";

export type ChatMessageBubbleProps = {
  senderName: string;
  senderIcon?: string | null;
  createdAt?: string | null;
  content?: string | null;
  isMine?: boolean;
  isSystem?: boolean;
  action?: ReactNode;
  className?: string;
};

function joinClassNames(
  ...classNames: Array<string | undefined | false>
): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ChatMessageBubble({
  senderName,
  senderIcon,
  createdAt,
  content,
  isMine = false,
  isSystem = false,
  action,
  className,
}: ChatMessageBubbleProps) {
  const bubbleClassName = joinClassNames(
    "chat-detail-page__reply",
    isMine && "chat-detail-page__reply--avatar",
    isSystem && "chat-detail-page__reply--system",
    className,
  );

  return (
    <article className={bubbleClassName}>
      <ChatMessageHeader
        name={senderName}
        icon={senderIcon}
        createdAt={createdAt}
        action={action}
        showAvatar={!isSystem}
      />

      {content ? (
        <p className="chat-detail-page__content">
          {content}
        </p>
      ) : null}
    </article>
  );
}