// frontend/amol/src/pages/LandingPage.tsx
import { useCallback, useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { onAuthStateChanged, type User } from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/landing-page.css";
import "../styles/price-plan-page.css";
import "../styles/company-overview.css";
import "../styles/contact-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import Button from "../components/ui/Button";
import ContactForm from "../features/contact/components/ContactForm";
import { useContactAttachments } from "../features/contact/hooks/useContactAttachments";
import { useContactSubmit } from "../features/contact/hooks/useContactSubmit";
import { auth } from "../lib/firebase";

const subscriptionPlanColumns = [
  "Starter",
  "Simple",
  "Grow",
  "Advanced",
  "Enterprise",
];

const subscriptionPlanRows = [
  {
    label: "料金",
    values: [
      "4,990円/月",
      "9,990円/月",
      "19,990円/月",
      "29,990円/月",
      "別途相談",
    ],
  },
  {
    label: "電子名札発行",
    values: ["〇", "〇", "〇", "〇", "〇"],
  },
  {
    label: "お問い合わせ",
    values: ["〇", "〇", "〇", "〇", "〇"],
  },
  {
    label: "レビュー",
    values: ["×", "〇", "〇", "〇", "〇"],
  },
  {
    label: "一斉告知",
    values: ["×", "×", "〇", "〇", "〇"],
  },
  {
    label: "メンバー招待",
    values: ["×", "×", "×", "〇", "〇"],
  },
  {
    label: "ブランド数",
    values: ["1", "1", "1", "無制限", "無制限"],
  },
];

const companyOverviewRows = [
  {
    label: "商号又は名称",
    value: "株式会社ＡＭＯＬ",
  },
  {
    label: "商号又は名称（フリガナ）",
    value: "アモル",
  },
  {
    label: "法人番号",
    value: "5010901059633",
  },
  {
    label: "代表者",
    value: "奥岡 曹太朗",
  },
  {
    label: "本店所在地",
    value:
      "東京都世田谷区太子堂４丁目１８番１５号 マガザン三軒茶屋２－３Ｆ－３",
  },
  {
    label: "法人番号指定年月日",
    value: "令和8年5月29日",
  },
  {
    label: "最終更新年月日",
    value: "令和8年5月29日",
  },
];

export default function LandingPage() {
  const navigate = useNavigate();
  const location = useLocation();

  const authenticationEyebrowRef = useRef<HTMLParagraphElement | null>(null);
  const salesSupportEyebrowRef = useRef<HTMLParagraphElement | null>(null);
  const fleaMarketEyebrowRef = useRef<HTMLParagraphElement | null>(null);
  const contactSectionRef = useRef<HTMLElement | null>(null);

  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [authResolved, setAuthResolved] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [isContactSectionVisible, setIsContactSectionVisible] = useState(false);

  const {
    mediaInputRef,
    carouselRef,
    carouselIndex,
    attachments,
    setAttachments,
    setCarouselIndex,
    handleFilesSelected,
    handleRemoveAttachment,
    handleCarouselScroll,
    handleMoveToSlide,
    revokeAllAttachmentPreviewUrls,
  } = useContactAttachments();

  const isLoggedIn = !!currentUser;
  const isDesktop = !isMobile;

  const {
    name,
    setName,
    guestEmail,
    setGuestEmail,
    company,
    setCompany,
    message,
    setMessage,
    submitting,
    handleSubmit,
  } = useContactSubmit({
    currentUser,
    isLoggedIn,
    attachments,
    setAttachments,
    setCarouselIndex,
    revokeAllAttachmentPreviewUrls,
  });

  const shouldShowGuestEmailInput = authResolved && !isLoggedIn;
  const submitButtonLabel = submitting ? "送信中..." : "問い合わせる";
  const shouldShowFooterNav = authResolved && isLoggedIn && isMobile;

  const scrollToElement = useCallback((element: HTMLElement | null) => {
    if (!element || typeof window === "undefined") {
      return;
    }

    const isMobileViewport = window.matchMedia("(max-width: 767px)").matches;
    const headerOffset = isMobileViewport ? 72 : 88;
    const elementTop =
      window.scrollY + element.getBoundingClientRect().top - headerOffset;

    window.scrollTo({
      top: Math.max(0, elementTop),
      behavior: "smooth",
    });
  }, []);

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

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const sectionId = location.hash.replace(/^#/, "").trim();

    if (!sectionId) {
      return;
    }

    let firstFrameId = 0;
    let secondFrameId = 0;

    firstFrameId = window.requestAnimationFrame(() => {
      secondFrameId = window.requestAnimationFrame(() => {
        const section = document.getElementById(sectionId);
        scrollToElement(section);
      });
    });

    return () => {
      window.cancelAnimationFrame(firstFrameId);
      window.cancelAnimationFrame(secondFrameId);
    };
  }, [location.hash, location.key, scrollToElement]);

  useEffect(() => {
    if (typeof window === "undefined" || isDesktop) {
      setIsContactSectionVisible(false);
      return;
    }

    const contactSection = contactSectionRef.current;

    if (!contactSection) {
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        setIsContactSectionVisible(entry.isIntersecting);
      },
      {
        threshold: 0.1,
      },
    );

    observer.observe(contactSection);

    return () => {
      observer.disconnect();
    };
  }, [isDesktop]);

  const scrollToAuthentication = () => {
    scrollToElement(authenticationEyebrowRef.current);
  };

  const scrollToSalesSupport = () => {
    scrollToElement(salesSupportEyebrowRef.current);
  };

  const scrollToFleaMarket = () => {
    scrollToElement(fleaMarketEyebrowRef.current);
  };

  return (
    <Layout title="AMOL" mode="landing">
      <div
        className={[
          "landing-page",
          shouldShowFooterNav || (!isDesktop && isContactSectionVisible)
            ? "landing-page--with-footer-nav"
            : "",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <section className="landing-page-hero">
          <div className="landing-page-hero__inner">
            <div className="landing-page-hero__content">
              <p className="landing-page-hero__eyebrow">
                二次流通まで繋げる真贋証明
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

            <div className="landing-page-hero__video-wrap">
              <iframe
                className="landing-page-hero__video"
                src="https://www.youtube.com/embed/fOH4hQUXwhc"
                title="AMOL 紹介動画"
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                referrerPolicy="strict-origin-when-cross-origin"
                allowFullScreen
              />
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
                aria-label="フリーマーケットの詳細へ移動"
                onClick={scrollToFleaMarket}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    scrollToFleaMarket();
                  }
                }}
              >
                <h2 className="landing-page-feature-card__title">
                  フリーマーケット
                </h2>

                <p className="landing-page-feature-card__text">
                  フリーマーケットでの売上の5%をブランド様に還元します。
                </p>

                <div className="landing-page-feature-card__image-placeholder">
                  <img
                    src="/2ndCustomer.png"
                    alt="フリーマーケットで二次流通した商品の所有者が更新される図"
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

            <div className="landing-page-anti-copy">
              <div className="landing-page-anti-copy__header">
                <h2 className="landing-page-section__title landing-page-anti-copy__title">
                  QRコードがコピーされたらどう本物を判別する？
                </h2>
              </div>

              <div className="landing-page-anti-copy__body">
                <div className="landing-page-anti-copy__image-wrap">
                  <img
                    src="/antiCopy.png"
                    alt="QRコードのコピーとブロックチェーントークンによる真贋判定のイメージ"
                    className="landing-page-anti-copy__image"
                    loading="lazy"
                  />
                </div>

                <div className="landing-page-anti-copy__content">
                  <p className="landing-page-anti-copy__text">
                    QRコード自体はコピーできますが、
                    ブロックチェーントークンの移譲履歴は枝分かれできません。
                    模倣品偽造業者が１点の本物からコピー品を量産したとしても、正常に決済処理できるのは１点のみです。
                    よって偽造業者は模造品から利益を上げることができません。
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section
          id="flea-market"
          className="landing-page-section landing-page-sales-support"
        >
          <div className="landing-page-section__inner">
            <div className="landing-page-sales-support__header">
              <p
                ref={fleaMarketEyebrowRef}
                className="landing-page-sales-support__eyebrow"
              >
                フリーマーケット
              </p>

              <h2 className="landing-page-section__title landing-page-sales-support__title">
                利益を上げながら模倣品対策
              </h2>

              <p className="landing-page-card__text landing-page-sales-support__lead">
                ブランドが利益を回収できるフリマを提供します。
              </p>
            </div>

            <div className="landing-page-sales-support__benefits">
              <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
                <div className="landing-page-sales-support-benefit__content">
                  <p className="landing-page-sales-support-card__label">
                    真贋証明をフリマにまで届ける
                  </p>

                  <h3 className="landing-page-sales-support-benefit__title">
                    真贋証明付きで出品できるフリマ
                  </h3>

                  <p className="landing-page-sales-support-benefit__text">
                    AMOLのフリマでは製造時にブロックチェーントークンを連携した電子名札を発行した商品のみ出品されます。
                    本物であることが証明されているため、従来のフリマよりも高く販売できることが期待できます。
                    結果、店舗での購入価格とフリマでの販売価格の差額が縮まり、
                    これまで手が届かなかったお客様を値下げをせずに新規顧客として迎えられるようになります。
                  </p>

                  <div className="landing-page-sales-support-benefit__image-wrap">
                    <img
                      src="/freemarket_expect.jpg"
                      alt="二次流通市場でブランドに利益が還元されにくい状態"
                      className="landing-page-sales-support-benefit__image"
                      loading="lazy"
                    />
                  </div>
                </div>
              </article>

              <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
                <div className="landing-page-sales-support-benefit__content">
                  <p className="landing-page-sales-support-card__label">
                    ブランドが利益を回収できるフリマ
                  </p>

                  <h3 className="landing-page-sales-support-benefit__title">
                    フリーマーケット売上の5%をブランド様に還元
                  </h3>

                  <p className="landing-page-sales-support-benefit__text">
                    AMOL登録商品がフリーマーケットで販売された場合、売上の5%をブランド様に還元します。長く使われる商品ほど、二次流通市場でも継続的な利益機会をつくることができます。
                  </p>

                  <div className="landing-page-sales-support-benefit__image-wrap">
                    <img
                      src="/freemarket_5percent.jpg"
                      alt="フリーマーケットでの売上の一部がブランドに還元される状態"
                      className="landing-page-sales-support-benefit__image"
                      loading="lazy"
                    />
                  </div>
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
              <Button variant="primary" onClick={() => navigate("/how-to-use")}>
                使い方解説
              </Button>
            </div>
          </div>
        </section>

        <section
          id="pricing"
          className="landing-page-section landing-page-pricing"
        >
          <div className="landing-page-section__inner">
            <div className="landing-page-sales-support__header">
              <p className="landing-page-sales-support__eyebrow">利用料金</p>

              <h2 className="landing-page-section__title landing-page-sales-support__title">
                本番運用時の料金体系
              </h2>

              <p className="landing-page-card__text landing-page-sales-support__lead">
                現在試作品段階です。本番運用リリース時は以下の料金体系を予定しております。
              </p>
            </div>

            <div className="landing-page-pricing-grid">
              <article className="landing-page-pricing-card">
                <p className="landing-page-pricing-card__label">基本利用料金</p>

                <h3 className="landing-page-pricing-card__price">
                  4,990円/月～
                </h3>

                <p className="landing-page-pricing-card__badge">初月無料</p>

                <p className="landing-page-pricing-card__text">
                  試験運用価格であり、今後金額が上下する可能性があります。
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
                  販売手数料
                </p>

                <h3 className="landing-page-pricing-card__price">売上の10%</h3>

                <p className="landing-page-pricing-card__text">
                  AMOLモール上で商品が販売された場合に発生します。
                </p>

                <p className="landing-page-pricing-card__text price-plan-fee-card__description--emphasis">
                  自社EC接続の場合は販売手数料をいただきません。
                </p>

                <p className="landing-page-pricing-card__text">
                  自社ECとの接続工事費は別途相談させてください。
                </p>
              </article>
            </div>

            <h3 className="price-plan-page__table-title">基本料金表</h3>

            <div className="price-plan-table-wrap">
              <table className="price-plan-table">
                <thead>
                  <tr>
                    <th scope="col" className="price-plan-table__corner">
                      プラン
                    </th>

                    {subscriptionPlanColumns.map((column) => (
                      <th key={column} scope="col">
                        {column}
                      </th>
                    ))}
                  </tr>
                </thead>

                <tbody>
                  {subscriptionPlanRows.map((row) => (
                    <tr key={row.label}>
                      <th scope="row">{row.label}</th>

                      {row.values.map((value, index) => (
                        <td
                          key={`${row.label}-${subscriptionPlanColumns[index]}`}
                          className={
                            value === "〇"
                              ? "price-plan-table__available"
                              : value === "×"
                                ? "price-plan-table__unavailable"
                                : ""
                          }
                        >
                          {value}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </section>

        <section
          id="company-overview"
          className="landing-page-section landing-page-company-overview"
        >
          <div className="landing-page-section__inner">
            <header className="landing-page-company-overview__header">
              <p className="landing-page-sales-support__eyebrow">Company</p>

              <h2 className="landing-page-section__title landing-page-company-overview__title">
                会社概要
              </h2>
            </header>

            <div className="landing-page-company-overview__content">
              <div className="landing-page-company-overview__table-wrap">
                <table className="landing-page-company-overview__table">
                  <tbody>
                    {companyOverviewRows.map((row) => (
                      <tr key={row.label}>
                        <th scope="row">{row.label}</th>
                        <td>{row.value}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <figure className="landing-page-company-overview__portrait">
                <div className="landing-page-company-overview__portrait-image-wrap">
                  <img
                    src="/founder.jpg"
                    alt="株式会社AMOL代表 奥岡曹太朗"
                    className="landing-page-company-overview__portrait-image"
                    loading="lazy"
                  />
                </div>

                <figcaption className="landing-page-company-overview__portrait-caption">
                  <span className="landing-page-company-overview__portrait-role">
                    代表
                  </span>

                  <span className="landing-page-company-overview__portrait-name">
                    奥岡 曹太朗
                  </span>
                </figcaption>
              </figure>
            </div>
          </div>
        </section>

        <section
          ref={contactSectionRef}
          id="contact"
          className="landing-page-section landing-page-section--with-mobile-footer-action"
        >
          <div className="landing-page-section__inner">
            <header className="how-to-use-page__header">
              <p className="how-to-use-page__eyebrow">Contact</p>
              <h2 className="how-to-use-page__title">お問い合わせ</h2>
            </header>

            <div className="landing-page-card">
              <ContactForm
                shouldShowGuestEmailInput={shouldShowGuestEmailInput}
                name={name}
                guestEmail={guestEmail}
                company={company}
                message={message}
                submitting={submitting}
                attachments={attachments}
                carouselIndex={carouselIndex}
                mediaInputRef={mediaInputRef}
                carouselRef={carouselRef}
                onNameChange={setName}
                onGuestEmailChange={setGuestEmail}
                onCompanyChange={setCompany}
                onMessageChange={setMessage}
                onFilesSelected={handleFilesSelected}
                onRemoveAttachment={handleRemoveAttachment}
                onCarouselScroll={handleCarouselScroll}
                onMoveToSlide={handleMoveToSlide}
              />

              {isDesktop ? (
                <div className="page-actions contact-page__actions">
                  <Button
                    variant="primary"
                    disabled={submitting}
                    onClick={handleSubmit}
                  >
                    {submitButtonLabel}
                  </Button>
                </div>
              ) : null}
            </div>
          </div>
        </section>
      </div>

      {!isDesktop && isContactSectionVisible ? (
        <FooterNav
          variant="action"
          buttonLabel={submitButtonLabel}
          disabled={submitting}
          onButtonClick={handleSubmit}
        />
      ) : shouldShowFooterNav ? (
        <FooterNav />
      ) : null}
    </Layout>
  );
}