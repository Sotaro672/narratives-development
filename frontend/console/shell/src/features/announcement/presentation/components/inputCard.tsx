// frontend/console/shell/src/features/announcement/presentation/components/inputCard.tsx

import { useEffect, useMemo, useState } from "react";
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
import { Input } from "../../../../shared/ui/input";
import { Label } from "../../../../shared/ui/label";
import Media from "../../../../shared/ui/media";
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

function ImageIcon() {
  return (
    <svg
      width="28"
      height="28"
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
    >
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
    <svg
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
    >
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
  const [attachments, setAttachments] =
    useState<AnnouncementInputAttachment[]>(initialAttachments);
  const [mainImageIndex, setMainImageIndex] = useState(0);

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
  }, [
    inputTitle,
    text,
    attachments,
    onChange,
  ]);

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
  }, [
    attachments,
    mainImageIndex,
  ]);

  const hasImages = previewImages.length > 0;
  const mainImage = previewImages[mainImageIndex] ?? null;

  const thumbIndices = previewImages
    .map((_, index) => index)
    .filter((index) => index !== mainImageIndex);

  const addImages = (nextFiles: File[]) => {
    if (
      !isEditMode ||
      isBusy ||
      nextFiles.length === 0
    ) {
      return;
    }

    setAttachments((previousAttachments) => {
      const existingFileIdentities = previousAttachments
        .filter(
          (attachment) =>
            attachment.type === "new",
        )
        .map((attachment) =>
          getFileIdentity(attachment.file),
        );

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

  const handleDropImages = (
    event: React.DragEvent<HTMLDivElement>,
  ) => {
    event.preventDefault();
    event.stopPropagation();

    if (!isEditMode || isBusy) {
      return;
    }

    addImages(
      Array.from(event.dataTransfer.files ?? []),
    );
  };

  const handleDragOverImages = (
    event: React.DragEvent<HTMLDivElement>,
  ) => {
    event.preventDefault();
    event.stopPropagation();
  };

  const handleRemoveImageAt = (
    targetIndex: number,
  ) => {
    if (!isEditMode || isBusy) {
      return;
    }

    setAttachments((previousAttachments) =>
      previousAttachments.filter(
        (_, index) => index !== targetIndex,
      ),
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

  const handleSelectMainImage = (
    index: number,
  ) => {
    if (!isEditMode) {
      return;
    }

    setMainImageIndex(index);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>

      <CardContent>
        <Stack gap="md">
          <div className="card__field">
            <div className="announcement-input-card__section-header">
              <Label>画像アップロード</Label>

              {isEditMode && hasImages ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  disabled={isDisabled}
                  onClick={handleClearImages}
                >
                  クリア
                </Button>
              ) : null}
            </div>

            <MediaUploader
              items={[]}
              accept="image/*"
              multiple
              title={null}
              showCount={false}
              showPicker={false}
              showFileNames={false}
              showRemoveButton={false}
              previewFramed={false}
              disabled={isDisabled}
              className="announcement-input-card__uploader"
              onFilesSelected={addImages}
              renderEmpty={({ openPicker }) => (
                <div className="announcement-input-card__image-panel">
                  {!hasImages && isEditMode ? (
                    <div
                      onDrop={handleDropImages}
                      onDragOver={handleDragOverImages}
                      title="クリックで画像を追加"
                    >
                      <Media
                        className="announcement-input-card__empty-media"
                        emptyIcon={<ImageIcon />}
                        emptyText="画像を追加"
                        emptyDescription="クリックで選択（複数可） / ドロップでも追加できます"
                        onActivate={openPicker}
                        disabled={isBusy}
                      />
                    </div>
                  ) : null}

                  {!hasImages && isViewMode ? (
                    <Media
                      className="announcement-input-card__empty-media"
                      emptyIcon={<ImageIcon />}
                      emptyText="画像はありません"
                    />
                  ) : null}

                  {hasImages ? (
                    <div className="announcement-input-card__images">
                      <div
                        className="announcement-input-card__main-image-wrap"
                        onDrop={
                          isEditMode
                            ? handleDropImages
                            : undefined
                        }
                        onDragOver={
                          isEditMode
                            ? handleDragOverImages
                            : undefined
                        }
                        title={
                          isEditMode
                            ? "クリックで画像追加"
                            : undefined
                        }
                      >
                        <Media
                          src={mainImage?.url}
                          alt={mainImage?.name}
                          name={mainImage?.name}
                          variant="viewer"
                          fit="contain"
                          className="announcement-input-card__main-media"
                          onActivate={
                            isEditMode
                              ? openPicker
                              : undefined
                          }
                          disabled={isBusy}
                        />

                        {isEditMode ? (
                          <DeleteButton
                            size="md"
                            disabled={isDisabled}
                            ariaLabel="remove main image"
                            onClick={(event) => {
                              event.stopPropagation();
                              handleRemoveImageAt(
                                mainImageIndex,
                              );
                            }}
                          />
                        ) : null}

                        <Text
                          as="div"
                          size="xs"
                          tone="muted"
                          className="announcement-input-card__image-meta"
                        >
                          {isEditMode
                            ? `${previewImages.length} 枚（×で削除 / クリックで追加）`
                            : `${previewImages.length} 枚`}
                        </Text>
                      </div>

                      <div className="announcement-input-card__thumbnail-grid">
                        {thumbIndices.map((index) => {
                          const item =
                            previewImages[index];

                          if (!item) {
                            return null;
                          }

                          return (
                            <div
                              key={item.key}
                              className="announcement-input-card__thumbnail"
                              title={
                                isEditMode
                                  ? "クリックでメインに設定"
                                  : undefined
                              }
                            >
                              <Media
                                src={item.url}
                                alt={item.name}
                                name={item.name}
                                variant="square"
                                fit="cover"
                                className="announcement-input-card__thumbnail-media"
                                onActivate={
                                  isEditMode
                                    ? () =>
                                        handleSelectMainImage(
                                          index,
                                        )
                                    : undefined
                                }
                                disabled={isBusy}
                              />

                              {isEditMode ? (
                                <DeleteButton
                                  size="sm"
                                  disabled={isDisabled}
                                  ariaLabel="remove image"
                                  onClick={(event) => {
                                    event.stopPropagation();
                                    handleRemoveImageAt(
                                      index,
                                    );
                                  }}
                                />
                              ) : null}
                            </div>
                          );
                        })}

                        {isEditMode ? (
                          <div
                            onDrop={handleDropImages}
                            onDragOver={
                              handleDragOverImages
                            }
                            title="クリックで画像を追加"
                          >
                            <Media
                              variant="square"
                              className="announcement-input-card__add-media"
                              emptyIcon={<PlusIcon />}
                              emptyText="画像を追加"
                              onActivate={openPicker}
                              disabled={isBusy}
                            />
                          </div>
                        ) : null}
                      </div>
                    </div>
                  ) : null}
                </div>
              )}
            />
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
                  setInputTitle(
                    event.target.value,
                  )
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