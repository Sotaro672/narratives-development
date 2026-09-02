// frontend/amol/src/features/inquiry/presentation/components/InquiryMessageCard.tsx

import { formatDateTime } from "../../../../components/utils/date";

import ChatThreadCard from "../../../shared/presentation/components/ChatThreadCard";
import type { InquiryDetail } from "../../../shared/types/inquiryTypes";
import { getInquiryTypeLabel } from "../../../shared/types/inquiryTypes";

import InquiryImageGrid from "./InquiryImageGrid";
import InquiryModelMeta from "./InquiryModelMeta";

type InquiryMessageCardProps = {
  inquiry: InquiryDetail;
};

export default function InquiryMessageCard({
  inquiry,
}: InquiryMessageCardProps) {
  const statusLabel = getInquiryStatusLabel(inquiry.status);
  const title = getInquiryTitle(inquiry);
  const avatarInitial = getInitial(inquiry.avatarName);

  return (
    <ChatThreadCard variant="inquiry">
      <div className="chat-detail-page__message-head">
        <div className="chat-detail-page__sender-profile">
          <div
            className="chat-detail-page__sender-icon"
            aria-hidden="true"
          >
            {inquiry.avatarIcon ? (
              <img
                src={inquiry.avatarIcon}
                alt=""
                className="chat-detail-page__sender-icon-image"
              />
            ) : (
              <span>{avatarInitial}</span>
            )}
          </div>

          <div>
            <span className="chat-detail-page__sender">
              {inquiry.avatarName}
            </span>

            <time
              className="chat-detail-page__date"
              dateTime={inquiry.createdAt}
            >
              {formatDateTime(inquiry.createdAt)}
            </time>
          </div>
        </div>

        <span className="chat-detail-page__status">
          {statusLabel}
        </span>
      </div>

      <h2 className="chat-detail-page__subject">
        {title}
      </h2>

      {inquiry.inquiryType !== "product" && inquiry.modelMeta ? (
        <InquiryModelMeta modelMeta={inquiry.modelMeta} />
      ) : null}

      <p className="chat-detail-page__content">
        {inquiry.content}
      </p>

      <InquiryImageGrid images={inquiry.images} />
    </ChatThreadCard>
  );
}

function getInquiryTitle(
  inquiry: InquiryDetail,
): string {
  const inquiryLabel =
    inquiry.inquiryType === "product"
      ? inquiry.subject || getInquiryTypeLabel(inquiry.inquiryType)
      : getInquiryTypeLabel(inquiry.inquiryType);

  return `${inquiry.productName}/${inquiryLabel}`;
}

function getInquiryStatusLabel(
  status: InquiryDetail["status"],
): string {
  switch (status) {
    case "open":
      return "未対応";

    case "in_progress":
      return "対応中";

    case "resolved":
      return "解決済み";

    case "closed":
      return "クローズ";
  }
}

function getInitial(value: string): string {
  return Array.from(value)[0] ?? "？";
}