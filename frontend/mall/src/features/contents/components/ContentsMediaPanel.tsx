// frontend/mall/src/features/contents/components/ContentsMediaPanel.tsx

import type { MediaGalleryItem } from "../../../components/ui/MediaGallery";
import MediaGallery from "../../../components/ui/MediaGallery";

type ContentsMediaPanelProps = {
  loading: boolean;
  error: string;
  metadataUri: string;
  moderationHidden: boolean;
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
  moderationHidden,
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

      {!loading && moderationHidden ? (
        <MediaGallery
          items={[]}
          activeIndex={0}
          altFallback={tokenName || "トークンコンテンツ"}
          placeholderText="不適切な内容として削除されました。"
          className="contents-page-media-gallery"
          onPrev={onPrevFile}
          onNext={onNextFile}
          onSelect={onSelectFile}
        />
      ) : null}

      {!loading && !moderationHidden && error ? (
        <p className="contents-page-card__error">{error}</p>
      ) : null}

      {!loading && !moderationHidden && !error && !metadataUri ? (
        <p className="contents-page-card__error">
          metadataUri が指定されていません。
        </p>
      ) : null}

      {!loading &&
      !moderationHidden &&
      !error &&
      metadataUri &&
      !hasMediaItems ? (
        <p className="contents-page-card__message">
          表示できるコンテンツはまだありません。
        </p>
      ) : null}

      {!loading &&
      !moderationHidden &&
      !error &&
      hasMediaItems ? (
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