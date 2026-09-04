// frontend/admin/shell/src/pages/InquiryDetailPage.tsx
import { useEffect, useMemo, useState } from "react";
import { getDownloadURL, ref } from "firebase/storage";
import { useLocation, useNavigate, useParams } from "react-router-dom";

import { storage } from "../auth/infrastructure/firebaseClient";
import {
  getContact,
  type Contact,
} from "../features/contact/infrastructure/contactApi";
import Page, {
  DetailPageBody,
  PageHeader,
} from "../shared/ui/Page/Page";

type InquiryDetailLocationState = {
  contact?: Contact;
};

type ContactAttachment = {
  fileName: string;
  storagePath: string;
  contentType: string;
  size: number | null;
};

type AttachmentImage = ContactAttachment & {
  imageUrl: string;
};

type ParsedContactMessage = {
  message: string;
  attachments: ContactAttachment[];
};

const ATTACHMENT_MARKER = "--- 添付ファイル ---";

export default function InquiryDetailPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { inquiryId } = useParams();

  const state = location.state as InquiryDetailLocationState | null;
  const initialContact =
    state?.contact && state.contact.id === inquiryId
      ? state.contact
      : null;

  const [contact, setContact] = useState<Contact | null>(initialContact);
  const [loading, setLoading] = useState(!initialContact);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!inquiryId) {
      setContact(null);
      setLoading(false);
      setError("問い合わせIDが指定されていません。");
      return;
    }

    let cancelled = false;

    const loadContact = async () => {
      try {
        if (!initialContact) {
          setLoading(true);
        }

        setError(null);

        const result = await getContact(inquiryId);

        if (!cancelled) {
          setContact(result);
        }
      } catch (cause) {
        if (!cancelled) {
          setError(
            cause instanceof Error
              ? cause.message
              : "問い合わせ情報の取得に失敗しました。",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void loadContact();

    return () => {
      cancelled = true;
    };
  }, [inquiryId]);

  if (loading) {
    return (
      <Page>
        <PageHeader
          title="問い合わせ詳細"
          leading={
            <button type="button" onClick={() => navigate("/inquiries")}>
              戻る
            </button>
          }
        />
        <p>問い合わせ情報を読み込んでいます。</p>
      </Page>
    );
  }

  if (!contact || error) {
    return (
      <Page>
        <PageHeader
          title="問い合わせ詳細"
          leading={
            <button type="button" onClick={() => navigate("/inquiries")}>
              戻る
            </button>
          }
        />
        <p role="alert">
          問い合わせ情報を取得できませんでした。
          {error ? ` ${error}` : ""}
        </p>
      </Page>
    );
  }

  return (
    <InquiryDetailContent
      contact={contact}
      onBack={() => navigate("/inquiries")}
    />
  );
}

function InquiryDetailContent({
  contact,
  onBack,
}: {
  contact: Contact;
  onBack: () => void;
}) {
  const parsed = useMemo(
    () => parseContactMessage(contact.message),
    [contact.message],
  );

  const [attachmentImages, setAttachmentImages] =
    useState<AttachmentImage[]>([]);
  const [attachmentsLoading, setAttachmentsLoading] = useState(false);
  const [attachmentError, setAttachmentError] =
    useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadAttachmentImages = async () => {
      const imageAttachments = parsed.attachments.filter((attachment) =>
        attachment.contentType.startsWith("image/"),
      );

      if (imageAttachments.length === 0) {
        setAttachmentImages([]);
        setAttachmentError(null);
        return;
      }

      try {
        setAttachmentsLoading(true);
        setAttachmentError(null);

        const images = await Promise.all(
          imageAttachments.map(async (attachment) => {
            const imageUrl = await getDownloadURL(
              ref(storage, attachment.storagePath),
            );

            return {
              ...attachment,
              imageUrl,
            };
          }),
        );

        if (!cancelled) {
          setAttachmentImages(images);
        }
      } catch (cause) {
        console.error(
          "[inquiry-detail] failed to load attachments",
          cause,
        );

        if (!cancelled) {
          setAttachmentImages([]);
          setAttachmentError("添付画像の取得に失敗しました。");
        }
      } finally {
        if (!cancelled) {
          setAttachmentsLoading(false);
        }
      }
    };

    void loadAttachmentImages();

    return () => {
      cancelled = true;
    };
  }, [parsed.attachments]);

  return (
    <Page>
      <PageHeader
        title="問い合わせ詳細"
        leading={
          <button type="button" onClick={onBack}>
            戻る
          </button>
        }
      />

      <DetailPageBody
        main={
          <>
            <section className="ui-detail-section">
              <h2 className="ui-detail-section__title">
                問い合わせ内容
              </h2>
              <p className="ui-detail-section__text">
                {parsed.message}
              </p>
            </section>

            {parsed.attachments.length > 0 && (
              <section className="ui-detail-section">
                <h2 className="ui-detail-section__title">
                  添付ファイル
                </h2>

                {attachmentsLoading && (
                  <p>添付画像を読み込んでいます。</p>
                )}

                {!attachmentsLoading && attachmentError && (
                  <p role="alert">{attachmentError}</p>
                )}

                {!attachmentsLoading &&
                  !attachmentError &&
                  attachmentImages.length > 0 && (
                    <div className="ui-detail-attachments">
                      {attachmentImages.map((attachment) => (
                        <figure
                          key={attachment.storagePath}
                          className="ui-detail-attachment"
                        >
                          <img
                            src={attachment.imageUrl}
                            alt={attachment.fileName}
                            className="ui-detail-attachment__image"
                          />
                          <figcaption className="ui-detail-attachment__caption">
                            {attachment.fileName}
                          </figcaption>
                        </figure>
                      ))}
                    </div>
                  )}
              </section>
            )}
          </>
        }
        aside={
          <>
            <section className="ui-detail-section">
              <h2 className="ui-detail-section__title">
                管理情報
              </h2>
              <dl className="ui-detail-definition-list">
                <dt>ステータス</dt>
                <dd>{contact.status}</dd>

                <dt>受信日時</dt>
                <dd>{formatCreatedAt(contact.createdAt)}</dd>

                <dt>送信元</dt>
                <dd>{contact.source || "-"}</dd>
              </dl>
            </section>

            <section className="ui-detail-section">
              <h2 className="ui-detail-section__title">
                送信者情報
              </h2>
              <dl className="ui-detail-definition-list">
                <dt>名前</dt>
                <dd>{contact.name}</dd>

                <dt>会社名</dt>
                <dd>{contact.company || "-"}</dd>

                <dt>メールアドレス</dt>
                <dd>
                  <a href={`mailto:${contact.email}`}>
                    {contact.email}
                  </a>
                </dd>
              </dl>
            </section>
          </>
        }
      />
    </Page>
  );
}

function parseContactMessage(value: string): ParsedContactMessage {
  const markerIndex = value.indexOf(ATTACHMENT_MARKER);

  if (markerIndex < 0) {
    return {
      message: value.trim(),
      attachments: [],
    };
  }

  const message = value.slice(0, markerIndex).trim();
  const attachmentSection = value
    .slice(markerIndex + ATTACHMENT_MARKER.length)
    .trim();

  if (!attachmentSection) {
    return {
      message,
      attachments: [],
    };
  }

  return {
    message,
    attachments: attachmentSection
      .split(/\n\s*\n/)
      .map(parseAttachmentBlock)
      .filter(
        (attachment): attachment is ContactAttachment =>
          attachment !== null,
      ),
  };
}

function parseAttachmentBlock(
  block: string,
): ContactAttachment | null {
  const lines = block
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);

  if (lines.length === 0) {
    return null;
  }

  const fileName = lines[0]
    .replace(/^\d+\.\s*/, "")
    .trim();

  const storagePath = findLineValue(
    lines,
    "Storage Path:",
  );

  if (!fileName || !storagePath) {
    return null;
  }

  const contentType =
    findLineValue(lines, "Content Type:") ||
    "application/octet-stream";

  const rawSize = findLineValue(lines, "Size:");
  const parsedSize = Number(rawSize);

  return {
    fileName,
    storagePath,
    contentType,
    size:
      rawSize !== "" &&
      Number.isFinite(parsedSize) &&
      parsedSize >= 0
        ? parsedSize
        : null,
  };
}

function findLineValue(
  lines: string[],
  prefix: string,
): string {
  const line = lines.find((candidate) =>
    candidate.startsWith(prefix),
  );

  return line
    ? line.slice(prefix.length).trim()
    : "";
}

function formatCreatedAt(value: string): string {
  if (!value) {
    return "-";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString("ja-JP");
}