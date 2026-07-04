// frontend/amol/src/pages/MarketDetailPage.tsx
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  fetchMarketResaleById,
  type MarketResaleListing,
} from "../features/market/marketApi";
import { getApiBaseUrl } from "../lib/apiBaseUrl";
import { auth } from "../lib/firebase";

import "../styles/page-layout.css";
import "../styles/market-detail-page.css";

async function readResponseErrorMessage(response: Response): Promise<string> {
  const contentType = response.headers.get("content-type") ?? "";

  if (contentType.includes("application/json")) {
    const data = (await response.json().catch(() => null)) as
      | { error?: unknown; message?: unknown }
      | null;

    if (typeof data?.error === "string" && data.error.trim() !== "") {
      return data.error;
    }

    if (typeof data?.message === "string" && data.message.trim() !== "") {
      return data.message;
    }
  }

  const text = await response.text().catch(() => "");

  if (text.trim() !== "") {
    return text;
  }

  return "リクエストに失敗しました。";
}

async function addResaleProductToCart(args: {
  resaleId: string;
  productId: string;
}): Promise<void> {
  const currentUser = auth.currentUser;

  if (!currentUser) {
    throw new Error("カートに追加するにはログインが必要です。");
  }

  const apiBaseUrl = getApiBaseUrl();

  if (!apiBaseUrl) {
    throw new Error("APIの接続先が設定されていません。");
  }

  const idToken = await currentUser.getIdToken();

  const response = await fetch(`${apiBaseUrl}/mall/me/cart/resales`, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
    body: JSON.stringify({
      resaleId: args.resaleId,
      productId: args.productId,
    }),
  });

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "カートへの追加に失敗しました。");
  }
}

export default function MarketDetailPage() {
  const navigate = useNavigate();
  const { resaleId } = useParams<{ resaleId: string }>();

  const [item, setItem] = useState<MarketResaleListing | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [addingToCart, setAddingToCart] = useState<boolean>(false);
  const [error, setError] = useState<string>("");
  const [cartMessage, setCartMessage] = useState<string>("");
  const [cartErrorMessage, setCartErrorMessage] = useState<string>("");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (!resaleId) {
        setError("出品情報が見つかりません。");
        setLoading(false);
        return;
      }

      setLoading(true);
      setError("");
      setCartMessage("");
      setCartErrorMessage("");

      try {
        const data = await fetchMarketResaleById(resaleId);

        if (!cancelled) {
          setItem(data);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : "出品情報の取得に失敗しました。",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [resaleId]);

  const title = item?.productName || item?.tokenName || "マーケット詳細";

  const priceLabel =
    typeof item?.price === "number"
      ? `${item.price.toLocaleString("ja-JP")}円`
      : "価格未設定";

  const canAddToCart = Boolean(
    item?.id && item?.productId && !loading && !error && !addingToCart,
  );

  async function handleAddToCart() {
    const targetResaleId = item?.id?.trim();
    const targetProductId = item?.productId?.trim();

    if (!targetResaleId || !targetProductId) {
      setCartMessage("");
      setCartErrorMessage("出品情報が不足しています。");
      return;
    }

    setAddingToCart(true);
    setCartMessage("");
    setCartErrorMessage("");

    try {
      await addResaleProductToCart({
        resaleId: targetResaleId,
        productId: targetProductId,
      });

      setCartMessage("カートに追加しました。");
    } catch (err) {
      setCartErrorMessage(
        err instanceof Error ? err.message : "カートへの追加に失敗しました。",
      );
    } finally {
      setAddingToCart(false);
    }
  }

  return (
    <Layout
      title={title}
      titleClickable={false}
      showBackButton
      onBackButtonClick={() => navigate(-1)}
      hideAnnouncementButton
      hideSettingsButton
      hideHamburgerMenu
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={() => navigate("/cart")}
      actionButtonLabel={addingToCart ? "追加中" : "カートに入れる"}
      onActionButtonClick={handleAddToCart}
      actionButtonDisabled={!canAddToCart}
      showFooter
      footerProps={{
        variant: "action",
        buttonLabel: addingToCart ? "追加中" : "カートに入れる",
        disabled: !canAddToCart,
        onButtonClick: handleAddToCart,
      }}
    >
      <div className="page-layout market-detail-page">
        {loading ? (
          <div className="market-detail-page__state">
            <p>読み込み中です...</p>
          </div>
        ) : null}

        {!loading && error ? (
          <div className="market-detail-page__state market-detail-page__state--error">
            <p>{error}</p>
          </div>
        ) : null}

        {!loading && !error && item ? (
          <section className="market-detail-page__card">
            <div className="market-detail-page__image-wrap">
              {item.imageUrl ? (
                <img
                  src={item.imageUrl}
                  alt={item.productName || item.tokenName || "出品画像"}
                  className="market-detail-page__image"
                />
              ) : (
                <div className="market-detail-page__image-placeholder">
                  No Image
                </div>
              )}
            </div>

            <div className="market-detail-page__content">
              <p className="market-detail-page__brand">
                {item.brandName || "ブランド名未設定"}
              </p>

              <h1 className="market-detail-page__title">
                {item.productName || item.tokenName || "商品名未設定"}
              </h1>

              <p className="market-detail-page__price">{priceLabel}</p>

              {item.condition ? (
                <dl className="market-detail-page__meta">
                  <div className="market-detail-page__meta-row">
                    <dt>状態</dt>
                    <dd>{item.condition}</dd>
                  </div>
                </dl>
              ) : null}

              {item.description ? (
                <div className="market-detail-page__description">
                  <h2>商品説明</h2>
                  <p>{item.description}</p>
                </div>
              ) : null}

              {cartMessage ? (
                <p className="market-detail-page__cart-message">
                  {cartMessage}
                </p>
              ) : null}

              {cartErrorMessage ? (
                <p className="market-detail-page__cart-error" role="alert">
                  {cartErrorMessage}
                </p>
              ) : null}
            </div>
          </section>
        ) : null}
      </div>
    </Layout>
  );
}