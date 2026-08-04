// frontend/amol/src/pages/MarketDetailPage.tsx

import {
  useNavigate,
  useParams,
} from "react-router-dom";

import Layout from "../components/layout/Layout";

import MarketDetailContent from "../features/market/presentation/components/MarketDetailContent";
import {
  useMarketDetailPage,
} from "../features/market/presentation/hooks/useMarketDetailPage";

import {
  getApiBaseUrl,
} from "../lib/apiBaseUrl";
import {
  auth,
} from "../lib/firebase";

import "../styles/page-layout.css";
import "../styles/market-detail-page.css";

async function readResponseErrorMessage(
  response: Response,
): Promise<string> {
  const contentType =
    response.headers.get(
      "content-type",
    ) ?? "";

  if (
    contentType.includes(
      "application/json",
    )
  ) {
    const data = (
      await response
        .json()
        .catch(() => null)
    ) as
      | {
          error?: unknown;
        }
      | null;

    if (
      typeof data?.error ===
        "string" &&
      data.error.trim() !== ""
    ) {
      return data.error;
    }
  }

  const text =
    await response
      .text()
      .catch(() => "");

  if (
    text.trim() !== ""
  ) {
    return text;
  }

  return "リクエストに失敗しました。";
}

async function addResaleProductToCart(
  args: {
    resaleId: string;
    productId: string;
  },
): Promise<void> {
  const currentUser =
    auth.currentUser;

  if (!currentUser) {
    throw new Error(
      "カートに追加するにはログインが必要です。",
    );
  }

  const apiBaseUrl =
    getApiBaseUrl();

  if (!apiBaseUrl) {
    throw new Error(
      "APIの接続先が設定されていません。",
    );
  }

  const normalizedApiBaseUrl =
    apiBaseUrl.replace(
      /\/+$/,
      "",
    );

  const idToken =
    await currentUser
      .getIdToken();

  const response =
    await fetch(
      `${normalizedApiBaseUrl}/mall/me/cart/resales`,
      {
        method: "POST",
        headers: {
          Accept:
            "application/json",
          "Content-Type":
            "application/json",
          Authorization:
            `Bearer ${idToken}`,
        },
        credentials:
          "include",
        body: JSON.stringify({
          resaleId:
            args.resaleId,
          productId:
            args.productId,
        }),
      },
    );

  if (!response.ok) {
    const message =
      await readResponseErrorMessage(
        response,
      );

    throw new Error(
      message ||
        "カートへの追加に失敗しました。",
    );
  }
}

export default function MarketDetailPage() {
  const navigate =
    useNavigate();

  const {
    resaleId,
  } = useParams<{
    resaleId: string;
  }>();

  const detail =
    useMarketDetailPage({
      resaleId,
      addResaleProductToCart,
    });

  const {
    title,
    addingToCart,
    canAddToCart,
    sellerAvatarId,
    handleAddToCart,
  } = detail;

  function handleOpenSellerAvatar() {
    if (!sellerAvatarId) {
      return;
    }

    navigate(
      `/avatars/${encodeURIComponent(
        sellerAvatarId,
      )}`,
    );
  }

  return (
    <Layout
      title={title}
      titleClickable={false}
      showBackButton
      onBackButtonClick={() =>
        navigate(-1)
      }
      hideAnnouncementButton
      hideSettingsButton
      hideHamburgerMenu
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={() =>
        navigate("/cart")
      }
      actionButtonLabel={
        addingToCart
          ? "追加中"
          : "カートに入れる"
      }
      onActionButtonClick={
        handleAddToCart
      }
      actionButtonDisabled={
        !canAddToCart
      }
      showFooter
      footerProps={{
        variant:
          "action",
        buttonLabel:
          addingToCart
            ? "追加中"
            : "カートに入れる",
        disabled:
          !canAddToCart,
        onButtonClick:
          handleAddToCart,
      }}
    >
      <MarketDetailContent
        detail={detail}
        onOpenSeller={
          handleOpenSellerAvatar
        }
      />
    </Layout>
  );
}