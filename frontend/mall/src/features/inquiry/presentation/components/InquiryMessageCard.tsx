// frontend/amol/src/features/inquiry/presentation/components/InquiryMessageCard.tsx

import ChatImageGrid from "../../../shared/presentation/components/ChatImageGrid";
import ChatMessageHeader from "../../../shared/presentation/components/ChatMessageHeader";
import ChatThreadCard from "../../../shared/presentation/components/ChatThreadCard";
import type { InquiryDetail } from "../../../shared/types/inquiryTypes";
import { getInquiryTypeLabel } from "../../../shared/types/inquiryTypes";

import InquiryModelMeta from "./InquiryModelMeta";

type InquiryMessageCardProps = {
  inquiry: InquiryDetail;
};

export default function InquiryMessageCard({
  inquiry,
}: InquiryMessageCardProps) {
  const statusLabel = getInquiryStatusLabel(inquiry.status);
  const title = getInquiryTitle(inquiry);

  const images = inquiry.images?.map((image) => ({
    key: image.fileUrl,
    url: image.fileUrl,
    alt: image.fileName,
  }));

  return (
    <ChatThreadCard variant="inquiry">
      <ChatMessageHeader
        name={inquiry.avatarName}
        icon={inquiry.avatarIcon}
        createdAt={inquiry.createdAt}
        action={
          <span className="chat-detail-page__status">
            {statusLabel}
          </span>
        }
      />

      <h2 className="chat-detail-page__subject">
        {title}
      </h2>

      {inquiry.inquiryType !== "product" && inquiry.modelMeta ? (
        <InquiryModelMeta modelMeta={inquiry.modelMeta} />
      ) : null}

      <p className="chat-detail-page__content">
        {inquiry.content}
      </p>

      <ChatImageGrid images={images} />
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