// frontend/mall/src/features/shared/presentation/components/ChatMessageBubble.tsx

import {
  useState,
  type ReactNode,
} from "react";

import Card from "../../../../components/ui/Card";
import Preview from "../../../../components/ui/Preview";
import ChatMessageHeader from "./ChatMessageHeader";

export type ChatMessageBubbleImage = {
  fileName?: string | null;
  fileUrl: string;
  mimeType?: string | null;
  objectPath?: string | null;
};

export type ChatMessageBubbleProps = {
  senderName: string;
  senderIcon?: string | null;
  createdAt?: string | null;
  content?: string | null;
  images?: ChatMessageBubbleImage[];
  isMine?: boolean;
  isSystem?: boolean;
  action?: ReactNode;
  afterContent?: ReactNode;
  className?: string;
};

function joinClassNames(
  ...classNames: Array<string | undefined | false>
): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ChatMessageBubble({
  senderName,
  senderIcon,
  createdAt,
  content,
  images = [],
  isMine = false,
  isSystem = false,
  action,
  afterContent,
  className,
}: ChatMessageBubbleProps) {
  const [previewImageIndex, setPreviewImageIndex] = useState<number | null>(null);

  const bubbleClassName = joinClassNames(
    "chat-detail-page__reply",
    isMine && "chat-detail-page__reply--avatar",
    isSystem && "chat-detail-page__reply--system",
    className,
  );

  const previewImage =
    previewImageIndex !== null
      ? images[previewImageIndex] ?? null
      : null;

  const handlePreviewPrevious = (): void => {
    if (
      previewImageIndex === null ||
      images.length <= 1
    ) {
      return;
    }

    setPreviewImageIndex(
      previewImageIndex === 0
        ? images.length - 1
        : previewImageIndex - 1,
    );
  };

  const handlePreviewNext = (): void => {
    if (
      previewImageIndex === null ||
      images.length <= 1
    ) {
      return;
    }

    setPreviewImageIndex(
      previewImageIndex === images.length - 1
        ? 0
        : previewImageIndex + 1,
    );
  };

  return (
    <>
      <Card
        as="article"
        padding="md"
        className={bubbleClassName}
      >
        <ChatMessageHeader
          name={senderName}
          icon={senderIcon}
          createdAt={createdAt}
          action={action}
          showAvatar={!isSystem}
        />

        {content ? (
          <p className="chat-detail-page__content">
            {content}
          </p>
        ) : null}

        {images.length > 0 ? (
          <div
            className="chat-detail-page__images"
            aria-label="添付画像"
          >
            {images.map((image, index) => (
              <a
                key={
                  image.objectPath ||
                  `${image.fileUrl}-${index}`
                }
                href={image.fileUrl}
                className="chat-detail-page__image-link"
                aria-label={`${
                  image.fileName ||
                  `添付画像${index + 1}`
                }を表示`}
                onClick={(event) => {
                  event.preventDefault();
                  setPreviewImageIndex(index);
                }}
              >
                <img
                  src={image.fileUrl}
                  alt={
                    image.fileName ||
                    `添付画像${index + 1}`
                  }
                  className="chat-detail-page__image"
                  loading="lazy"
                  decoding="async"
                />
              </a>
            ))}
          </div>
        ) : null}

        {afterContent}
      </Card>

      <Preview
        open={previewImage !== null}
        src={previewImage?.fileUrl}
        alt={
          previewImage?.fileName ||
          "メッセージの添付画像"
        }
        type={
          previewImage?.mimeType ||
          "image"
        }
        onClose={() => {
          setPreviewImageIndex(null);
        }}
        onPrev={
          images.length > 1
            ? handlePreviewPrevious
            : undefined
        }
        onNext={
          images.length > 1
            ? handlePreviewNext
            : undefined
        }
      />
    </>
  );
}