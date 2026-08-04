// frontend/amol/src/features/inquiry/presentation/components/InquiryImageGrid.tsx

import type {
  InquiryImage,
} from "../../api/inquiryApi";

type InquiryImageGridProps = {
  images?: InquiryImage[] | null;
};

export default function InquiryImageGrid({
  images,
}: InquiryImageGridProps) {
  if (
    !Array.isArray(images) ||
    images.length === 0
  ) {
    return null;
  }

  return (
    <div className="chat-detail-page__images">
      {images.map((image, index) => {
        const src = getImageSrc(image);

        if (!src) {
          return null;
        }

        const label = getImageLabel(
          image,
          index,
        );

        return (
          <a
            key={`${getImageKey(
              image,
              src,
            )}-${index}`}
            className="chat-detail-page__image-link"
            href={src}
            target="_blank"
            rel="noreferrer"
          >
            <img
              className="chat-detail-page__image"
              src={src}
              alt={label}
              loading="lazy"
            />
          </a>
        );
      })}
    </div>
  );
}

function getImageSrc(
  image: InquiryImage,
): string {
  return image.fileUrl || "";
}

function getImageLabel(
  image: InquiryImage,
  index: number,
): string {
  if (image.fileName) {
    return image.fileName;
  }

  return `添付画像 ${index + 1}`;
}

function getImageKey(
  image: InquiryImage,
  src: string,
): string {
  if (image.objectPath) {
    return image.objectPath;
  }

  return src;
}