// frontend/amol/src/pages/LandingPage.tsx
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { onAuthStateChanged, type User } from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/landing-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import Button from "../components/ui/Button";
import { auth } from "../lib/firebase";

export default function LandingPage() {
  const navigate = useNavigate();
  const authenticationEyebrowRef = useRef<HTMLParagraphElement | null>(null);
  const salesSupportEyebrowRef = useRef<HTMLParagraphElement | null>(null);

  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [authResolved, setAuthResolved] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const unsubscribe = onAuthStateChanged(auth, (user) => {
      setCurrentUser(user);
      setAuthResolved(true);
    });

    return unsubscribe;
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const mediaQuery = window.matchMedia("(max-width: 1023px)");

    const updateViewportState = () => {
      setIsMobile(mediaQuery.matches);
    };

    updateViewportState();

    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener("change", updateViewportState);

      return () => {
        mediaQuery.removeEventListener("change", updateViewportState);
      };
    }

    mediaQuery.addListener(updateViewportState);

    return () => {
      mediaQuery.removeListener(updateViewportState);
    };
  }, []);

  const shouldShowFooterNav = authResolved && !!currentUser && isMobile;

  const scrollToElement = (element: HTMLElement | null) => {
    if (!element) return;

    if (typeof window === "undefined") {
      element.scrollIntoView({
        behavior: "smooth",
        block: "start",
      });
      return;
    }

    const isMobileViewport = window.matchMedia("(max-width: 767px)").matches;
    const rect = element.getBoundingClientRect();
    const mobileOffset = isMobileViewport ? element.offsetHeight + 56 : 0;

    window.scrollTo({
      top: window.scrollY + rect.top - mobileOffset,
      behavior: "smooth",
    });
  };

  const scrollToAuthentication = () => {
    scrollToElement(authenticationEyebrowRef.current);
  };

  const scrollToSalesSupport = () => {
    scrollToElement(salesSupportEyebrowRef.current);
  };

  return (
    <Layout title="AMOL" mode="landing">
      <div
        className={[
          "landing-page",
          shouldShowFooterNav ? "landing-page--with-footer-nav" : "",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <section className="landing-page-hero">
          <div className="landing-page-hero__inner">
            <div className="landing-page-hero__content">
              <p className="landing-page-hero__eyebrow">
                業務を変えずに真贋を自動証明
              </p>

              <h1 className="landing-page-hero__title">AMOL</h1>

              <div className="page-actions">
                <Button
                  variant="primary"
                  onClick={() => navigate("/how-to-use")}
                >
                  使い方解説
                </Button>
              </div>
            </div>

            <div className="landing-page-hero__image-wrap" aria-hidden="true">
              <img src="/hero.png" alt="" className="landing-page-hero__image" />
            </div>
          </div>
        </section>

        <section className="landing-page-section">
          <div className="landing-page-section__inner">
            <div className="landing-page-feature-grid">
              <article
                className="landing-page-feature-card landing-page-feature-card--clickable"
                role="button"
                tabIndex={0}
                aria-label="真贋証明の詳細へ移動"
                onClick={scrollToAuthentication}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    scrollToAuthentication();
                  }
                }}
              >
                <h2 className="landing-page-feature-card__title">真贋証明</h2>
                <p className="landing-page-feature-card__text">
                  商品のQRコードをスキャンするだけで、製品情報、コメント、所有履歴にアクセスでき、本物であると瞬時に分かります。
                </p>
                <div className="landing-page-feature-card__image-placeholder">
                  <img
                    src="/scan.png"
                    alt="商品QRコードをスキャンした結果画面"
                    className="landing-page-feature-card__image"
                    loading="lazy"
                  />
                </div>
              </article>

              <article
                className="landing-page-feature-card landing-page-feature-card--clickable"
                role="button"
                tabIndex={0}
                aria-label="営業支援の詳細へ移動"
                onClick={scrollToSalesSupport}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    scrollToSalesSupport();
                  }
                }}
              >
                <h2 className="landing-page-feature-card__title">営業支援</h2>
                <p className="landing-page-feature-card__text">
                  商品を誰が所有しているかがリアルタイムで分かり、販売後も新商品の情報を本当に興味のある人に届けることができます。
                </p>
                <div className="landing-page-feature-card__image-placeholder">
                  <img
                    src="/comment.png"
                    alt="商品所有者とのコメント画面"
                    className="landing-page-feature-card__image"
                    loading="lazy"
                  />
                </div>
              </article>
            </div>
          </div>
        </section>

        <section
          id="authentication"
          className="landing-page-section landing-page-service-case"
        >
          <div className="landing-page-section__inner">
            <div className="landing-page-service-case__header">
              <p
                ref={authenticationEyebrowRef}
                className="landing-page-sales-support__eyebrow"
              >
                真贋証明
              </p>
              <h2 className="landing-page-section__title landing-page-service-case__title">
                電子名札で見える商品の変遷
              </h2>
              <p className="landing-page-card__text landing-page-service-case__lead">
                届いた商品のQRコードをスキャンするだけで貴方の商品の所有権を閲覧、記録することができます。
              </p>
            </div>

            <div className="landing-page-service-case__top-grid">
              <article className="landing-page-service-case-card landing-page-service-case-card--proof">
                <div className="landing-page-service-case-card__content">
                  <h3 className="landing-page-service-case-card__title">
                    QRコードを読み取るだけで、現在の所有者を確認できる
                  </h3>
                  <p className="landing-page-service-case-card__text">
                    購入した商品のQRコードを読み取るだけで所有者が購入されたお客様のアバター名に自動的に更新されます。
                  </p>
                </div>

                <div className="landing-page-service-case-card__image-wrap">
                  <img
                    src="/Scaning.png"
                    alt="日本酒ラベルのQRコードを読み取り、所有者情報が更新される流れ"
                    className="landing-page-service-case-card__image"
                    loading="lazy"
                  />
                </div>
              </article>

              <article className="landing-page-service-case-card landing-page-service-case-card--sales">
                <div className="landing-page-service-case-card__content">
                  <h3 className="landing-page-service-case-card__title">
                    あなたの所有権を堅牢に守る仕組み
                  </h3>
                  <p className="landing-page-service-case-card__text">
                    商品の所有権はブロックチェーンネットワークで記録され、外部からの改ざん攻撃から貴方の所有権を堅牢に守ります。
                  </p>
                </div>

                <div className="landing-page-service-case-card__image-wrap">
                  <img
                    src="/BlockchainNetwork.png"
                    alt="世界中のサーバーに所有者情報が記録される図"
                    className="landing-page-service-case-card__image"
                    loading="lazy"
                  />
                </div>
              </article>
            </div>

            <div className="landing-page-service-case__flow-title">
              <span className="landing-page-service-case__flow-line" />
              <p>所有者の移り変わりと、名札の書き換えの流れ</p>
              <span className="landing-page-service-case__flow-line" />
            </div>

            <div className="landing-page-service-case__bottom-grid">
              <article className="landing-page-service-case-card landing-page-service-case-card--brewery">
                <div className="landing-page-service-case-card__content">
                  <p className="landing-page-service-case-card__label">
                    ブランド
                  </p>
                  <h3 className="landing-page-service-case-card__title">
                    生産者・ブランド名を記入する
                  </h3>
                  <p className="landing-page-service-case-card__text">
                    生産したブランド名を最初の所有者として電子名札に名前を書き込みます。
                  </p>
                </div>

                <div className="landing-page-service-case-card__image-wrap">
                  <img
                    src="/Crafter.png"
                    alt="蔵元が日本酒の最初の所有者として登録される図"
                    className="landing-page-service-case-card__image"
                    loading="lazy"
                  />
                </div>
              </article>

              <article className="landing-page-service-case-card landing-page-service-case-card--mall">
                <div className="landing-page-service-case-card__content">
                  <p className="landing-page-service-case-card__label">
                    店舗購入のお客様
                  </p>
                  <h3 className="landing-page-service-case-card__title">
                    QRコードをスキャンで自動更新
                  </h3>
                  <p className="landing-page-service-case-card__text">
                    お客様のアカウントで購入した商品のQRコードをスキャンすると自動で所有者名（アバター名）が更新されます。
                  </p>
                </div>

                <div className="landing-page-service-case-card__image-wrap">
                  <img
                    src="/Customer.png"
                    alt="AMOL MALLで日本酒が販売され、所有者が書き換わる図"
                    className="landing-page-service-case-card__image"
                    loading="lazy"
                  />
                </div>
              </article>

              <article className="landing-page-service-case-card landing-page-service-case-card--customer">
                <div className="landing-page-service-case-card__content">
                  <p className="landing-page-service-case-card__label">
                    二次流通先のお客様
                  </p>
                  <h3 className="landing-page-service-case-card__title">
                    移譲と同時に自動更新
                  </h3>
                  <p className="landing-page-service-case-card__text">
                    フォロー先、フォロワー間でトークンを渡すことができます。譲渡と同時に所有者名（アバター名）が自動更新されます。
                  </p>
                </div>

                <div className="landing-page-service-case-card__image-wrap">
                  <img
                    src="/2ndCustomer.png"
                    alt="個人のお客様が日本酒の所有者として記録される図"
                    className="landing-page-service-case-card__image"
                    loading="lazy"
                  />
                </div>
              </article>
            </div>
          </div>
        </section>

        <section
          id="sales-support"
          className="landing-page-section landing-page-sales-support"
        >
          <div className="landing-page-section__inner">
            <div className="landing-page-sales-support__header">
              <p
                ref={salesSupportEyebrowRef}
                className="landing-page-sales-support__eyebrow"
              >
                営業支援
              </p>
              <h2 className="landing-page-section__title landing-page-sales-support__title">
                購入後もお客様とつながる
              </h2>
              <p className="landing-page-card__text landing-page-sales-support__lead">
                電子名札を通じて、生産者は販売した商品の現在の所有者と繋がることができます。
              </p>
            </div>

            <div className="landing-page-sales-support__benefits">
              <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
                <div className="landing-page-sales-support-benefit__content">
                  <p className="landing-page-sales-support-card__label">
                    これまでの課題
                  </p>

                  <h3 className="landing-page-sales-support-benefit__title">
                    従来のSNSではお客様からフォローしてもらう必要があった。
                  </h3>
                  <p className="landing-page-sales-support-benefit__text">
                    従来のSNSでは折角商品を気に入ってもらっても、お客様からフォローをして頂かないと商品を忘れてしまう可能性があります。
                  </p>
                  <div className="landing-page-sales-support-benefit__image-wrap">
                    <img
                      src="/BeforeConnection1.png"
                      alt="従来のSNSを用いたお客様との繋がり"
                      className="landing-page-sales-support-benefit__image"
                      loading="lazy"
                    />
                  </div>

                  <h3 className="landing-page-sales-support-benefit__title">
                    二次流通を追跡できない
                  </h3>
                  <p className="landing-page-sales-support-benefit__text">
                    従来のSNSでは販売した商品が年々拡大する二次流通市場でどの様なお客様に購入されているのかを追跡することができません。
                  </p>
                  <div className="landing-page-sales-support-benefit__image-wrap">
                    <img
                      src="/BeforeConnection2.png"
                      alt="二次流通でお客様との繋がりを追跡できない状態"
                      className="landing-page-sales-support-benefit__image"
                      loading="lazy"
                    />
                  </div>
                </div>
              </article>

              <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
                <div className="landing-page-sales-support-benefit__content">
                  <p className="landing-page-sales-support-card__label">
                    AMOLでできること
                  </p>

                  <h3 className="landing-page-sales-support-benefit__title">
                    電子名札を介したお客様との繋がり
                  </h3>
                  <p className="landing-page-sales-support-benefit__text">
                    商品を購入していただき、オプトイン操作をしていただいた全てのお客様にお知らせを送ることができます。
                  </p>
                  <div className="landing-page-sales-support-benefit__image-wrap">
                    <img
                      src="/AfterConnection1.png"
                      alt="電子名札を介したお客様との繋がり"
                      className="landing-page-sales-support-benefit__image"
                      loading="lazy"
                    />
                  </div>

                  <h3 className="landing-page-sales-support-benefit__title">
                    二次流通市場でも現在の所有者と繋がれる
                  </h3>
                  <p className="landing-page-sales-support-benefit__text">
                    お客様間で電子名札を譲渡することも可能です。二次流通市場で購入されたお客様にもお知らせをお届けすることができます。
                  </p>
                  <div className="landing-page-sales-support-benefit__image-wrap">
                    <img
                      src="/AfterConnection2.png"
                      alt="二次流通市場でも現在の所有者と繋がれる状態"
                      className="landing-page-sales-support-benefit__image"
                      loading="lazy"
                    />
                  </div>
                </div>
              </article>
            </div>

            <div className="page-actions">
              <Button variant="primary" onClick={() => navigate("/use-cases")}>
                想定導入例
              </Button>
            </div>
          </div>
        </section>

        <section id="pricing" className="landing-page-section landing-page-pricing">
          <div className="landing-page-section__inner">
            <div className="landing-page-sales-support__header">
              <p className="landing-page-sales-support__eyebrow">利用料金</p>
              <h2 className="landing-page-section__title landing-page-sales-support__title">
                予定料金プラン
              </h2>
              <p className="landing-page-card__text landing-page-sales-support__lead">
                現在試作品段階です。本番運用リリース時は以下の料金体系を予定しております。
              </p>
            </div>

            <div className="landing-page-pricing-grid">
              <article className="landing-page-pricing-card">
                <p className="landing-page-pricing-card__label">基本利用料金</p>
                <h3 className="landing-page-pricing-card__price">4,990円/月</h3>
                <p className="landing-page-pricing-card__badge">
                  試験運用価格
                </p>
                <p className="landing-page-pricing-card__text">
                  コンソールの月額利用料です。月毎の契約です。
                </p>
              </article>

              <article className="landing-page-pricing-card">
                <p className="landing-page-pricing-card__label">
                  電子名札発行手数料
                </p>
                <h3 className="landing-page-pricing-card__price">10円/点</h3>
                <p className="landing-page-pricing-card__text">
                  発行した点数に応じて課金されます。
                </p>
              </article>

              <article className="landing-page-pricing-card">
                <p className="landing-page-pricing-card__label">
                  モール販売手数料
                </p>
                <h3 className="landing-page-pricing-card__price">売上の10%</h3>
                <p className="landing-page-pricing-card__text">
                  テナント様所有EC経由での販売は対象外です。
                  <br />
                  接続工事費は別途相談させてください。
                </p>
              </article>
            </div>

            <div className="page-actions">
              <Button variant="primary" onClick={() => navigate("/pricing")}>
                料金プラン
              </Button>
            </div>
          </div>
        </section>
      </div>

      {shouldShowFooterNav ? <FooterNav /> : null}
    </Layout>
  );
}