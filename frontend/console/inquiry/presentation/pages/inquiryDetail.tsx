// frontend/console/inquiry/presentation/pages/inquiryDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import { safeDateTimeLabelJa } from "../../../shell/src/shared/util/dateJa";
import { useAuth } from "../../../shell/src/auth/presentation/hook/useCurrentMember";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../shell/src/shared/ui/card";
import "../style/inquiry-page.css";
import {
  getInquiryHTTP,
  reopenInquiryHTTP,
  replyInquiryHTTP,
  resolveInquiryHTTP,
  type InquiryDetail as InquiryDetailDTO,
} from "../../infrastructure/inquiryRepositoryHTTP";

const INQUIRY_READ_STATE_CHANGED_EVENT = "inquiry:read-state-changed";

type InquiryImageView = {
  id: string;
  fileName: string;
  fileUrl: string;
  mimeType: string;
};

function textOrDash(value: string | null | undefined): string {
  const trimmed = String(value ?? "").trim();
  return trimmed || "-";
}

function normalizeText(value: unknown): string {
  return String(value ?? "").trim();
}

function statusLabel(value: string | null | undefined): string {
  const status = String(value ?? "").trim();

  switch (status) {
    case "open":
      return "未対応";
    case "in_progress":
      return "対応中";
    case "resolved":
      return "対応済み";
    case "closed":
      return "クローズ";
    default:
      return status || "-";
  }
}

function isResolvedStatus(value: string | null | undefined): boolean {
  return String(value ?? "").trim() === "resolved";
}

function isUnresolvedStatus(value: string | null | undefined): boolean {
  const status = String(value ?? "").trim();

  return status === "" || status === "open" || status === "unresolved";
}

function typeLabel(value: string | null | undefined): string {
  const inquiryType = String(value ?? "").trim();

  switch (inquiryType) {
    case "product_description":
      return "商品説明";
    case "exchange":
      return "交換";
    case "shipping":
      return "配送";
    case "payment":
      return "決済";
    case "other":
      return "その他";
    case "product":
      return "商品";
    default:
      return inquiryType || "-";
  }
}

function uniqueTextValues(values: Array<string | null | undefined>): string[] {
  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const normalized = normalizeText(value);
    if (!normalized || normalized === "-") continue;
    if (seen.has(normalized)) continue;

    seen.add(normalized);
    result.push(normalized);
  }

  return result;
}

function getShippingAddressLine(address: Record<string, unknown>): string {
  const postalCode =
    normalizeText(address.zipCode) ||
    normalizeText(address.postalCode) ||
    normalizeText(address.postCode);

  const state =
    normalizeText(address.state) ||
    normalizeText(address.prefecture) ||
    normalizeText(address.region);
  const city = normalizeText(address.city);
  const street =
    normalizeText(address.street) ||
    normalizeText(address.address1) ||
    normalizeText(address.line1);

  const parts = [
    postalCode ? `〒${postalCode}` : "",
    state,
    city,
    street,
  ].filter(Boolean);

  return parts.length > 0 ? parts.join(" ") : "-";
}

function getShippingAddressStreet2(address: Record<string, unknown>): string {
  return (
    normalizeText(address.street2) ||
    normalizeText(address.address2) ||
    normalizeText(address.line2)
  );
}

function getShippingAddresses(
  detail: InquiryDetailDTO | null,
): Record<string, unknown>[] {
  if (!detail?.shippingAddresses?.length) {
    return [];
  }

  return detail.shippingAddresses.map((address) => {
    return address as unknown as Record<string, unknown>;
  });
}

function getOrderItemsLabel(
  order: NonNullable<InquiryDetailDTO["orders"]>[number],
): string {
  if (!Array.isArray(order.items) || order.items.length === 0) {
    return "-";
  }

  const labels = order.items.map((item) => {
    const tokenName = textOrDash(item.tokenName);
    const qty = Number(item.qty ?? 0);

    return qty > 0 ? `${tokenName} × ${qty}` : tokenName;
  });

  return labels.join(" / ");
}

function getOrderTransferredAtLabel(
  order: NonNullable<InquiryDetailDTO["orders"]>[number],
): string {
  if (!Array.isArray(order.items) || order.items.length === 0) {
    return "-";
  }

  const transferredAtValues = uniqueTextValues(
    order.items.map((item) => item.transferredAt ?? null),
  );

  if (transferredAtValues.length === 0) {
    return "-";
  }

  return transferredAtValues
    .map((transferredAt) => safeDateTimeLabelJa(transferredAt, "-"))
    .join(" / ");
}

function getInquiryImages(
  inquiry: InquiryDetailDTO["inquiry"] | null | undefined,
): InquiryImageView[] {
  const rawImages = (inquiry as unknown as { images?: unknown })?.images;

  if (!Array.isArray(rawImages)) {
    return [];
  }

  return rawImages
    .map((raw, index): InquiryImageView | null => {
      const image = raw as Record<string, unknown>;

      const fileUrl =
        normalizeText(image.fileUrl) ||
        normalizeText(image.FileURL) ||
        normalizeText(image.url) ||
        normalizeText(image.URL);

      if (!fileUrl) {
        return null;
      }

      const fileName =
        normalizeText(image.fileName) ||
        normalizeText(image.FileName) ||
        `問い合わせ画像${index + 1}`;

      const mimeType =
        normalizeText(image.mimeType) ||
        normalizeText(image.MimeType) ||
        "image/*";

      const id =
        normalizeText(image.id) ||
        normalizeText(image.ID) ||
        normalizeText(image.objectPath) ||
        normalizeText(image.ObjectPath) ||
        `${fileUrl}-${index}`;

      return {
        id,
        fileName,
        fileUrl,
        mimeType,
      };
    })
    .filter((image): image is InquiryImageView => image !== null);
}

function replaceDetailInquiry(
  detail: InquiryDetailDTO,
  inquiry: InquiryDetailDTO["inquiry"],
): InquiryDetailDTO {
  return {
    ...detail,
    inquiry,
  };
}

export default function InquiryDetail() {
  const navigate = useNavigate();
  const { inquiryId } = useParams<{ inquiryId: string }>();
  const { currentMember } = useAuth();

  const [detail, setDetail] = React.useState<InquiryDetailDTO | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [statusUpdating, setStatusUpdating] = React.useState(false);
  const [replyModalOpen, setReplyModalOpen] = React.useState(false);
  const [replyContent, setReplyContent] = React.useState("");
  const [replySubmitting, setReplySubmitting] = React.useState(false);
  const [replyErrorMessage, setReplyErrorMessage] = React.useState<string | null>(
    null,
  );
  const [errorMessage, setErrorMessage] = React.useState<string | null>(null);

  const memberId = String(currentMember?.id ?? "").trim();

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  React.useEffect(() => {
    let active = true;

    async function load() {
      const trimmedInquiryId = String(inquiryId ?? "").trim();

      if (!trimmedInquiryId) {
        setDetail(null);
        setErrorMessage("問い合わせIDが指定されていません。");
        setLoading(false);
        return;
      }

      setLoading(true);
      setErrorMessage(null);

      try {
        const result = await getInquiryHTTP(trimmedInquiryId);

        if (!active) return;

        setDetail(result);
        window.dispatchEvent(new Event(INQUIRY_READ_STATE_CHANGED_EVENT));
      } catch (error) {
        if (!active) return;

        const message =
          error instanceof Error
            ? error.message
            : "問い合わせ詳細の取得に失敗しました";

        setErrorMessage(message);
        setDetail(null);
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      active = false;
    };
  }, [inquiryId]);

  const inquiry = detail?.inquiry ?? null;

  const title = textOrDash(inquiry?.subject);
  const body = textOrDash(inquiry?.content);
  const avatarName = textOrDash(detail?.avatarName);
  const userFullName = textOrDash(detail?.userFullName);
  const status = statusLabel(inquiry?.status);
  const type = typeLabel(inquiry?.inquiryType);
  const productName = textOrDash(detail?.productName);
  const brandName = textOrDash(detail?.brandName);
  const inquiredAt = safeDateTimeLabelJa(inquiry?.createdAt, "-");
  const updatedAt = safeDateTimeLabelJa(inquiry?.updatedAt, "-");
  const inquiryImages = getInquiryImages(inquiry);
  const shippingAddresses = getShippingAddresses(detail);
  const orders = Array.isArray(detail?.orders) ? detail.orders : [];

  const statusButtonVariant = isUnresolvedStatus(inquiry?.status)
    ? "danger"
    : "neutral";

  const onOpenReplyModal = React.useCallback(() => {
    setReplyErrorMessage(null);
    setReplyModalOpen(true);
  }, []);

  const onCloseReplyModal = React.useCallback(() => {
    if (replySubmitting) {
      return;
    }

    setReplyModalOpen(false);
    setReplyContent("");
    setReplyErrorMessage(null);
  }, [replySubmitting]);

  const onSubmitReply = React.useCallback(async () => {
    const trimmedInquiryId = String(inquiryId ?? "").trim();
    const trimmedContent = replyContent.trim();

    if (!trimmedInquiryId) {
      setReplyErrorMessage("問い合わせIDが指定されていません。");
      return;
    }

    if (!memberId) {
      setReplyErrorMessage("メンバーIDが取得できません。ログインし直してください。");
      return;
    }

    if (!trimmedContent) {
      setReplyErrorMessage("返信内容を入力してください。");
      return;
    }

    setReplySubmitting(true);
    setReplyErrorMessage(null);
    setErrorMessage(null);

    try {
      const updatedInquiry = await replyInquiryHTTP(trimmedInquiryId, {
        memberId,
        content: trimmedContent,
        images: [],
      });

      setDetail((current) => {
        if (!current) {
          return current;
        }

        return replaceDetailInquiry(current, updatedInquiry);
      });

      setReplyModalOpen(false);
      setReplyContent("");
      setReplyErrorMessage(null);
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "問い合わせ返信の送信に失敗しました";

      setReplyErrorMessage(message);
    } finally {
      setReplySubmitting(false);
    }
  }, [inquiryId, memberId, replyContent]);

  const onToggleStatus = React.useCallback(async () => {
    const trimmedInquiryId = String(inquiryId ?? "").trim();

    if (!detail || !trimmedInquiryId) {
      return;
    }

    if (!memberId) {
      setErrorMessage("メンバーIDが取得できません。ログインし直してください。");
      return;
    }

    setStatusUpdating(true);
    setErrorMessage(null);

    try {
      const updatedInquiry = isResolvedStatus(detail.inquiry.status)
        ? await reopenInquiryHTTP(trimmedInquiryId, { memberId })
        : await resolveInquiryHTTP(trimmedInquiryId, { memberId });

      setDetail((current) => {
        if (!current) {
          return current;
        }

        return replaceDetailInquiry(current, updatedInquiry);
      });
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "問い合わせステータスの更新に失敗しました";

      setErrorMessage(message);
    } finally {
      setStatusUpdating(false);
    }
  }, [detail, inquiryId, memberId]);

  if (loading) {
    return (
      <PageStyle
        layout="grid-2"
        title="問い合わせ詳細"
        onBack={onBack}
        onSave={undefined}
      >
        <Card>
          <CardHeader>
            <CardTitle>問い合わせ内容</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="inq__empty">問い合わせ詳細を読み込み中です。</div>
          </CardContent>
        </Card>

        <div>
          <Card>
            <CardHeader>
              <CardTitle>問い合わせ情報</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="inq__empty">問い合わせ情報を読み込み中です。</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>商品・注文情報</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="inq__empty">商品・注文情報を読み込み中です。</div>
            </CardContent>
          </Card>
        </div>
      </PageStyle>
    );
  }

  if (errorMessage && !detail) {
    return (
      <PageStyle
        layout="grid-2"
        title="問い合わせ詳細"
        onBack={onBack}
        onSave={undefined}
      >
        <Card>
          <CardHeader>
            <CardTitle>問い合わせ内容</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="inq__empty">{errorMessage}</div>
          </CardContent>
        </Card>

        <div>
          <Card>
            <CardHeader>
              <CardTitle>問い合わせ情報</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="inq__empty">問い合わせ情報を表示できません。</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>商品・注文情報</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="inq__empty">商品・注文情報を表示できません。</div>
            </CardContent>
          </Card>
        </div>
      </PageStyle>
    );
  }

  return (
    <>
      <PageStyle
        layout="grid-2"
        title="問い合わせ詳細"
        onBack={onBack}
        onSave={undefined}
        onReply={onOpenReplyModal}
        statusButtonLabel={status}
        statusButtonBusyLabel="更新中"
        statusButtonVariant={statusButtonVariant}
        onStatusButtonClick={onToggleStatus}
        isStatusButtonLoading={statusUpdating}
        statusButtonDisabled={!detail || !memberId}
      >
        <Card>
          <CardHeader>
            <CardTitle>問い合わせ内容</CardTitle>
          </CardHeader>

          <CardContent>
            <div className="inq-detail">
              {errorMessage ? (
                <div className="inq__empty">{errorMessage}</div>
              ) : null}

              <h2 className="inq-detail__title">{title}</h2>

              <div className="inq-detail__meta">
                <div>
                  <span className="inq-detail__label">タイプ</span>
                  <span className="inq__chip">{type}</span>
                </div>
              </div>

              <div className="inq-detail__body">
                <div className="inq-detail__label">問い合わせ本文</div>
                <p className="inq-detail__text">{body}</p>
              </div>

              <div className="inq-detail__body">
                <div className="inq-detail__label">添付画像</div>

                {inquiryImages.length > 0 ? (
                  <div className="inq-detail__image-grid">
                    {inquiryImages.map((image) => (
                      <a
                        key={image.id}
                        href={image.fileUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="inq-detail__image-link"
                        aria-label={`${image.fileName}を開く`}
                      >
                        <img
                          src={image.fileUrl}
                          alt={image.fileName}
                          className="inq-detail__image"
                          loading="lazy"
                        />
                      </a>
                    ))}
                  </div>
                ) : (
                  <div className="inq__empty">添付画像はありません。</div>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <div>
          <Card>
            <CardHeader>
              <CardTitle>問い合わせ情報</CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq-detail">
                <div className="inq-detail__meta">
                  <div>
                    <span className="inq-detail__label">アバター名</span>
                    <span className="inq-detail__value">{avatarName}</span>
                  </div>

                  <div>
                    <span className="inq-detail__label">ユーザー名</span>
                    <span className="inq-detail__value">{userFullName}</span>
                  </div>

                  <div>
                    <span className="inq-detail__label">配送先情報</span>

                    {shippingAddresses.length > 0 ? (
                      <div className="inq-detail__value">
                        {shippingAddresses.map((address, index) => {
                          const addressLine = getShippingAddressLine(address);
                          const street2 = getShippingAddressStreet2(address);

                          return (
                            <div key={`${normalizeText(address.id) || index}`}>
                              <div>{addressLine}</div>
                              {street2 ? <div>{street2}</div> : null}
                            </div>
                          );
                        })}
                      </div>
                    ) : (
                      <span className="inq-detail__value">-</span>
                    )}
                  </div>

                  <div>
                    <span className="inq-detail__label">問い合わせ日</span>
                    <span className="inq-detail__value">{inquiredAt}</span>
                  </div>

                  <div>
                    <span className="inq-detail__label">最終更新日</span>
                    <span className="inq-detail__value">{updatedAt}</span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>商品・注文情報</CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq-detail">
                <div className="inq-detail__meta">
                  <div>
                    <span className="inq-detail__label">商品名</span>
                    <span className="inq-detail__value">{productName}</span>
                  </div>

                  <div>
                    <span className="inq-detail__label">ブランド</span>
                    <span className="inq-detail__value">{brandName}</span>
                  </div>

                  {orders.length > 0 ? (
                    orders.map((order) => (
                      <div key={order.id}>
                        <span className="inq-detail__label">注文ID</span>
                        <span className="inq-detail__value">
                          {textOrDash(order.id)}
                        </span>

                        <span className="inq-detail__label">発注日時</span>
                        <span className="inq-detail__value">
                          {safeDateTimeLabelJa(order.createdAt, "-")}
                        </span>

                        <span className="inq-detail__label">移譲日</span>
                        <span className="inq-detail__value">
                          {getOrderTransferredAtLabel(order)}
                        </span>

                        <span className="inq-detail__label">注文内容</span>
                        <span className="inq-detail__value">
                          {getOrderItemsLabel(order)}
                        </span>
                      </div>
                    ))
                  ) : (
                    <div className="inq__empty">注文情報はありません。</div>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </PageStyle>

      {replyModalOpen ? (
        <div
          className="inq-reply-modal"
          role="presentation"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) {
              onCloseReplyModal();
            }
          }}
        >
          <div
            className="inq-reply-modal__panel"
            role="dialog"
            aria-modal="true"
            aria-labelledby="inquiry-reply-modal-title"
          >
            <div className="inq-reply-modal__header">
              <div>
                <h2
                  id="inquiry-reply-modal-title"
                  className="inq-reply-modal__title"
                >
                  返信を入力
                </h2>
                <p className="inq-reply-modal__description">
                  この問い合わせに対する返信内容を入力してください。
                </p>
              </div>

              <button
                type="button"
                className="inq-reply-modal__close"
                onClick={onCloseReplyModal}
                disabled={replySubmitting}
                aria-label="返信モーダルを閉じる"
              >
                ×
              </button>
            </div>

            <div className="inq-reply-modal__body">
              {replyErrorMessage ? (
                <div className="inq__empty">{replyErrorMessage}</div>
              ) : null}

              <label
                className="inq-reply-modal__label"
                htmlFor="inquiry-reply-content"
              >
                返信内容
              </label>

              <textarea
                id="inquiry-reply-content"
                className="inq-reply-modal__textarea"
                value={replyContent}
                placeholder="返信内容を入力してください"
                rows={8}
                maxLength={2000}
                disabled={replySubmitting}
                onChange={(event) => setReplyContent(event.target.value)}
              />

              <div className="inq-reply-modal__counter">
                {replyContent.length.toLocaleString()} / 2,000
              </div>
            </div>

            <div className="inq-reply-modal__actions">
              <button
                type="button"
                className="inq-reply-modal__button inq-reply-modal__button--ghost"
                onClick={onCloseReplyModal}
                disabled={replySubmitting}
              >
                キャンセル
              </button>

              <button
                type="button"
                className="inq-reply-modal__button"
                disabled={replySubmitting || !replyContent.trim()}
                onClick={() => void onSubmitReply()}
              >
                {replySubmitting ? "送信中" : "送信"}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}