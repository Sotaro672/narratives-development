// frontend/amol/src/features/shared/presentation/components/ChatMessageHeader.tsx

import type { ReactNode } from "react";

import DateDisplay from "../../../../components/ui/Date";
import MediaIcon from "../../../../components/ui/MediaIcon";

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
          <span aria-hidden="true">
            <MediaIcon
              src={icon}
              alt=""
              fallback={initial}
              size="sm"
              shape="circle"
            />
          </span>
        ) : null}

        <div>
          <span className="chat-detail-page__sender">
            {displayName}
          </span>

          {createdAt ? (
            <DateDisplay
              value={createdAt}
              variant="dateTime"
              className="chat-detail-page__date"
            />
          ) : null}
        </div>
      </div>

      {action ?? null}
    </div>
  );
}