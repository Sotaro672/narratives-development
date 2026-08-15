// frontend/amol/src/features/inquiry/presentation/components/InquiryImageGrid.tsx

import type { InquiryImage } from "../../api/inquiryApi";

type InquiryImageGridProps = {
  images?: InquiryImage[];
};

export default function InquiryImageGrid({
  images,
}: InquiryImageGridProps) {
  if (!images?.length) {
    return null;
  }

  return (
    <div className="chat-detail-page__images">
      {images.map((image) => (
        <a
          key={image.fileUrl}
          className="chat-detail-page__image-link"
          href={image.fileUrl}
          target="_blank"
          rel="noreferrer"
        >
          <img
            className="chat-detail-page__image"
            src={image.fileUrl}
            alt={image.fileName}
            loading="lazy"
          />
        </a>
      ))}
    </div>
  );
}