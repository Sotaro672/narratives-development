// frontend/console/shell/src/features/announcement/presentation/components/inputCard.tsx

import { useEffect, useMemo, useRef, useState } from "react";
import type * as React from "react";

import type {
  AnnouncementInputAttachment,
  AnnouncementInputPayload,
} from "../../application/announcement_input";

import { Button } from "../../../../shared/ui/button";
import DeleteButton from "../../../../shared/ui/delete";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { Label } from "../../../../shared/ui/label";
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

function ImageIcon() {
  return (
    <svg width="28" height="28" viewBox="0 0 24 24" fill="none">
      <path
        d="M21 19V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2Z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
      <path
        d="M8.5 10.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
      <path
        d="M21 16l-5.5-5.5a2 2 0 0 0-2.8 0L5 18"
        stroke="currentColor"
        strokeWidth="1.6"
      />
    </svg>
  );
}

function PlusIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
      <path
        d="M12 5v14M5 12h14"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
    </svg>
  );
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
  const [attachments, setAttachments] = useState<AnnouncementInputAttachment[]>(initialAttachments);
  const [mainImageIndex, setMainImageIndex] = useState(0);
  const imageInputRef = useRef<HTMLInputElement | null>(null);

  const isEditMode = mode === "edit";
  const isViewMode = mode === "view";
  const isBusy = saving || sending;
  const isDisabled = isBusy || isViewMode;

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
  }, [attachments, mainImageIndex]);

  const hasImages = previewImages.length > 0;
  const mainImage = previewImages[mainImageIndex] ?? null;
  const thumbIndices = previewImages
    .map((_, index) => index)
    .filter((index) => index !== mainImageIndex);

  const openPicker = () => {
    if (!isEditMode || isBusy) return;
    imageInputRef.current?.click();
  };

  const addImages = (nextFiles: File[]) => {
    if (!isEditMode || nextFiles.length === 0) return;

    setAttachments((previousAttachments) => {
      const existingFileIdentities = previousAttachments
        .filter((attachment) => attachment.type === "new")
        .map((attachment) => getFileIdentity(attachment.file));

      const seen = new Set(existingFileIdentities);
      const merged = [...previousAttachments];
      let firstAddedIndex = -1;

      for (const file of nextFiles) {
        if (!file.type.startsWith("image/")) continue;

        const identity = getFileIdentity(file);
        if (seen.has(identity)) continue;

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

  const handleSelectImages = (event: React.ChangeEvent<HTMLInputElement>) => {
    const nextFiles = Array.from(event.target.files ?? []);
    addImages(nextFiles);
    event.target.value = "";
  };

  const handleDropImages = (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();

    if (!isEditMode || isBusy) return;

    const nextFiles = Array.from(event.dataTransfer.files ?? []);
    addImages(nextFiles);
  };

  const handleDragOverImages = (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
  };

  const handleRemoveImageAt = (targetIndex: number) => {
    if (!isEditMode || isBusy) return;

    setAttachments((previousAttachments) =>
      previousAttachments.filter((_, index) => index !== targetIndex),
    );

    setMainImageIndex((previousIndex) => {
      if (targetIndex < previousIndex) return previousIndex - 1;
      if (targetIndex === previousIndex) return 0;
      return previousIndex;
    });
  };

  const handleClearImages = () => {
    if (!isEditMode || isBusy) return;
    setAttachments([]);
    setMainImageIndex(0);
  };

  const handleSelectMainImage = (index: number) => {
    if (!isEditMode) return;
    setMainImageIndex(index);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>

      <CardContent>
        <div className="announcement-input-card__content">
          <div className="announcement-input-card__section">
            <div className="announcement-input-card__section-header">
              <Label>画像アップロード</Label>

              {isEditMode && hasImages ? (
                <Button
                  type="button"
                  variant="ghost"
                  className="announcement-input-card__clear-button"
                  disabled={isDisabled}
                  onClick={handleClearImages}
                >
                  クリア
                </Button>
              ) : null}
            </div>

            {isEditMode ? (
              <input
                ref={imageInputRef}
                type="file"
                accept="image/*"
                multiple
                className="announcement-input-card__file-input"
                onChange={handleSelectImages}
              />
            ) : null}

            <div className="announcement-input-card__image-panel">
              {!hasImages && isEditMode ? (
                <div
                  className="announcement-input-card__empty-upload"
                  onClick={openPicker}
                  onDrop={handleDropImages}
                  onDragOver={handleDragOverImages}
                  role="button"
                  tabIndex={0}
                  title="クリックで画像を追加"
                >
                  <div className="announcement-input-card__empty-icon">
                    <ImageIcon />
                  </div>
                  <div className="announcement-input-card__empty-title">
                    画像を追加
                  </div>
                  <div className="announcement-input-card__empty-help">
                    クリックで選択（複数可） / ドロップでも追加できます
                  </div>
                </div>
              ) : null}

              {!hasImages && isViewMode ? (
                <div className="announcement-input-card__empty-view">
                  <div className="announcement-input-card__empty-icon">
                    <ImageIcon />
                  </div>
                  <div className="announcement-input-card__empty-title">
                    画像はありません
                  </div>
                </div>
              ) : null}

              {hasImages ? (
                <div className="announcement-input-card__images">
                  <div
                    className="announcement-input-card__main-image-wrap"
                    onDrop={isEditMode ? handleDropImages : undefined}
                    onDragOver={isEditMode ? handleDragOverImages : undefined}
                    title={isEditMode ? "クリックで画像追加" : undefined}
                  >
                    <div
                      className={[
                        "announcement-input-card__main-image-stage",
                        isEditMode ? "announcement-input-card__main-image-stage--clickable" : "",
                      ].filter(Boolean).join(" ")}
                      onClick={openPicker}
                      role={isEditMode ? "button" : undefined}
                      tabIndex={isEditMode ? 0 : undefined}
                    >
                      {mainImage ? (
                        <img
                          src={mainImage.url}
                          alt={mainImage.name}
                          className="announcement-input-card__main-image"
                        />
                      ) : null}
                    </div>

                    {isEditMode ? (
                      <DeleteButton
                        size="md"
                        disabled={isDisabled}
                        ariaLabel="remove main image"
                        onClick={(event) => {
                          event.stopPropagation();
                          handleRemoveImageAt(mainImageIndex);
                        }}
                      />
                    ) : null}

                    <div className="announcement-input-card__image-meta">
                      <div>
                        {isEditMode
                          ? `${previewImages.length} 枚（×で削除 / クリックで追加）`
                          : `${previewImages.length} 枚`}
                      </div>
                    </div>
                  </div>

                  <div className="announcement-input-card__thumbnail-grid">
                    {thumbIndices.map((index) => {
                      const item = previewImages[index];
                      if (!item) return null;

                      return (
                        <div
                          key={item.key}
                          className={[
                            "announcement-input-card__thumbnail",
                            isEditMode ? "announcement-input-card__thumbnail--clickable" : "",
                          ].filter(Boolean).join(" ")}
                          onClick={() => handleSelectMainImage(index)}
                          role={isEditMode ? "button" : undefined}
                          tabIndex={isEditMode ? 0 : undefined}
                          title={isEditMode ? "クリックでメインに設定" : undefined}
                        >
                          <div className="announcement-input-card__thumbnail-image-wrap">
                            <img
                              src={item.url}
                              alt={item.name}
                              className="announcement-input-card__thumbnail-image"
                            />
                          </div>

                          {isEditMode ? (
                            <DeleteButton
                              size="sm"
                              disabled={isDisabled}
                              ariaLabel="remove image"
                              onClick={(event) => {
                                event.stopPropagation();
                                handleRemoveImageAt(index);
                              }}
                            />
                          ) : null}
                        </div>
                      );
                    })}

                    {isEditMode ? (
                      <div
                        className="announcement-input-card__add-image"
                        onClick={openPicker}
                        onDrop={handleDropImages}
                        onDragOver={handleDragOverImages}
                        role="button"
                        tabIndex={0}
                        title="クリックで画像を追加"
                      >
                        <div className="announcement-input-card__add-image-icon">
                          <PlusIcon />
                        </div>
                        <div className="announcement-input-card__add-image-label">
                          画像を追加
                        </div>
                      </div>
                    ) : null}
                  </div>
                </div>
              ) : null}
            </div>
          </div>

          <div className="announcement-input-card__section">
            <Label htmlFor="sales-input-title">タイトル</Label>

            {isEditMode ? (
              <input
                id="sales-input-title"
                type="text"
                value={inputTitle}
                onChange={(event) => setInputTitle(event.target.value)}
                placeholder="タイトルを入力してください"
                disabled={isDisabled}
                className="announcement-input-card__title-input"
              />
            ) : (
              <div className="announcement-input-card__title-view">
                {formatViewText(inputTitle)}
              </div>
            )}
          </div>

          <div className="announcement-input-card__section">
            <Label htmlFor="sales-input-text">文章</Label>

            {isEditMode ? (
              <Textarea
                id="sales-input-text"
                size="medium"
                value={text}
                onChange={(event) => setText(event.target.value)}
                placeholder="文章を入力してください"
                disabled={isDisabled}
              />
            ) : (
              <div className="announcement-input-card__text-view">
                {formatViewText(text)}
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}