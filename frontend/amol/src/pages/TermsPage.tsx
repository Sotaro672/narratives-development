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
                AMOLは、ブランド、商品、在庫、検品、販売、購入、配送、トークンおよびデジタル証明等を管理・利用するためのサービスです。
                ユーザーは、本サービスを通じて、商品情報の閲覧、商品の購入、購入履歴の確認、保有トークンまたはデジタル証明の確認、アバターおよびウォレット機能の利用、その他当社が定める機能を利用できます。
              </p>
              <p className="landing-page-card__text">
                本サービス上で表示されるトークン、デジタル証明、保有情報その他のデジタル情報は、対象商品との関係性、購入履歴、保有状態その他当社が定める情報を表示・確認するためのものであり、当社が別途明示する場合を除き、金融商品、暗号資産、前払式支払手段その他の決済手段として提供されるものではありません。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第3条（利用登録）</h2>
              <p className="landing-page-card__text">
                ユーザーは、当社所定の方法により利用登録を行うものとします。
                登録情報に虚偽、誤り、漏れがある場合、当社は登録を拒否または削除できるものとします。
              </p>
              <p className="landing-page-card__text">
                ユーザーは、登録情報、配送先情報、決済情報その他本サービスの利用に必要な情報を、常に正確かつ最新の状態に保つものとします。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第4条（商品購入および支払い）</h2>
              <p className="landing-page-card__text">
                ユーザーは、本サービス上で表示される商品価格、送料、手数料、配送時期、返品条件その他の販売条件を確認したうえで、商品を購入するものとします。
              </p>
              <p className="landing-page-card__text">
                商品の代金、送料、手数料その他購入に必要な費用は、各商品の購入ページまたは決済画面に表示します。
                支払い方法は、クレジットカードその他購入に係る決済を代行する会社が提供する方法によるものとします。
              </p>
              <p className="landing-page-card__text">
                注文確定後のキャンセル、返品、交換は、法令上必要な場合、商品に不良、破損、誤配送その他契約内容への不適合がある場合、または当社が別途認める場合を除き、原則としてできないものとします。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">
                第5条（配送、検品および提供時期）
              </h2>
              <p className="landing-page-card__text">
                実物商品は、決済完了後、各商品の購入ページまたは当社が別途表示する条件に従い発送します。
                予約販売、受注生産、検品、ミント処理その他特別な条件がある商品については、各商品の購入ページに表示する条件が適用されます。
              </p>
              <p className="landing-page-card__text">
                商品に付随するトークン、デジタル証明、保有情報その他AMOL上の機能は、決済完了後または当社所定の処理完了後に利用可能となります。
                ブロックチェーン、外部ネットワーク、ウォレットその他外部サービスの状況により、反映まで時間を要する場合があります。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">
                第6条（アバター、ウォレットおよびトークン）
              </h2>
              <p className="landing-page-card__text">
                ユーザーは、本サービス上でアバターを作成し、当該アバターに紐づくウォレット、保有トークン、購入履歴、フォロー、フォロワーその他の機能を利用できる場合があります。
              </p>
              <p className="landing-page-card__text">
                トークン、デジタル証明、ウォレット情報その他の表示情報は、ブロックチェーン、外部サービス、通信環境、システム処理状況等により、実際の状態と反映タイミングが異なる場合があります。
                当社は、表示情報の正確性、完全性、即時性を保証するものではありません。
              </p>
              <p className="landing-page-card__text">
                ユーザーは、当社が別途認める範囲で、アバター間のトークンまたはコンテンツの共有、譲渡その他の操作を行うことができます。
                ただし、当該操作の完了後は、サービスの性質上、取消しまたは復元ができない場合があります。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第7条（禁止事項）</h2>
              <p className="landing-page-card__text">
                ユーザーは、法令または公序良俗に違反する行為、虚偽の情報を登録する行為、
                他人になりすます行為、不正アクセス、本サービスの運営を妨害する行為、
                商品、トークン、デジタル証明その他本サービス上の情報を不正に取得、改ざん、複製、転売または悪用する行為、
                その他当社が不適切と判断する行為をしてはなりません。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第8条（表示情報）</h2>
              <p className="landing-page-card__text">
                本サービスでは、商品情報、在庫情報、購入履歴、注文状況、配送状況、検品状況、
                トークン情報、デジタル証明、アバター情報、フォローおよびフォロワー情報等が表示される場合があります。
              </p>
              <p className="landing-page-card__text">
                当社は、表示情報の正確性、完全性、最新性、即時性を保証するものではありません。
                表示情報に誤り、遅延、欠落等がある場合、当社は合理的な範囲で修正または補正を行うことがあります。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第9条（知的財産権）</h2>
              <p className="landing-page-card__text">
                本サービスに関するシステム、デザイン、画像、文章、ロゴ、プログラムその他一切のコンテンツに関する知的財産権は、当社または正当な権利を有する第三者に帰属します。
              </p>
              <p className="landing-page-card__text">
                ユーザーは、当社の事前の許可なく、本サービス上のコンテンツを複製、転載、改変、配布、販売その他当社が認めない方法で利用してはなりません。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第10条（免責）</h2>
              <p className="landing-page-card__text">
                当社は、本サービス、商品情報、トークン、デジタル証明その他本サービスに関する情報について、事実上または法律上の瑕疵がないことを保証しません。
              </p>
              <p className="landing-page-card__text">
                当社は、外部サービス、決済事業者、配送事業者、ブロックチェーンネットワーク、ウォレット、通信環境、端末環境等に起因する損害について、当社に故意または重過失がある場合を除き、責任を負いません。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第11条（サービスの変更・停止）</h2>
              <p className="landing-page-card__text">
                当社は、必要と判断した場合、本サービスの全部または一部を変更、追加、停止または終了できるものとします。
                当社は、当該変更、停止または終了によりユーザーに生じた損害について、当社に故意または重過失がある場合を除き、責任を負いません。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第12条（規約変更）</h2>
              <p className="landing-page-card__text">
                当社は、必要と判断した場合、本規約を変更できるものとします。
                変更後の規約は、本サービス上に表示した時点または当社が別途定める時点から効力を生じます。
              </p>
            </section>

            <section className="terms-page__section">
              <h2 className="terms-page__heading">第13条（準拠法・管轄）</h2>
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