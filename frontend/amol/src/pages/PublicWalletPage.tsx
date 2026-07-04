// frontend/amol/src/pages/PublicWalletPage.tsx
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/wallet-page.css";
import "../styles/wallet-page/resale-panel.css";

import Layout from "../components/layout/Layout";
import WalletProfile from "../features/wallet/components/WalletProfile";
import WalletProfileActions from "../features/wallet/components/WalletProfileActions";
import WalletResalePanel from "../features/wallet/components/WalletResalePanel";
import { useWalletPage } from "../features/wallet/hooks/useWalletPage";

export default function PublicWalletPage() {
  const navigate = useNavigate();

  const {
    avatarId,
    viewedAvatarId,
    avatarName,
    avatarIcon,
    profile,
    loading,
    error,
    pageTitle,
  } = useWalletPage();

  const targetAvatarId = viewedAvatarId || avatarId;

  const handleOpenMarketDetail = (resaleId: string) => {
    const id = resaleId.trim();

    if (!id) {
      return;
    }

    navigate(`/market/${encodeURIComponent(id)}`);
  };

  return (
    <Layout title={pageTitle || "AMOL"} mode="mypage">
      <section className="content-page-section wallet-page">
        <div className="wallet-page-layout">
          <aside className="wallet-page-layout__profile">
            {loading ? (
              <p className="wallet-page__message">読み込み中です...</p>
            ) : null}

            {!loading && error ? (
              <div role="alert" className="wallet-page__message">
                <p>{error}</p>
              </div>
            ) : null}

            {!loading && !error ? (
              <>
                <WalletProfile
                  avatarName={avatarName}
                  avatarIcon={avatarIcon}
                  profile={profile}
                  isOwnAvatar={false}
                />

                <WalletProfileActions
                  avatarId={targetAvatarId}
                  isOwnAvatar={false}
                />
              </>
            ) : null}
          </aside>

          <div className="wallet-page-layout__main">
            {!loading && !error ? (
              <WalletResalePanel
                avatarId={targetAvatarId}
                onItemClick={handleOpenMarketDetail}
              />
            ) : null}
          </div>
        </div>
      </section>
    </Layout>
  );
}