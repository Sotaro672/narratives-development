// frontend/amol/src/features/shared/presentation/components/ChatImageGrid.tsx

export type ChatImageGridItem = {
  key: string;
  url: string;
  alt?: string;
};

export type ChatImageGridProps = {
  images?: ChatImageGridItem[] | null;
  defaultAlt?: string;
};

export default function ChatImageGrid({
  images,
  defaultAlt = "画像",
}: ChatImageGridProps) {
  if (!images?.length) {
    return null;
  }

  return (
    <div className="chat-detail-page__images">
      {images.map((image) => (
        <a
          key={image.key}
          href={image.url}
          target="_blank"
          rel="noreferrer"
          className="chat-detail-page__image-link"
        >
          <img
            src={image.url}
            alt={image.alt || defaultAlt}
            className="chat-detail-page__image"
            loading="lazy"
          />
        </a>
      ))}
    </div>
  );
}