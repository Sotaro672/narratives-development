// frontend/amol/src/pages/WalletPage.tsx
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/wallet-page.css";
import "../styles/wallet-page/resale-panel.css";

import Layout from "../components/layout/Layout";
import WalletHistoryPanel from "../features/wallet/components/WalletHistoryPanel";
import WalletProfile from "../features/wallet/components/WalletProfile";
import WalletProfileActions from "../features/wallet/components/WalletProfileActions";
import WalletResalePanel from "../features/wallet/components/WalletResalePanel";
import WalletTabs from "../features/wallet/components/WalletTabs";
import WalletTokenContentsCard from "../features/wallet/components/WalletTokenContentsCard";
import WalletTokenEmpty from "../features/wallet/components/WalletTokenEmpty";
import { useWalletPage } from "../features/wallet/hooks/useWalletPage";
import type { WalletTokenItem } from "../features/wallet/types/tokenTypes";

export default function WalletPage() {
  const navigate = useNavigate();

  const {
    avatarId,
    viewedAvatarId,
    isOwnAvatar,
    avatarName,
    avatarIcon,
    profile,
    walletTokens,
    orderHistory,
    activeTab,
    setActiveTab,
    loading,
    error,
    tokenLoading,
    tokenError,
    orderLoading,
    orderError,
    hasItems,
    hasTokens,
    pageTitle,
  } = useWalletPage();

  const handleOpenContents = (token: WalletTokenItem) => {
    const params = new URLSearchParams();

    params.set("mintAddress", token.mintAddress);

    if (token.productId) params.set("productId", token.productId);
    if (token.brandId) params.set("brandId", token.brandId);
    if (token.brandName) params.set("brandName", token.brandName);
    if (token.productName) params.set("productName", token.productName);
    if (token.productBlueprintId) {
      params.set("productBlueprintId", token.productBlueprintId);
    }
    if (token.tokenBlueprintId) {
      params.set("tokenBlueprintId", token.tokenBlueprintId);
    }
    if (token.metadataUri) params.set("metadataUri", token.metadataUri);

    const tokenName = token.metadata?.name || "";
    const tokenIconUrl = token.metadata?.image || "";

    if (tokenName) params.set("tokenName", tokenName);
    if (tokenIconUrl) params.set("tokenIconUrl", tokenIconUrl);

    navigate(`/contents?${params.toString()}`);
  };

  const handleOpenBrand = (brandId: string) => {
    const id = brandId.trim();

    if (!id) {
      return;
    }

    navigate(`/brands/${encodeURIComponent(id)}`);
  };

  const renderTokenList = () => (
    <div className="wallet-page-token-list">
      {tokenLoading ? (
        <p className="wallet-page__message">読み込み中です...</p>
      ) : null}

      {!tokenLoading && tokenError ? (
        <div role="alert" className="wallet-page__message">
          <p>{tokenError}</p>
        </div>
      ) : null}

      {!tokenLoading && !tokenError && !hasTokens ? <WalletTokenEmpty /> : null}

      {!tokenLoading && !tokenError && hasTokens
        ? walletTokens.map((token) => (
            <div key={token.mintAddress} className="wallet-page-token-list__item">
              <WalletTokenContentsCard
                tokenIconUrl={token.metadata?.image || null}
                tokenName={token.metadata?.name || ""}
                productName={token.productName}
                onClick={() => handleOpenContents(token)}
              />
            </div>
          ))
        : null}
    </div>
  );

  return (
    <Layout title={pageTitle || "AMOL"} mode="mypage">
      <section className="content-page-section wallet-page">
        <div className="wallet-page-layout">
          <aside className="wallet-page-layout__profile">
            <WalletProfile
              avatarName={avatarName}
              avatarIcon={avatarIcon}
              profile={profile}
              isOwnAvatar={isOwnAvatar}
            />

            <WalletProfileActions avatarId={avatarId} isOwnAvatar={isOwnAvatar} />
          </aside>

          <div className="wallet-page-layout__main">
            {isOwnAvatar ? (
              <>
                <WalletTabs activeTab={activeTab} onChange={setActiveTab} />

                {activeTab === "history" ? (
                  <WalletHistoryPanel
                    loading={loading || orderLoading}
                    error={error || orderError}
                    hasItems={hasItems}
                    orderHistory={orderHistory}
                    onBrandClick={handleOpenBrand}
                  />
                ) : null}

                {activeTab === "tokens" ? renderTokenList() : null}

                {activeTab === "resales" ? <WalletResalePanel /> : null}
              </>
            ) : (
              <WalletResalePanel avatarId={viewedAvatarId} />
            )}
          </div>
        </div>
      </section>
    </Layout>
  );
}