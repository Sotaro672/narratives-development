// frontend/amol/src/features/inquiry/presentation/components/InquiryListItem.tsx

import { textOrEmpty } from "../../../../components/utils/textOrEmpty";

import type {
  InquiryChatListItem,
} from "../hooks/useInquiryListPage";

type InquiryListItemProps = {
  item: InquiryChatListItem;
  navigating: boolean;
  onOpen: () => void;
};

export default function InquiryListItem({
  item,
  navigating,
  onOpen,
}: InquiryListItemProps) {
  const isUnread =
    item.isRead === false;

  const title =
    getInquiryTitle(item);

  const preview =
    getInquiryPreview(item);

  const latestActivityAt =
    getLatestActivityAt(item);

  const dateLabel =
    formatInquiryDate(
      latestActivityAt,
    );

  const subLabel =
    getInquirySubLabel(item);

  const statusLabel =
    getInquiryStatusLabel(
      item.status,
    );

  const countLabel =
    getReplyCountLabel(item);

  const avatarIcon =
    getInquiryAvatarIcon(item);

  const avatarInitial =
    getInitial(title);

  const handleOpen = () => {
    if (navigating) {
      return;
    }

    onOpen();
  };

  return (
    <article
      className={
        isUnread
          ? "chat-list-page__row chat-list-page__row--unread"
          : "chat-list-page__row"
      }
      role="button"
      tabIndex={0}
      aria-label={`${title} のチャットを開く`}
      aria-busy={navigating}
      onClick={handleOpen}
      onKeyDown={(event) => {
        if (
          navigating ||
          (
            event.key !== "Enter" &&
            event.key !== " "
          )
        ) {
          return;
        }

        event.preventDefault();
        onOpen();
      }}
    >
      <div
        className="chat-list-page__avatar"
        aria-hidden="true"
      >
        {avatarIcon ? (
          <img
            src={avatarIcon}
            alt=""
            loading="lazy"
            referrerPolicy="no-referrer"
            style={{
              width: "100%",
              height: "100%",
              display: "block",
              objectFit: "cover",
              borderRadius: "inherit",
            }}
            onError={(event) => {
              event.currentTarget.style.display =
                "none";

              const fallback =
                event.currentTarget
                  .nextElementSibling;

              if (
                fallback instanceof
                HTMLElement
              ) {
                fallback.style.display =
                  "inline";
              }
            }}
          />
        ) : null}

        <span
          style={
            avatarIcon
              ? {
                  display: "none",
                }
              : undefined
          }
        >
          {avatarInitial}
        </span>
      </div>

      <div className="chat-list-page__body">
        <div className="chat-list-page__head">
          <div className="chat-list-page__title-wrap">
            <h2 className="chat-list-page__title">
              {title}
            </h2>

            {subLabel ? (
              <span className="chat-list-page__sub-label">
                {subLabel}
              </span>
            ) : null}
          </div>

          {dateLabel ? (
            <time
              className="chat-list-page__date"
              dateTime={
                latestActivityAt ??
                undefined
              }
            >
              {dateLabel}
            </time>
          ) : null}
        </div>

        <div className="chat-list-page__content">
          <p className="chat-list-page__preview">
            {preview}
          </p>

          <div className="chat-list-page__meta">
            {countLabel ? (
              <span className="chat-list-page__reply-count">
                {countLabel}
              </span>
            ) : null}

            {statusLabel ? (
              <span className="chat-list-page__status">
                {statusLabel}
              </span>
            ) : null}

            {isUnread ? (
              <span
                className="chat-list-page__unread-dot"
                aria-label="未読"
              />
            ) : null}
          </div>
        </div>
      </div>
    </article>
  );
}

function getInquiryTitle(
  item: InquiryChatListItem,
): string {
  const subject =
    textOrEmpty(item.subject);

  if (subject) {
    return subject;
  }

  const productName =
    textOrEmpty(
      item.productName,
    );

  if (productName) {
    return productName;
  }

  const tokenName =
    textOrEmpty(
      item.tokenName,
    );

  if (tokenName) {
    return tokenName;
  }

  return "問い合わせ";
}

function getInquiryPreview(
  item: InquiryChatListItem,
): string {
  const latestReply =
    getLatestReply(
      item.replies,
    );

  const latestReplyContent =
    textOrEmpty(
      latestReply?.content,
    );

  if (latestReplyContent) {
    return latestReplyContent;
  }

  const content =
    textOrEmpty(item.content);

  if (content) {
    return content;
  }

  if (
    Array.isArray(item.images) &&
    item.images.length > 0
  ) {
    return `画像 ${item.images.length} 件`;
  }

  return "メッセージはありません";
}

function getInquirySubLabel(
  item: InquiryChatListItem,
): string {
  return (
    textOrEmpty(
      item.brandName,
    ) ||
    textOrEmpty(
      item.avatarName,
    ) ||
    textOrEmpty(
      item.senderName,
    ) ||
    textOrEmpty(
      item.productId,
    ) ||
    ""
  );
}

function getInquiryStatusLabel(
  status?: string | null,
): string {
  switch (status) {
    case "open":
      return "未対応";

    case "resolved":
      return "解決済み";

    case "closed":
      return "クローズ";

    default:
      return "";
  }
}

function getReplyCountLabel(
  item: InquiryChatListItem,
): string {
  return item.replies.length > 0
    ? `返信 ${item.replies.length} 件`
    : "";
}

function getInquiryAvatarIcon(
  _item: InquiryChatListItem,
): string {
  return "";
}

function getInitial(
  value: string,
): string {
  const normalizedValue =
    textOrEmpty(value);

  if (!normalizedValue) {
    return "？";
  }

  return (
    Array.from(
      normalizedValue,
    )[0] ?? "？"
  );
}

function getLatestActivityAt(
  item: InquiryChatListItem,
): string | null | undefined {
  const latestReply =
    getLatestReply(
      item.replies,
    );

  return (
    latestReply?.updatedAt ||
    latestReply?.createdAt ||
    item.updatedAt ||
    item.createdAt
  );
}

function getLatestReply(
  replies: InquiryChatListItem["replies"],
): InquiryChatListItem["replies"][number] | null {
  if (
    !Array.isArray(replies) ||
    replies.length === 0
  ) {
    return null;
  }

  return (
    [...replies].sort(
      (
        firstReply,
        secondReply,
      ) => {
        const firstTime =
          getComparableTime(
            firstReply.updatedAt ??
              firstReply.createdAt,
          );

        const secondTime =
          getComparableTime(
            secondReply.updatedAt ??
              secondReply.createdAt,
          );

        return (
          secondTime -
          firstTime
        );
      },
    )[0] ?? null
  );
}

function formatInquiryDate(
  value?: string | null,
): string {
  if (!value) {
    return "";
  }

  const date =
    new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "";
  }

  const now =
    new Date();

  const isToday =
    date.getFullYear() ===
      now.getFullYear() &&
    date.getMonth() ===
      now.getMonth() &&
    date.getDate() ===
      now.getDate();

  if (isToday) {
    return new Intl.DateTimeFormat(
      "ja-JP",
      {
        hour: "2-digit",
        minute: "2-digit",
      },
    ).format(date);
  }

  const isCurrentYear =
    date.getFullYear() ===
    now.getFullYear();

  if (isCurrentYear) {
    return new Intl.DateTimeFormat(
      "ja-JP",
      {
        month: "2-digit",
        day: "2-digit",
      },
    ).format(date);
  }

  return new Intl.DateTimeFormat(
    "ja-JP",
    {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    },
  ).format(date);
}

function getComparableTime(
  value?: string | null,
): number {
  if (!value) {
    return 0;
  }

  const date =
    new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return 0;
  }

  return date.getTime();
}