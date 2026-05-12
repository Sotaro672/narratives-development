//frontend\amol\src\features\contents\components\ContentsMediaPanel.tsx
import type { MediaGalleryItem } from "../../../components/ui/MediaGallery";
import MediaGallery from "../../../components/ui/MediaGallery";

type ContentsMediaPanelProps = {
  loading: boolean;
  error: string;
  metadataUri: string;
  hasMediaItems: boolean;
  mediaItems: MediaGalleryItem[];
  activeFileIndex: number;
  tokenName: string;
  onPrevFile: () => void;
  onNextFile: () => void;
  onSelectFile: (index: number) => void;
};

export default function ContentsMediaPanel({
  loading,
  error,
  metadataUri,
  hasMediaItems,
  mediaItems,
  activeFileIndex,
  tokenName,
  onPrevFile,
  onNextFile,
  onSelectFile,
}: ContentsMediaPanelProps) {
  return (
    <div className="split-page-left contents-page-media-area">
      {loading ? (
        <p className="contents-page-card__message">読み込み中です...</p>
      ) : null}

      {!loading && error ? (
        <p className="contents-page-card__error">{error}</p>
      ) : null}

      {!loading && !error && !metadataUri ? (
        <p className="contents-page-card__error">
          metadataUri が指定されていません。
        </p>
      ) : null}

      {!loading && !error && metadataUri && !hasMediaItems ? (
        <p className="contents-page-card__message">
          表示できるコンテンツはまだありません。
        </p>
      ) : null}

      {!loading && !error && hasMediaItems ? (
        <MediaGallery
          items={mediaItems}
          activeIndex={activeFileIndex}
          altFallback={tokenName || "トークンコンテンツ"}
          className="contents-page-media-gallery"
          onPrev={onPrevFile}
          onNext={onNextFile}
          onSelect={onSelectFile}
        />
      ) : null}
    </div>
  );
}