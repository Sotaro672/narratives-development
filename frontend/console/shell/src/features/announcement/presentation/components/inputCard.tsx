// frontend/console/shell/src/features/announcement/presentation/components/inputCard.tsx

import { useEffect, useMemo, useState } from "react";
import { Image as ImageIcon } from "lucide-react";

import type {
  AnnouncementInputAttachment,
  AnnouncementInputPayload,
} from "../../application/announcement_input";

import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { Input } from "../../../../shared/ui/input";
import { Label } from "../../../../shared/ui/label";
import MediaGallery from "../../../../shared/ui/mediaGallery";
import MediaUploader from "../../../../shared/ui/mediaUploader";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";
import Textarea from "../../../../shared/ui/textarea";

import "./inputCard.css";

export type InputCardMode = "view" | "edit";

type Props = {
  title?: string;
  mode?: InputCardMode;
  initialTitle?: string;
  initialText?: string;
  initialAttachments?: AnnouncementInputAttachment[];
  saving?: boolean;
  sending?: boolean;
  onChange?: (payload: AnnouncementInputPayload) => void;
};

type PreviewImage = {
  key: string;
  url: string;
  name: string;
  revokeOnCleanup: boolean;
};

const EMPTY_INITIAL_ATTACHMENTS: AnnouncementInputAttachment[] = [];

function fileKey(file: File, index: number): string {
  return `${file.name}-${file.size}-${file.lastModified}-${index}`;
}

function getFileIdentity(file: File): string {
  return `${file.name}-${file.size}-${file.lastModified}`;
}

function formatViewText(value: string): string {
  return value.trim() || "-";
}

export default function InputCard({
  title = "入力",
  mode = "edit",
  initialTitle = "",
  initialText = "",
  initialAttachments = EMPTY_INITIAL_ATTACHMENTS,
  saving = false,
  sending = false,
  onChange,
}: Props) {
  const [inputTitle, setInputTitle] = useState(initialTitle);
  const [text, setText] = useState(initialText);
  const [attachments, setAttachments] =
    useState<AnnouncementInputAttachment[]>(initialAttachments);
  const [mainImageIndex, setMainImageIndex] = useState(0);

  const isEditMode = mode === "edit";
  const isBusy = saving || sending;
  const isDisabled = isBusy || !isEditMode;

  useEffect(() => {
    setInputTitle(initialTitle);
  }, [initialTitle]);

  useEffect(() => {
    setText(initialText);
  }, [initialText]);

  useEffect(() => {
    setAttachments(initialAttachments);
    setMainImageIndex(0);
  }, [initialAttachments]);

  useEffect(() => {
    onChange?.({
      title: inputTitle,
      text,
      attachments,
    });
  }, [inputTitle, text, attachments, onChange]);

  const previewImages = useMemo<PreviewImage[]>(() => {
    return attachments.map((attachment, index) => {
      if (attachment.type === "new") {
        return {
          key: fileKey(attachment.file, index),
          url: URL.createObjectURL(attachment.file),
          name: attachment.file.name,
          revokeOnCleanup: true,
        };
      }

      return {
        key: attachment.id,
        url: attachment.fileUrl,
        name: attachment.fileName,
        revokeOnCleanup: false,
      };
    });
  }, [attachments]);

  useEffect(() => {
    return () => {
      previewImages.forEach((item) => {
        if (item.revokeOnCleanup) {
          URL.revokeObjectURL(item.url);
        }
      });
    };
  }, [previewImages]);

  useEffect(() => {
    if (attachments.length === 0) {
      if (mainImageIndex !== 0) {
        setMainImageIndex(0);
      }

      return;
    }

    if (mainImageIndex > attachments.length - 1) {
      setMainImageIndex(attachments.length - 1);
    }
  }, [attachments.length, mainImageIndex]);

  const hasImages = previewImages.length > 0;

  const galleryItems = useMemo(
    () =>
      previewImages.map((item) => ({
        id: item.key,
        src: item.url,
        name: item.name,
        alt: item.name,
        type: "image" as const,
      })),
    [previewImages],
  );

  const addImages = (nextFiles: File[]) => {
    if (!isEditMode || isBusy || nextFiles.length === 0) {
      return;
    }

    setAttachments((previousAttachments) => {
      const existingFileIdentities = previousAttachments
        .filter((attachment) => attachment.type === "new")
        .map((attachment) => getFileIdentity(attachment.file));

      const seen = new Set(existingFileIdentities);
      const merged = [...previousAttachments];
      let firstAddedIndex = -1;

      for (const file of nextFiles) {
        if (!file.type.startsWith("image/")) {
          continue;
        }

        const identity = getFileIdentity(file);

        if (seen.has(identity)) {
          continue;
        }

        seen.add(identity);

        if (firstAddedIndex === -1) {
          firstAddedIndex = merged.length;
        }

        merged.push({
          type: "new",
          file,
        });
      }

      if (firstAddedIndex !== -1) {
        setMainImageIndex(firstAddedIndex);
      }

      return merged;
    });
  };

  const handleRemoveImageAt = (targetIndex: number) => {
    if (!isEditMode || isBusy) {
      return;
    }

    setAttachments((previousAttachments) =>
      previousAttachments.filter((_, index) => index !== targetIndex),
    );

    setMainImageIndex((previousIndex) => {
      if (targetIndex < previousIndex) {
        return previousIndex - 1;
      }

      if (targetIndex === previousIndex) {
        return 0;
      }

      return previousIndex;
    });
  };

  const handleClearImages = () => {
    if (!isEditMode || isBusy) {
      return;
    }

    setAttachments([]);
    setMainImageIndex(0);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>

        {isEditMode && hasImages ? (
          <div className="announcement-input-card__header-actions">
            <MediaUploader
              accept="image/*"
              multiple
              pickerVariant="button"
              title={null}
              showCount={false}
              pickerLabel="画像を追加"
              disabled={isDisabled}
              className="announcement-input-card__header-uploader"
              onFilesSelected={addImages}
            />

            <Button
              type="button"
              variant="destructive-outline"
              size="sm"
              disabled={isDisabled}
              onClick={handleClearImages}
            >
              クリア
            </Button>
          </div>
        ) : null}
      </CardHeader>

      <CardContent>
        <Stack gap="md">
          <div className="card__field">
            <Label>画像アップロード</Label>

            {!hasImages && isEditMode ? (
              <MediaUploader
                accept="image/*"
                multiple
                title={null}
                showCount={false}
                pickerLabel="画像をアップロード"
                pickerDescription="クリックまたはドラッグ＆ドロップで画像を追加できます"
                emptyIcon={<ImageIcon />}
                disabled={isDisabled}
                className="announcement-input-card__uploader"
                onFilesSelected={addImages}
              />
            ) : (
              <MediaGallery
                items={galleryItems}
                activeIndex={mainImageIndex}
                onActiveIndexChange={setMainImageIndex}
                editable={isEditMode}
                deleteDisabled={isDisabled}
                mainVariant="viewer"
                mainFit="contain"
                thumbnailFit="cover"
                emptyIcon={<ImageIcon />}
                emptyText="画像はありません"
                onDelete={
                  isEditMode
                    ? (_item, index) => {
                        handleRemoveImageAt(index);
                      }
                    : undefined
                }
              />
            )}
          </div>

          <div className="card__field">
            <Label htmlFor="sales-input-title">
              タイトル
            </Label>

            {isEditMode ? (
              <Input
                id="sales-input-title"
                type="text"
                value={inputTitle}
                onChange={(event) =>
                  setInputTitle(event.target.value)
                }
                placeholder="タイトルを入力してください"
                disabled={isDisabled}
              />
            ) : (
              <Text
                as="div"
                className="card__view-value"
              >
                {formatViewText(inputTitle)}
              </Text>
            )}
          </div>

          <div className="card__field">
            <Label htmlFor="sales-input-text">
              文章
            </Label>

            {isEditMode ? (
              <Textarea
                id="sales-input-text"
                size="medium"
                value={text}
                onChange={(event) =>
                  setText(event.target.value)
                }
                placeholder="文章を入力してください"
                disabled={isDisabled}
              />
            ) : (
              <Text
                as="div"
                className="card__view-value card__view-value--multiline"
              >
                {formatViewText(text)}
              </Text>
            )}
          </div>
        </Stack>
      </CardContent>
    </Card>
  );
}