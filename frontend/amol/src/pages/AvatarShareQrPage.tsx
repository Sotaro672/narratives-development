// frontend/amol/src/pages/AvatarShareQrPage.tsx
import { useMemo } from "react";
import { useParams } from "react-router-dom";
import { QRCodeCanvas } from "qrcode.react";

import "../styles/page-layout.css";
import "../styles/wallet-page.css";

import Layout from "../components/layout/Layout";

export default function AvatarShareQrPage() {
  const { avatarId } = useParams<{ avatarId: string }>();

  const profileUrl = useMemo(() => {
    if (!avatarId) {
      return "";
    }

    if (typeof window === "undefined") {
      return `/avatars/${avatarId}`;
    }

    return `${window.location.origin}/avatars/${avatarId}`;
  }, [avatarId]);

  return (
    <Layout
      title="プロフィールをシェア"
      mode="mypage"
      showBackButton
      backTo="/wallet"
    >
      <section className="content-page-section wallet-page">
        <div className="wallet-page-layout">
          <div className="wallet-page-layout__main">
            <div className="wallet-page-share-qr">
              {profileUrl ? (
                <div className="wallet-page-share-qr__qr-card">
                  <QRCodeCanvas value={profileUrl} size={240} />
                </div>
              ) : null}
            </div>
          </div>
        </div>
      </section>
    </Layout>
  );
}