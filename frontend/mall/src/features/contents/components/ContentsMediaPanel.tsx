// frontend/mall/src/features/contents/components/ContentsMediaPanel.tsx

import type { MediaGalleryItem } from "../../../components/ui/MediaGallery";
import MediaGallery from "../../../components/ui/MediaGallery";
import TextState from "../../../components/ui/TextState";

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
    <div className="split-page-left">
      {loading ? (
        <TextState variant="loading">読み込み中です...</TextState>
      ) : null}

      {!loading && moderationHidden ? (
        <MediaGallery
          items={[]}
          activeIndex={0}
          altFallback={tokenName || "トークンコンテンツ"}
          placeholderText="不適切な内容として削除されました。"
          variant="fill"
          onPrev={onPrevFile}
          onNext={onNextFile}
          onSelect={onSelectFile}
        />
      ) : null}

      {!loading && !moderationHidden && error ? (
        <TextState variant="error">{error}</TextState>
      ) : null}

      {!loading && !moderationHidden && !error && !metadataUri ? (
        <TextState variant="error">
          metadataUri が指定されていません。
        </TextState>
      ) : null}

      {!loading &&
      !moderationHidden &&
      !error &&
      metadataUri &&
      !hasMediaItems ? (
        <TextState variant="empty">
          表示できるコンテンツはまだありません。
        </TextState>
      ) : null}

      {!loading &&
      !moderationHidden &&
      !error &&
      hasMediaItems ? (
        <MediaGallery
          items={mediaItems}
          activeIndex={activeFileIndex}
          altFallback={tokenName || "トークンコンテンツ"}
          variant="fill"
          onPrev={onPrevFile}
          onNext={onNextFile}
          onSelect={onSelectFile}
        />
      ) : null}
    </div>
  );
}