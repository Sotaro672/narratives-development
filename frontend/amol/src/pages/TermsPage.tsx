// frontend/src/pages/TermsPage.tsx
import Layout from "../components/layout/Layout";

export default function TermsPage() {
  return (
    <Layout title="利用規約" mode="landing">
      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <div className="landing-page-card terms-page">
            <section className="terms-page__section">
              <h2 className="terms-page__heading">第1条（適用）</h2>
              <p className="landing-page-card__text">
                本規約は、AMOL運営者が提供する「AMOL」の利用条件を定めるものです。
                ユーザーは、本規約に同意したうえで本サービスを利用するものとします。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第2条（サービス内容）</h2>
              <p className="landing-page-card__text">
                AMOLは、ライブ配信アプリではなく、配信者・推しが応援金を受け取るための集金アプリです。
                既存のライブ配信サービスやSNS、オフラインイベントと併用して利用されることを想定しています。
              </p>
              <p className="landing-page-card__text">
                本サービスでは、応援金の送受信、メッセージ表示、ランキング表示、QRコードによる支援実績確認、
                その他当社が定める機能を提供します。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第3条（利用登録）</h2>
              <p className="landing-page-card__text">
                ユーザーは、当社所定の方法により利用登録を行うものとします。
                登録情報に虚偽がある場合、当社は登録を拒否または削除できるものとします。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第4条（手数料および支払い）</h2>
              <p className="landing-page-card__text">
                本サービスの手数料は、原則として支援金額の5%とします。
                これとは別に、決済事業者所定の手数料等が発生する場合があります。
              </p>
              <p className="landing-page-card__text">
                支援実行後の返金、取消し、キャンセルは、法令上必要な場合または当社が認める場合を除き、
                原則としてできないものとします。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第5条（禁止事項）</h2>
              <p className="landing-page-card__text">
                ユーザーは、法令または公序良俗に違反する行為、虚偽の情報により支援を募る行為、
                他人になりすます行為、不正アクセス、本サービスの運営を妨害する行為、
                その他当社が不適切と判断する行為をしてはなりません。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第6条（表示情報）</h2>
              <p className="landing-page-card__text">
                本サービスでは、支援実績、支援金額、順位、メッセージ等がルーム、ランキング、
                QRコード読取結果その他の画面に表示される場合があります。
              </p>
              <p className="landing-page-card__text">
                当社は、表示情報の正確性、完全性、即時性を保証するものではありません。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第7条（免責）</h2>
              <p className="landing-page-card__text">
                当社は、本サービスに事実上または法律上の瑕疵がないことを保証しません。
                また、外部サービス、決済事業者、通信環境等に起因する損害について責任を負いません。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第8条（規約変更）</h2>
              <p className="landing-page-card__text">
                当社は、必要と判断した場合、本規約を変更できるものとします。
                変更後の規約は、本サービス上に表示した時点または別途定める時点から効力を生じます。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第9条（準拠法・管轄）</h2>
              <p className="landing-page-card__text">
                本規約は日本法に準拠し、本サービスに関して生じる一切の紛争については、
                当社所在地を管轄する地方裁判所を第一審の専属的合意管轄裁判所とします。
              </p>
            </section>
          </div>
        </div>
      </section>
    </Layout>
  );
}