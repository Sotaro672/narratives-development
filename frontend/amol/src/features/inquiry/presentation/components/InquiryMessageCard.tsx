// frontend/amol/src/features/inquiry/presentation/components/InquiryMessageCard.tsx

import { formatDateTime } from "../../../../components/utils/date";

import type { Inquiry } from "../../api/inquiryApi";

import InquiryImageGrid from "./InquiryImageGrid";

type InquiryMessageCardProps = {
  inquiry: Inquiry;
};

export default function InquiryMessageCard({
  inquiry,
}: InquiryMessageCardProps) {
  const statusLabel = getInquiryStatusLabel(inquiry.status);

  return (
    <article className="chat-detail-page__inquiry">
      <div className="chat-detail-page__message-head">
        <div>
          <span className="chat-detail-page__sender">
            あなたの問い合わせ
          </span>

          <time
            className="chat-detail-page__date"
            dateTime={inquiry.createdAt}
          >
            {formatDateTime(inquiry.createdAt)}
          </time>
        </div>

        <span className="chat-detail-page__status">
          {statusLabel}
        </span>
      </div>

      <h2 className="chat-detail-page__subject">
        {inquiry.subject}
      </h2>

      <p className="chat-detail-page__content">
        {inquiry.content}
      </p>

      <InquiryImageGrid images={inquiry.images} />
    </article>
  );
}

function getInquiryStatusLabel(
  status: Inquiry["status"],
): string {
  switch (status) {
    case "open":
      return "未対応";
    case "resolved":
      return "解決済み";
    case "closed":
      return "クローズ";
  }
}