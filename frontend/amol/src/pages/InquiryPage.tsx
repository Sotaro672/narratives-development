// frontend/amol/src/pages/InquiryPage.tsx
import { useCallback, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import { getApiBaseUrl } from "../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../lib/authToken";
import "../styles/inquiry-page.css";

type CreateInquiryRequest = {
  productId: string;
  subject: string;
  content: string;
  inquiryType: string;
  images: [];
};

type CreateInquiryResponse = {
  data?: {
    id?: string;
    productId?: string;
    avatarId?: string;
    subject?: string;
    content?: string;
    status?: string;
    inquiryType?: string;
    createdAt?: string;
    updatedAt?: string;
  };
  error?: string;
};

const DEFAULT_INQUIRY_TYPE = "product";

function buildApiUrl(path: string): string {
  const baseUrl = getApiBaseUrl();

  if (!baseUrl) {
    return path;
  }

  return `${baseUrl}${path}`;
}

export default function InquiryPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const productId = useMemo(() => {
    return (searchParams.get("productId") ?? "").trim();
  }, [searchParams]);

  const backTo = useMemo(() => {
    if (!productId) {
      return "/scan/result";
    }

    return `/scan/result/${encodeURIComponent(productId)}`;
  }, [productId]);

  const [subject, setSubject] = useState("");
  const [content, setContent] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canSubmit =
    Boolean(productId) &&
    Boolean(subject.trim()) &&
    Boolean(content.trim()) &&
    !submitting &&
    !submitted;

  const submitInquiry = useCallback(async () => {
    if (!canSubmit) {
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const token = await getFirebaseIdToken();

      const payload: CreateInquiryRequest = {
        productId,
        subject: subject.trim(),
        content: content.trim(),
        inquiryType: DEFAULT_INQUIRY_TYPE,
        images: [],
      };

      const url = buildApiUrl("/mall/me/inquiries");

      const res = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(payload),
      });

      const json = (await res.json().catch(() => ({}))) as CreateInquiryResponse;

      if (!res.ok) {
        throw new Error(json.error || "問い合わせの送信に失敗しました。");
      }

      setSubmitted(true);
      setSubject("");
      setContent("");
    } catch (e) {
      const message =
        e instanceof Error ? e.message : "問い合わせの送信に失敗しました。";
      setError(message);
    } finally {
      setSubmitting(false);
    }
  }, [canSubmit, content, productId, subject]);

  const handleBackToScanResult = useCallback(() => {
    navigate(backTo);
  }, [backTo, navigate]);

  return (
    <Layout
      title="問い合わせ"
      mode="mypage"
      showHeader
      showBackButton
      showFooter
      backTo={backTo}
      hideHamburgerMenu={false}
      hideSettingsButton={false}
      mainClassName="inquiry-page"
      footerProps={{
        variant: "action",
        buttonLabel: submitting ? "送信中" : submitted ? "送信済み" : "送信する",
        disabled: !canSubmit,
        onButtonClick: submitInquiry,
      }}
    >
      <section className="inquiry-page__container">
        <div className="inquiry-page__header">
          <p className="inquiry-page__eyebrow">CONTACT</p>
          <h1 className="inquiry-page__title">商品について問い合わせる</h1>
          <p className="inquiry-page__description">
            商品について確認したいことや、購入前に相談したい内容を入力してください。
          </p>
        </div>

        {!productId ? (
          <div className="inquiry-page__notice inquiry-page__notice--error">
            <p>商品IDが見つかりませんでした。</p>
            <button
              type="button"
              className="inquiry-page__secondary-button"
              onClick={() => navigate("/scan/result")}
            >
              スキャン結果へ戻る
            </button>
          </div>
        ) : null}

        {submitted ? (
          <div className="inquiry-page__notice inquiry-page__notice--success">
            <p>問い合わせを送信しました。</p>
            <p>返信があるまでしばらくお待ちください。</p>
            <button
              type="button"
              className="inquiry-page__secondary-button"
              onClick={handleBackToScanResult}
            >
              スキャン結果へ戻る
            </button>
          </div>
        ) : null}

        {error ? (
          <div className="inquiry-page__notice inquiry-page__notice--error">
            {error}
          </div>
        ) : null}

        {!submitted ? (
          <form
            className="inquiry-page__form"
            onSubmit={(event) => {
              event.preventDefault();
              void submitInquiry();
            }}
          >
            <input type="hidden" name="productId" value={productId} />

            <div className="inquiry-page__field">
              <label className="inquiry-page__label" htmlFor="inquiry-subject">
                件名
              </label>
              <input
                id="inquiry-subject"
                className="inquiry-page__input"
                type="text"
                value={subject}
                placeholder="例: 商品の状態について"
                maxLength={120}
                disabled={!productId || submitting}
                onChange={(event) => setSubject(event.target.value)}
              />
            </div>

            <div className="inquiry-page__field">
              <label className="inquiry-page__label" htmlFor="inquiry-content">
                問い合わせ内容
              </label>
              <textarea
                id="inquiry-content"
                className="inquiry-page__textarea"
                value={content}
                placeholder="問い合わせ内容を入力してください"
                rows={8}
                maxLength={2000}
                disabled={!productId || submitting}
                onChange={(event) => setContent(event.target.value)}
              />
              <div className="inquiry-page__counter">
                {content.length.toLocaleString()} / 2,000
              </div>
            </div>

            <div className="inquiry-page__meta">
              <span>商品ID</span>
              <code>{productId || "-"}</code>
            </div>

            <button
              type="submit"
              className="inquiry-page__submit-button"
              disabled={!canSubmit}
            >
              {submitting ? "送信中" : "送信する"}
            </button>
          </form>
        ) : null}
      </section>
    </Layout>
  );
}