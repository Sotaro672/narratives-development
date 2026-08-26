// frontend/console/shell/src/features/inquiry/presentation/components/inquiryContentCard.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

import type {
  InquiryImageFile,
} from "../../../../shared/types/inquiry";

export type InquiryContentCardProps = {
  content?: string | null;
  images?: InquiryImageFile[];
  errorMessage?: string | null;
};

type InquiryImageView = {
  id: string;
  fileName: string;
  fileUrl: string;
};

function textOrDash(
  value: string | null | undefined,
): string {
  const normalized = String(value ?? "").trim();

  return normalized || "-";
}

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

export default function InquiryContentCard({
  content,
  images,
  errorMessage,
}: InquiryContentCardProps) {
  const body = textOrDash(content);
  const imageViews = normalizeImages(images);

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          問い合わせ内容
        </CardTitle>
      </CardHeader>

      <CardContent>
        <div className="inq-detail">
          {errorMessage ? (
            <div className="inq__empty">
              {errorMessage}
            </div>
          ) : null}

          <div className="inq-detail__body">
            <div className="inq-detail__label">
              問い合わせ本文
            </div>

            <p className="inq-detail__text">
              {body}
            </p>
          </div>

          {imageViews.length > 0 ? (
            <div className="inq-detail__body">
              <div className="inq-detail__label">
                添付画像
              </div>

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
            </div>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}