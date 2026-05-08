// frontend/src/pages/VerificationSentPage.tsx

import { Link, useSearchParams } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/form.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

export default function VerificationSentPage() {
  const [searchParams] = useSearchParams();
  const email = searchParams.get("email");

  return (
    <Layout title="認証メールを送信しました">
      <section className="page-section">
        <p className="page-description">
          {email ? (
            <>
              <strong>{email}</strong>
              <br />
              宛に認証メールを送信しました。
            </>
          ) : (
            "認証メールを送信しました。"
          )}
          <br />
          メール内のリンクを開いて、メールアドレスの確認を完了してください。
        </p>

        <div className="page-actions">
          <Link to="/signin">
            <Button variant="primary">ログイン画面へ</Button>
          </Link>
        </div>
      </section>
    </Layout>
  );
}