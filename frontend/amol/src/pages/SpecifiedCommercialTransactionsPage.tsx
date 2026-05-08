// frontend/src/pages/SpecifiedCommercialTransactionsPage.tsx
import Layout from "../components/layout/Layout";

export default function SpecifiedCommercialTransactionsPage() {
  return (
    <Layout title="特定商取引法に基づく表記" mode="landing">
      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <p className="landing-page-card__text landing-page-legal-lead">
            「特定商取引に関する法律」第11条に基づき、以下のとおり表示いたします。
          </p>

          <h2 className="landing-page-section__title landing-page-section__title--legal">
            AMOLの利用に関して
          </h2>

          <div className="landing-page-card">
            <dl className="landing-page-definition-list">
              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  販売事業の名称
                </dt>
                <dd className="landing-page-definition-list__description">
                  AMOL
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">所在地</dt>
                <dd className="landing-page-definition-list__description">
                  〒154-0004 東京都世田谷区太子堂4丁目18−15
                  マガザン三軒茶屋2 3階3
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">代表者名</dt>
                <dd className="landing-page-definition-list__description">
                  奥岡 曹太朗
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  お問い合わせ
                </dt>
                <dd className="landing-page-definition-list__description">
                  お問い合わせは問い合わせ窓口ページよりご連絡ください。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">販売価格</dt>
                <dd className="landing-page-definition-list__description">
                  以下の「本サービスの利用にかかる手数料」のみがかかります。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  本サービスの利用にかかる手数料等
                </dt>
                <dd className="landing-page-definition-list__description">
                  支援金額の5%を決済手数料としていただきます。
                  <br />
                  当社が代理受領した支援金をユーザーに支払うまでに生じる振込手数料、決済代行会社所定の手数料、出金時の銀行振込手数料等が発生する場合があります。
                  <br />
                  デジタルサービスのため、送料は発生しません。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  サービスの提供時期
                </dt>
                <dd className="landing-page-definition-list__description">
                  利用規約に基づいて登録いただくことで本サービスをご利用いただけます。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  お支払い方法
                </dt>
                <dd className="landing-page-definition-list__description">
                  当社が代理受領した支援金から控除する形でお支払いいただきます。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  お支払い時期
                </dt>
                <dd className="landing-page-definition-list__description">
                  決済日
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">動作環境</dt>
                <dd className="landing-page-definition-list__description">
                  AMOLの推奨動作環境は以下の通りです。
                  <br />
                  ■ブラウザ Chrome、Safari、それぞれの最新版
                  <br />
                  ■OS iOS、Android、Windows、macOS、それぞれの最新版
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  キャンセル等
                </dt>
                <dd className="landing-page-definition-list__description">
                  一度確定した取引は、サービスの性質上、原則としてキャンセル・返金できません。
                  <br />
                  ただし、法令上必要な場合または当社が別途認める場合はこの限りではありません。
                  <br />
                  また、本サービスについて契約内容への不適合がある場合は、利用規約の定めに従って対応いたします。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">その他</dt>
                <dd className="landing-page-definition-list__description">
                  表示を省略した事項については、お客様からの開示の請求があった場合、法律の定めに従って遅滞なく開示するものとします。
                </dd>
              </div>
            </dl>
          </div>

          <h2 className="landing-page-section__title landing-page-section__title--legal">
            ユーザーが提供する有償サービスに関して
          </h2>

          <div className="landing-page-card">
            <dl className="landing-page-definition-list">
              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  有償サービスの利用料金
                </dt>
                <dd className="landing-page-definition-list__description">
                  決済時に表示される金額
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  利用料金以外の必要料金
                </dt>
                <dd className="landing-page-definition-list__description">
                  デジタルサービスのため、送料は発生しません。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  有償サービスの利用時期
                </dt>
                <dd className="landing-page-definition-list__description">
                  決済完了後、即時
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  お支払い方法
                </dt>
                <dd className="landing-page-definition-list__description">
                  クレジットカード、その他購入に係る決済を代行する会社が提供するお支払い方法
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  お支払い時期
                </dt>
                <dd className="landing-page-definition-list__description">
                  決済日
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">動作環境</dt>
                <dd className="landing-page-definition-list__description">
                  AMOLの推奨動作環境は以下の通りです。
                  <br />
                  ■ブラウザ Chrome、Safari、それぞれの最新版
                  <br />
                  ■OS iOS、Android、Windows、macOS、それぞれの最新版
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  キャンセル等
                </dt>
                <dd className="landing-page-definition-list__description">
                  一度確定した取引は、サービスの性質上、原則としてキャンセル・返金できません。
                  <br />
                  ただし、法令上必要な場合または当社が別途認める場合はこの限りではありません。
                  <br />
                  申込の有効期限や特別な提供条件がある有償サービスについては、各有償サービスの購入ページにおいて条件を表示します。
                  <br />
                  また、有償サービスにかかる契約内容への不適合がある場合は、民法その他の法令の定めに従って対応いたします。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">その他</dt>
                <dd className="landing-page-definition-list__description">
                  表示を省略した事項については、お客様からの開示の請求があった場合、法律の定めに従って遅滞なく開示するものとします。
                </dd>
              </div>
            </dl>
          </div>
        </div>
      </section>
    </Layout>
  );
}