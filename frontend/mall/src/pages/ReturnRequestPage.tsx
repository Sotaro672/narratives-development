// frontend/mall/src/pages/ReturnRequestPage.tsx

import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import Checkbox from "../components/ui/Checkbox";
import Chip from "../components/ui/Chip";
import MediaIcon from "../components/ui/MediaIcon";
import Textbox from "../components/ui/Textbox";

import { returnOrderItem, type ReturnPackageState } from "../features/order/api/orderDetailApi";
import { useOrderDetail } from "../features/order/hooks/useOrderDetail";
import { getFallbackInitial, getProductTitle } from "../features/order/util/orderItemDisplay";
import { getApiBaseUrl } from "../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../lib/authToken";

import "../styles/page-layout.css";
import "../styles/return-request-page.css";

function parseItemIndex(value: string | undefined): number | null {
  const normalized = value?.trim() ?? "";

  if (!/^\d+$/.test(normalized)) {
    return null;
  }

  const parsed = Number(normalized);

  return Number.isSafeInteger(parsed) && parsed >= 0
    ? parsed
    : null;
}

function getErrorMessage(caught: unknown, defaultMessage: string): string {
  return caught instanceof Error
    ? caught.message
    : defaultMessage;
}

export default function ReturnRequestPage() {
  const navigate = useNavigate();
  const { itemIndex: routeItemIndex } = useParams<{ itemIndex: string }>();
  const itemIndex = useMemo(() => parseItemIndex(routeItemIndex), [routeItemIndex]);

  const {
    orderId,
    order,
    loading,
    error: orderError,
    reload,
  } = useOrderDetail();

  const [packageState, setPackageState] = useState<ReturnPackageState | null>(null);
  const [reason, setReason] = useState("");
  const [agreedToReturnConditions, setAgreedToReturnConditions] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");

  const item =
    order && itemIndex !== null
      ? order.items[itemIndex] ?? null
      : null;

  const eligibilityError = useMemo(() => {
    if (!order || itemIndex === null || !item) {
      return null;
    }

    if (item.itemType === "resale") {
      return "フリマ取引の商品はこの返品申請画面から返品できません。取引画面から返品相談を行ってください。";
    }

    if (item.isCancelled) {
      return "キャンセル済みの商品は返品できません。";
    }

    if (!item.isDispatched) {
      return "未発送の商品は返品できません。";
    }

    if (item.transferred) {
      return "受け取り済みの商品は返品できません。";
    }

    if (item.isReturnCompleted) {
      return "この商品の返品は完了しています。";
    }

    if (item.isReturnRequested) {
      return "この商品は返品申請済みです。";
    }

    return null;
  }, [item, itemIndex, order]);

  const unopenedUnavailable =
    packageState === "unopened" &&
    Boolean(item?.tokenTransferVerifiedAt);

  const normalizedReason = reason.trim();

  const canSubmit =
    Boolean(item) &&
    !eligibilityError &&
    packageState !== null &&
    normalizedReason.length > 0 &&
    !unopenedUnavailable &&
    !submitting &&
    (
      packageState === "opened" ||
      agreedToReturnConditions
    );

  const handlePackageStateChange = (nextState: ReturnPackageState) => {
    if (submitting) {
      return;
    }

    setPackageState(nextState);
    setReason("");
    setSubmitError("");
    setAgreedToReturnConditions(false);
  };

  const handleBack = () => {
    if (submitting) {
      return;
    }

    if (orderId) {
      navigate(`/orders/${encodeURIComponent(orderId)}`);
      return;
    }

    navigate(-1);
  };

  const handleSubmit = async () => {
    if (
      !canSubmit ||
      !orderId ||
      itemIndex === null ||
      packageState === null
    ) {
      return;
    }

    setSubmitting(true);
    setSubmitError("");

    try {
      const backendUrl = getApiBaseUrl();

      if (!backendUrl) {
        throw new Error("VITE_API_BASE_URLが設定されていません。");
      }

      const idToken = await getFirebaseIdToken();

      await returnOrderItem({
        backendUrl,
        idToken,
        orderId,
        itemIndex,
        packageState,
        reason: normalizedReason,
      });

      navigate(
        `/orders/${encodeURIComponent(orderId)}`,
        { replace: true },
      );
    } catch (caught) {
      const message = getErrorMessage(
        caught,
        "商品の返品受付に失敗しました。",
      );

      try {
        await reload();
      } catch {
        // reload側で取得エラーが管理されるため、ここでは送信エラーを優先して表示する。
      }

      setSubmitError(message);
    } finally {
      setSubmitting(false);
    }
  };

  const visibleError = submitError || orderError;

  return (
    <Layout title="AMOL" mode="mypage" showFooter>
      <section className="page-section return-request-page">
        <header className="return-request-page__header">
          <h1 className="return-request-page__title">返品を申請する</h1>
          <p className="return-request-page__description">
            返品する商品を確認し、商品の開封状態と返品理由を入力してください。
          </p>
        </header>

        {loading ? (
          <div className="return-request-page__state">
            <p>注文情報を読み込んでいます...</p>
          </div>
        ) : null}

        {!loading && (!order || itemIndex === null || !item) ? (
          <div className="return-request-page__state">
            <Alert variant="error">
              {orderError || "返品対象の商品が見つかりません。"}
            </Alert>

            {orderId ? (
              <Button
                type="button"
                variant="secondary"
                onClick={() => void reload()}
              >
                再読み込み
              </Button>
            ) : null}
          </div>
        ) : null}

        {!loading && order && itemIndex !== null && item ? (
          <form
            className="return-request-page__form"
            onSubmit={(event) => {
              event.preventDefault();
              void handleSubmit();
            }}
          >
            <section className="return-request-page__section">
              <h2 className="return-request-page__section-title">返品する商品</h2>

              <div className="return-request-page__product">
                <MediaIcon
                  src={item.tokenIcon}
                  alt={item.tokenName || getProductTitle(item)}
                  fallback={getFallbackInitial(item.tokenName || getProductTitle(item))}
                  size="lg"
                  shape="rounded"
                />

                <div className="return-request-page__product-body">
                  <strong className="return-request-page__product-name">
                    {getProductTitle(item)}
                  </strong>

                  {item.tokenName ? (
                    <span className="return-request-page__token-name">
                      {item.tokenName}
                    </span>
                  ) : null}

                  {item.brandName ? (
                    <span className="return-request-page__brand-name">
                      {item.brandName}
                    </span>
                  ) : null}
                </div>
              </div>
            </section>

            {eligibilityError ? (
              <Alert variant="error">
                {eligibilityError}
              </Alert>
            ) : null}

            {!eligibilityError ? (
              <>
                <section
                  className="return-request-page__section"
                  aria-labelledby="return-request-package-state"
                >
                  <h2
                    id="return-request-package-state"
                    className="return-request-page__section-title"
                  >
                    商品の開封状態
                  </h2>

                  <p className="return-request-page__section-description">
                    商品の包装紙は開封されていますか？
                  </p>

                  <div
                    className="return-request-page__package-options"
                    role="group"
                    aria-labelledby="return-request-package-state"
                  >
                    <Chip
                      selected={packageState === "unopened"}
                      disabled={submitting}
                      onClick={() => handlePackageStateChange("unopened")}
                    >
                      開封前
                    </Chip>

                    <Chip
                      selected={packageState === "opened"}
                      disabled={submitting}
                      onClick={() => handlePackageStateChange("opened")}
                    >
                      開封済
                    </Chip>
                  </div>
                </section>

                {unopenedUnavailable ? (
                  <Alert variant="error">
                    この商品は開封確認済みのため、開封前として返品を申請できません。
                  </Alert>
                ) : null}

                {packageState === "unopened" && !unopenedUnavailable ? (
                  <section className="return-request-page__section">
                    <h2 className="return-request-page__section-title">返品条件</h2>

                    <Alert variant="warning">
                      <ol className="return-request-page__condition-list">
                        <li>返品が承認された場合、返金対象は商品代金（税込）のみです。</li>
                        <li>商品代金（税込）には、商品本体価格とその商品にかかる消費税が含まれます。</li>
                        <li>ご購入時の配送料および配送料にかかる消費税は返金対象外です。</li>
                        <li>返品商品の返送にかかる配送料はお客様のご負担となります。</li>
                        <li>返品手続き中は、商品が入っている配送用梱包材を開けないでください。</li>
                      </ol>
                    </Alert>

                    <Checkbox
                      id="return-request-agreement"
                      label="返品条件に合意する"
                      checked={agreedToReturnConditions}
                      disabled={submitting}
                      onChange={(event) => {
                        setAgreedToReturnConditions(event.target.checked);
                      }}
                    />
                  </section>
                ) : null}

                {packageState !== null ? (
                  <section className="return-request-page__section">
                    <h2 className="return-request-page__section-title">返品理由</h2>

                    <Textbox
                      id="return-request-reason"
                      value={reason}
                      rows={6}
                      required
                      disabled={submitting}
                      placeholder="返品理由を入力してください"
                      onChange={(event) => {
                        setReason(event.target.value);
                        setSubmitError("");
                      }}
                    />
                  </section>
                ) : null}

                {visibleError ? (
                  <Alert variant="error">
                    {visibleError}
                  </Alert>
                ) : null}
              </>
            ) : null}

            <div className="return-request-page__actions">
              <Button
                type="button"
                variant="secondary"
                size="lg"
                disabled={submitting}
                onClick={handleBack}
              >
                注文詳細へ戻る
              </Button>

              {!eligibilityError ? (
                <Button
                  type="submit"
                  variant="primary"
                  size="lg"
                  disabled={!canSubmit}
                >
                  {submitting ? "申請中..." : "返品を申請する"}
                </Button>
              ) : null}
            </div>
          </form>
        ) : null}
      </section>
    </Layout>
  );
}