// frontend/src/pages/PrivacyPolicyPage.tsx
import "../styles/landing-page.css";
import "../styles/terms-page.css";

import Layout from "../components/layout/Layout";

export default function PrivacyPolicyPage() {
  return (
    <Layout title="AMOL" mode="landing">
      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <header className="how-to-use-page__header">
            <p className="how-to-use-page__eyebrow">Privacy Policy</p>
            <h1 className="how-to-use-page__title">プライバシーポリシー</h1>
          </header>

          <div className="landing-page-card terms-page">
            <section className="terms-page__section">
              <h2 className="terms-page__heading">1. 基本方針</h2>
              <p className="landing-page-card__text">
                AMOL運営者（以下「当社」といいます。）は、AMOL
                （以下「本サービス」といいます。）におけるユーザーの個人情報の重要性を認識し、
                個人情報の保護に関する法令その他の規範を遵守し、適切に取り扱います。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">2. 取得する情報</h2>
              <p className="landing-page-card__text">
                当社は、本サービスの提供にあたり、氏名またはアバター名、メールアドレス、
                ログイン情報、決済に関する情報、利用履歴、お問い合わせ内容、端末情報、
                Cookieその他これに類する情報を取得することがあります。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">3. 利用目的</h2>
              <p className="landing-page-card__text">
                取得した情報は、本サービスの提供、本人確認、ログイン認証、決済処理、
                お問い合わせ対応、不正利用防止、機能改善、利用状況分析、
                重要なお知らせの通知等のために利用します。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">4. 第三者提供</h2>
              <p className="landing-page-card__text">
                当社は、法令に基づく場合を除き、本人の同意なく個人情報を第三者に提供しません。
                ただし、決済処理、認証、インフラ提供等に必要な範囲で、
                業務委託先または外部サービス提供事業者に情報を提供することがあります。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">5. 安全管理</h2>
              <p className="landing-page-card__text">
                当社は、個人情報への不正アクセス、漏えい、滅失またはき損の防止その他のために、
                必要かつ適切な安全管理措置を講じます。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">6. 開示・訂正・削除等</h2>
              <p className="landing-page-card__text">
                ユーザーは、当社に対し、法令の定めに従って自己の個人情報の開示、訂正、
                追加、削除、利用停止等を求めることができます。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">7. Cookie等の利用</h2>
              <p className="landing-page-card__text">
                当社は、利便性向上、利用状況分析、不正利用防止等のために、
                Cookieその他これに類する技術を利用することがあります。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">8. 改定</h2>
              <p className="landing-page-card__text">
                当社は、必要に応じて本ポリシーを改定することがあります。
                改定後の内容は、本サービス上に表示した時点または別途定める時点から効力を生じます。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">9. お問い合わせ</h2>
              <p className="landing-page-card__text">
                個人情報の取扱いに関するお問い合わせは、お問い合わせページよりご連絡ください。
              </p>
            </section>
          </div>
        </div>
      </section>
    </Layout>
  );
}