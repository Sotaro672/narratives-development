// frontend/mall/src/features/inquiry/presentation/components/InquiryMessageCard.tsx

import Badge from "../../../../components/ui/Badge";
import SectionHeader from "../../../../components/ui/SectionHeader";

import ChatImageGrid from "../../../shared/presentation/components/ChatImageGrid";
import type { InquiryDetail } from "../../../shared/types/inquiryTypes";
import { getInquiryTypeLabel } from "../../../shared/types/inquiryTypes";

import InquiryModelMeta from "./InquiryModelMeta";

type InquiryMessageCardProps = {
  inquiry: InquiryDetail;
};

type InquiryStatusBadgeVariant =
  | "neutral"
  | "info"
  | "success"
  | "warning";

export default function InquiryMessageCard({
  inquiry,
}: InquiryMessageCardProps) {
  const statusLabel = getInquiryStatusLabel(inquiry.status);
  const statusVariant = getInquiryStatusBadgeVariant(inquiry.status);
  const title = getInquiryTitle(inquiry);
  const isProductInquiry = inquiry.inquiryType === "product";

  const images = inquiry.images?.map((image) => ({
    key: image.fileUrl,
    url: image.fileUrl,
    alt: image.fileName,
  }));

  return (
    <article className="chat-detail-page__inquiry-detail">
      <div className="chat-detail-page__message-head">
        <h2 className="chat-detail-page__subject">
          {title}
        </h2>

        <Badge variant={statusVariant}>
          {statusLabel}
        </Badge>
      </div>

      {!isProductInquiry && inquiry.modelMeta ? (
        <InquiryModelMeta modelMeta={inquiry.modelMeta} />
      ) : null}

      {isProductInquiry ? (
        <section className="chat-detail-page__inquiry-section">
          <SectionHeader
            title="問い合わせ内容"
            titleAs="h3"
            className="ui-section-header--title-sm"
          />

          <p className="chat-detail-page__content">
            {inquiry.content}
          </p>

          <ChatImageGrid images={images} />
        </section>
      ) : (
        <ChatImageGrid images={images} />
      )}
    </article>
  );
}

function getInquiryTitle(inquiry: InquiryDetail): string {
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

function getInquiryStatusBadgeVariant(
  status: InquiryDetail["status"],
): InquiryStatusBadgeVariant {
  switch (status) {
    case "open":
      return "warning";
    case "in_progress":
      return "info";
    case "resolved":
      return "success";
    case "closed":
      return "neutral";
  }
}