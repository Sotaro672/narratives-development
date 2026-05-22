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
                  株式会社AMOL
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
                  各商品の購入ページまたは決済画面に表示される金額
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  販売価格以外の必要料金
                </dt>
                <dd className="landing-page-definition-list__description">
                  商品の配送が発生する場合、送料が別途発生する場合があります。
                  <br />
                  送料、手数料その他の必要料金が発生する場合は、各商品の購入ページまたは決済画面に表示します。
                  <br />
                  インターネット接続料金、通信料金その他AMOLの利用に必要な費用は、お客様の負担となります。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  商品の引渡時期・サービスの提供時期
                </dt>
                <dd className="landing-page-definition-list__description">
                  実物商品は、決済完了後、各商品の購入ページまたは当社が別途表示する条件に従い発送します。
                  <br />
                  商品に付随するトークン、デジタル証明、保有情報その他AMOL上の機能は、決済完了後または当社所定の処理完了後に利用可能となります。
                  <br />
                  予約販売、受注生産、検品、ミント処理その他特別な条件がある場合は、各商品の購入ページに表示します。
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
                  キャンセル・返品・交換等
                </dt>
                <dd className="landing-page-definition-list__description">
                  お客様都合による注文確定後のキャンセル、返品、交換は、原則としてお受けできません。
                  <br />
                  ただし、商品に不良、破損、誤配送その他契約内容への不適合がある場合、または法令上必要な場合は、当社所定の方法により対応いたします。
                  <br />
                  返品・交換の可否、条件、送料負担その他特別な条件がある場合は、各商品の購入ページに表示します。
                  <br />
                  商品に付随するトークン、デジタル証明、保有情報その他AMOL上の機能については、サービスの性質上、提供後のキャンセル・返金は原則としてお受けできません。
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
            AMOL上で販売される商品に関して
          </h2>

          <div className="landing-page-card">
            <dl className="landing-page-definition-list">
              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  商品の販売価格
                </dt>
                <dd className="landing-page-definition-list__description">
                  各商品の購入ページまたは決済画面に表示される金額
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  販売価格以外の必要料金
                </dt>
                <dd className="landing-page-definition-list__description">
                  商品の配送が発生する場合、送料が別途発生する場合があります。
                  <br />
                  送料、手数料その他の必要料金が発生する場合は、各商品の購入ページまたは決済画面に表示します。
                  <br />
                  インターネット接続料金、通信料金その他AMOLの利用に必要な費用は、お客様の負担となります。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  商品の引渡時期
                </dt>
                <dd className="landing-page-definition-list__description">
                  決済完了後、各商品の購入ページまたは当社が別途表示する条件に従い発送します。
                  <br />
                  予約販売、受注生産、検品、ミント処理その他特別な条件がある商品については、各商品の購入ページに表示します。
                </dd>
              </div>

              <div className="landing-page-definition-list__row">
                <dt className="landing-page-definition-list__term">
                  トークン・デジタル証明等の提供時期
                </dt>
                <dd className="landing-page-definition-list__description">
                  商品に付随するトークン、デジタル証明、保有情報その他AMOL上の機能は、決済完了後または当社所定の処理完了後に利用可能となります。
                  <br />
                  ブロックチェーン、外部ネットワーク、ウォレットその他外部サービスの状況により、反映まで時間を要する場合があります。
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
                  キャンセル・返品・交換等
                </dt>
                <dd className="landing-page-definition-list__description">
                  お客様都合による注文確定後のキャンセル、返品、交換は、原則としてお受けできません。
                  <br />
                  ただし、商品に不良、破損、誤配送その他契約内容への不適合がある場合、または法令上必要な場合は、当社所定の方法により対応いたします。
                  <br />
                  返品・交換の可否、条件、送料負担その他特別な条件がある場合は、各商品の購入ページに表示します。
                  <br />
                  商品に付随するトークン、デジタル証明、保有情報その他AMOL上の機能については、サービスの性質上、提供後のキャンセル・返金は原則としてお受けできません。
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