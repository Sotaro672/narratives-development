// frontend/amol/src/features/shared/presentation/components/ChatMessageHeader.tsx

import type { ReactNode } from "react";

import { formatDateTime } from "../../../../components/utils/date";

export type ChatMessageHeaderProps = {
  name: string;
  icon?: string | null;
  createdAt?: string | null;
  action?: ReactNode;
  showAvatar?: boolean;
  className?: string;
};

function joinClassNames(
  ...classNames: Array<string | undefined | false>
): string {
  return classNames.filter(Boolean).join(" ");
}

function getInitial(value: string): string {
  return Array.from(value.trim())[0] ?? "？";
}

export default function ChatMessageHeader({
  name,
  icon,
  createdAt,
  action,
  showAvatar = true,
  className,
}: ChatMessageHeaderProps) {
  const displayName = name.trim() || "ユーザー";
  const initial = getInitial(displayName);

  return (
    <div
      className={joinClassNames(
        "chat-detail-page__message-head",
        className,
      )}
    >
      <div className="chat-detail-page__sender-profile">
        {showAvatar ? (
          <div
            className="chat-detail-page__sender-icon"
            aria-hidden="true"
          >
            {icon ? (
              <img
                src={icon}
                alt=""
                className="chat-detail-page__sender-icon-image"
              />
            ) : (
              <span>{initial}</span>
            )}
          </div>
        ) : null}

        <div>
          <span className="chat-detail-page__sender">
            {displayName}
          </span>

          {createdAt ? (
            <time
              className="chat-detail-page__date"
              dateTime={createdAt}
            >
              {formatDateTime(createdAt)}
            </time>
          ) : null}
        </div>
      </div>

      {action ? action : null}
    </div>
  );
}
