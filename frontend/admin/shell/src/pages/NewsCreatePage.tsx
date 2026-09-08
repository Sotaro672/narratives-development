// frontend/admin/shell/src/pages/NewsCreatePage.tsx

import { useEffect, useRef, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { useNavigate } from "react-router-dom";

import NewsUploadProgressModal from "../features/news/presentation/components/NewsUploadProgressModal";
import { useNews } from "../features/news/presentation/hooks/useNews";
import Button from "../shared/ui/Button/Button";
import Page, { PageHeader } from "../shared/ui/Page/Page";

import "./NewsCreatePage.css";

const NEWS_CREATE_FORM_ID = "news-create-form";
const MAX_NEWS_IMAGE_SIZE = 5 * 1024 * 1024;
const ALLOWED_NEWS_IMAGE_TYPES = new Set([
  "image/jpeg",
  "image/png",
  "image/webp",
]);

export default function NewsCreatePage() {
  const navigate = useNavigate();
  const imageInputRef = useRef<HTMLInputElement | null>(null);

  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [image, setImage] = useState<File | null>(null);
  const [imageAlt, setImageAlt] = useState("");
  const [imagePreviewUrl, setImagePreviewUrl] = useState<string | null>(null);
  const [imageError, setImageError] = useState<string | null>(null);

  const {
    publishing,
    publishError,
    uploadingImage,
    uploadProgress,
    uploadFileName,
    publish,
  } = useNews();

  useEffect(() => {
    if (!image) {
      setImagePreviewUrl(null);
      return;
    }

    const objectUrl = URL.createObjectURL(image);
    setImagePreviewUrl(objectUrl);

    return () => {
      URL.revokeObjectURL(objectUrl);
    };
  }, [image]);

  const canPublish =
    title.length > 0 &&
    body.length > 0 &&
    !imageError &&
    !publishing;

  const clearImage = () => {
    setImage(null);
    setImageAlt("");
    setImageError(null);

    if (imageInputRef.current) {
      imageInputRef.current.value = "";
    }
  };

  const handleImageChange = (
    event: ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0] ?? null;

    if (!file) {
      clearImage();
      return;
    }

    if (!ALLOWED_NEWS_IMAGE_TYPES.has(file.type)) {
      setImage(null);
      setImageAlt("");
      setImageError(
        "JPEG、PNG、WebP形式の画像を選択してください。",
      );
      event.target.value = "";
      return;
    }

    if (file.size <= 0) {
      setImage(null);
      setImageAlt("");
      setImageError(
        "空の画像ファイルは選択できません。",
      );
      event.target.value = "";
      return;
    }

    if (file.size > MAX_NEWS_IMAGE_SIZE) {
      setImage(null);
      setImageAlt("");
      setImageError(
        "画像サイズは5MB以下にしてください。",
      );
      event.target.value = "";
      return;
    }

    setImage(file);
    setImageError(null);
  };

  const handleSubmit = async (
    event: FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault();

    if (
      title.length === 0 ||
      body.length === 0 ||
      imageError ||
      publishing
    ) {
      return;
    }

    const confirmed = window.confirm(
      "ConsoleとMallの全ユーザーへこの通知を公開します。公開後は編集・削除できません。通知しますか？",
    );

    if (!confirmed) {
      return;
    }

    const created = await publish({
      title,
      body,
      image: image ?? undefined,
      imageAlt: image ? imageAlt : undefined,
    });

    if (!created) {
      return;
    }

    navigate("/news", {
      replace: true,
    });
  };

  return (
    <Page>
      <PageHeader
        title="通知を作成"
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="通知一覧へ戻る"
            onClick={() => navigate("/news")}
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M19 12H5" />
              <path d="M12 19l-7-7 7-7" />
            </svg>
          </button>
        }
        actions={
          <Button
            type="submit"
            form={NEWS_CREATE_FORM_ID}
            variant="primary"
            loading={publishing}
            disabled={!canPublish}
          >
            {publishing ? "通知中..." : "通知する"}
          </Button>
        }
      />

      <section className="news-create-page__section">
        <div className="news-create-page__section-header">
          <h2 className="news-create-page__section-title">
            システム通知
          </h2>
          <p className="news-create-page__section-description">
            ConsoleとMallの全ユーザーへシステムに関するお知らせを通知します。
          </p>
        </div>

        <form
          id={NEWS_CREATE_FORM_ID}
          className="news-create-page__form"
          onSubmit={handleSubmit}
        >
          <div className="news-create-page__field">
            <label
              className="news-create-page__label"
              htmlFor="news-title"
            >
              タイトル
            </label>
            <input
              id="news-title"
              className="news-create-page__input"
              type="text"
              value={title}
              disabled={publishing}
              autoComplete="off"
              onChange={(event) => setTitle(event.target.value)}
            />
          </div>

          <div className="news-create-page__field">
            <label
              className="news-create-page__label"
              htmlFor="news-body"
            >
              本文
            </label>
            <textarea
              id="news-body"
              className="news-create-page__textarea"
              value={body}
              disabled={publishing}
              rows={8}
              onChange={(event) => setBody(event.target.value)}
            />
          </div>

          <div className="news-create-page__field">
            <label
              className="news-create-page__label"
              htmlFor="news-image"
            >
              画像（任意）
            </label>
            <p className="news-create-page__field-description">
              JPEG、PNG、WebP形式、5MB以下の画像を1枚添付できます。
            </p>

            <input
              ref={imageInputRef}
              id="news-image"
              className="news-create-page__file-input"
              type="file"
              accept="image/jpeg,image/png,image/webp"
              disabled={publishing}
              onChange={handleImageChange}
            />

            {imageError ? (
              <p
                className="news-create-page__error"
                role="alert"
              >
                {imageError}
              </p>
            ) : null}

            {image && imagePreviewUrl ? (
              <div className="news-create-page__image">
                <img
                  className="news-create-page__image-preview"
                  src={imagePreviewUrl}
                  alt={imageAlt || "添付画像のプレビュー"}
                />

                <div className="news-create-page__image-meta">
                  <span className="news-create-page__image-name">
                    {image.name}
                  </span>
                  <span className="news-create-page__image-size">
                    {(image.size / 1024 / 1024).toFixed(2)} MB
                  </span>
                </div>

                <div className="news-create-page__field">
                  <label
                    className="news-create-page__label"
                    htmlFor="news-image-alt"
                  >
                    画像の代替テキスト（任意）
                  </label>
                  <input
                    id="news-image-alt"
                    className="news-create-page__input"
                    type="text"
                    value={imageAlt}
                    disabled={publishing}
                    autoComplete="off"
                    onChange={(event) =>
                      setImageAlt(event.target.value)
                    }
                  />
                </div>

                <button
                  type="button"
                  className="news-create-page__image-remove"
                  disabled={publishing}
                  onClick={clearImage}
                >
                  画像を削除
                </button>
              </div>
            ) : null}
          </div>

          {publishError ? (
            <p
              className="news-create-page__error"
              role="alert"
            >
              通知に失敗しました。{publishError}
            </p>
          ) : null}
        </form>
      </section>

      <NewsUploadProgressModal
        open={uploadingImage}
        progress={uploadProgress}
        fileName={uploadFileName}
      />
    </Page>
  );
}