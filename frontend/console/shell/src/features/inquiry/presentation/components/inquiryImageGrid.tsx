// frontend/console/shell/src/features/inquiry/presentation/components/inquiryImageGrid.tsx

import type {
  InquiryImageFile,
} from "../../../../shared/types/inquiry";

export type InquiryImageGridProps = {
  images?: InquiryImageFile[];
};

type InquiryImageView = {
  id: string;
  fileName: string;
  fileUrl: string;
};

function normalizeImages(
  images: InquiryImageFile[] | undefined,
): InquiryImageView[] {
  if (!images) {
    return [];
  }

  return images.map(
    (
      image: InquiryImageFile,
      index: number,
    ): InquiryImageView => ({
      id:
        image.objectPath ||
        `${image.fileUrl}-${index}`,
      fileName: image.fileName,
      fileUrl: image.fileUrl,
    }),
  );
}

export default function InquiryImageGrid({
  images,
}: InquiryImageGridProps) {
  const imageViews =
    normalizeImages(images);

  if (imageViews.length === 0) {
    return null;
  }

  return (
    <div className="inq-detail__image-grid">
      {imageViews.map(
        (
          image:
            InquiryImageView,
        ) => (
          <a
            key={image.id}
            href={image.fileUrl}
            target="_blank"
            rel="noreferrer"
            className="inq-detail__image-link"
            aria-label={`${image.fileName}を開く`}
          >
            <img
              src={image.fileUrl}
              alt={image.fileName}
              className="inq-detail__image"
              loading="lazy"
            />
          </a>
        ),
      )}
    </div>
  );
}