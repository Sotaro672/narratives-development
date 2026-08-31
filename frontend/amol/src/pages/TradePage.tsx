// frontend/amol/src/pages/TradePage.tsx

import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";

import "../styles/page-layout.css";

export default function TradePage() {
  const navigate = useNavigate();
  const { orderId = "" } = useParams<{ orderId: string }>();

  return (
    <Layout title="取引" mode="mypage" showBackButton onBackButtonClick={() => navigate(-1)} showFooter>
      <section className="content-page-section">
        <div className="page-stack">
          <p>取引情報</p>
          {orderId ? <p>注文ID: {orderId}</p> : null}
        </div>
      </section>
    </Layout>
  );
}