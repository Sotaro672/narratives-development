// frontend/amol/src/pages/TradePage.tsx

import { useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";

import "../styles/page-layout.css";

export default function TradePage() {
  const { orderId = "" } = useParams<{
    orderId: string;
  }>();

  return (
    <Layout
      title="取引"
      mode="mypage"
      showFooter
    >
      <section className="content-page-section">
        <div className="page-stack">
          <p>取引情報</p>
          {orderId ? (
            <p>注文ID: {orderId}</p>
          ) : null}
        </div>
      </section>
    </Layout>
  );
}