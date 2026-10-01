// frontend/mall/src/features/inquiry/presentation/components/InquiryMessageCard.tsx

import { useNavigate } from "react-router-dom";

import Badge from "../../../../components/ui/Badge";
import SectionHeader from "../../../../components/ui/SectionHeader";
import TextLink from "../../../../components/ui/textLink";

import ChatImageGrid from "../../../shared/presentation/components/ChatImageGrid";
import type { InquiryDetail } from "../../../shared/types/inquiryTypes";
import { getInquiryTypeLabel } from "../../../shared/types/inquiryTypes";

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
  const navigate = useNavigate();
  const statusLabel = getInquiryStatusLabel(inquiry.status);
  const statusVariant = getInquiryStatusBadgeVariant(inquiry.status);
  const title = getInquiryTitle(inquiry);
  const isProductInquiry = inquiry.inquiryType === "product";
  const orderId = inquiry.orderId?.trim() ?? "";

  const images = inquiry.images?.map((image) => ({
    key: image.fileUrl,
    url: image.fileUrl,
    alt: image.fileName,
  }));

  const handleOpenOrderDetail = (): void => {
    if (!orderId) {
      return;
    }

    navigate(`/orders/${encodeURIComponent(orderId)}`);
  };

  return (
    <article className="chat-detail-page__inquiry-detail">
      <div className="chat-detail-page__message-head">
        <h2 className="chat-detail-page__subject">
          {title}
        </h2>

        <div className="chat-detail-page__header-meta">
          {!isProductInquiry && orderId ? (
            <TextLink onClick={handleOpenOrderDetail}>
              注文詳細を見る
            </TextLink>
          ) : null}

          <Badge variant={statusVariant}>
            {statusLabel}
          </Badge>
        </div>
      </div>

      {isProductInquiry ? (
        <section className="chat-detail-page__inquiry-section">
          <SectionHeader
            title="問い合わせ内容"
            titleAs="h3"
            titleSize="sm"
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
  const inquiryLabel = getInquiryTypeLabel(inquiry.inquiryType);
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