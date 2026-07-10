// frontend/amol/src/pages/HowToUsePage.tsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

import "../styles/page-layout.css";
import "../styles/how-to-use-page.css";

type Step = {
  number: string;
  title: string;
  description: string;
  youtubeUrl: string;
  links?: {
    label: string;
    url: string;
  }[];
};

const sellerSteps: Step[] = [
  {
    number: "01",
    title: "ブランド登録",
    description:
      "ブランドの名前、ロゴ、基本情報を入力し、ブランド専用ブロックチェーンウォレットを開設します。",
    youtubeUrl: "https://www.youtube.com/embed/CcpB-IrE4S0",
  },
  {
    number: "02",
    title: "メンバー招待",
    description:
      "ブランド運営に関わるメンバーを招待し、管理画面へのアクセス権を付与します。",
    youtubeUrl: "https://www.youtube.com/embed/B9pQoIeB-6Q",
  },
  {
    number: "03",
    title: "商品設計",
    description:
      "商品の名前、型番、採寸、色などの基本情報を登録します。",
    youtubeUrl: "https://www.youtube.com/embed/_Wm9PvLmkww",
  },
  {
    number: "04",
    title: "トークン設計",
    description:
      "ブロックチェーントークンのアイコン画像とコンテンツ画像、動画をアップロードします。",
    youtubeUrl: "https://www.youtube.com/embed/fACaWenex1E",
  },
  {
    number: "05",
    title: "生産",
    description:
      "型番毎の生産数を記入し、商品毎に固有のQRコードを印刷します。",
    youtubeUrl: "https://www.youtube.com/embed/bs9V3BrU6jE",
  },
  {
    number: "06",
    title: "検品",
    description:
      "検品スキャナーでQRコードをスキャンして検品結果を入力します。",
    youtubeUrl: "https://www.youtube.com/embed/Xd-HX6Mhoto",
    links: [
      {
        label: "検品スキャナーへログイン",
        url: "https://amol-inspector.web.app/",
      },
    ],
  },
  {
    number: "07",
    title: "ミント",
    description:
      "検品合格した商品にブロックチェーントークンを連携します。",
    youtubeUrl: "https://www.youtube.com/embed/n-8hQnyi-r4",
  },
  {
    number: "08",
    title: "出品",
    description:
      "すべての準備が完了した商品を購入者Mallに出品し、販売を開始します。",
    youtubeUrl: "https://www.youtube.com/embed/DS_S04o7LxA",
  },
  {
    number: "09",
    title: "注文・レビュー確認",
    description:
      "Mallからの注文と購入者からのレビューを確認します。",
    youtubeUrl: "https://www.youtube.com/embed/GiNrATQWiEk",
  },
  {
    number: "10",
    title: "告知",
    description:
      "発行したトークンを所有するアバターへお知らせを一斉送信します。",
    youtubeUrl: "https://www.youtube.com/embed/8-L7mRXos04",
  },
];

const buyerSteps: Step[] = [
  {
    number: "01",
    title: "アバター登録",
    description:
      "購入者Mallでアカウントを作成し、アバター情報を登録します。",
    youtubeUrl: "https://www.youtube.com/embed/31my_0TYhuw",
  },
  {
    number: "02",
    title: "購入",
    description:
      "一般的な購入フローで商品を購入し、届いた商品のQRコードをスキャンするだけでブロックチェーントークンがアバターに移譲されます。",
    youtubeUrl: "https://www.youtube.com/embed/XJuQe9XhQ3k",
  },
  {
    number: "03",
    title: "レビュー投稿",
    description:
      "購入後に商品の体験や評価をレビューとして投稿し、他の購入者に共有します。",
    youtubeUrl: "https://www.youtube.com/embed/RgLNaWVdWVE",
  },
  {
    number: "04",
    title: "お問い合わせ",
    description:
      "購入した商品や取引内容について、購入者Mallから出品者へお問い合わせを送信できます。返信が届いた場合は、問い合わせ詳細からこれまでのやり取りを確認できます。",
    youtubeUrl: "https://www.youtube.com/embed/rxoMwIyRt8Y",
  },
    {
    number: "05",
    title: "フリマ",
    description:
      "所有しているトークンをフリマへ出品することができます。",
    youtubeUrl: "https://www.youtube.com/embed/Tt2W_l-C79c",
  },
];

function StepFlow({
  title,
  subtitle,
  steps,
}: {
  title: string;
  subtitle: string;
  steps: Step[];
}) {
  return (
    <section className="how-to-use-flow">
      <div className="how-to-use-flow__header">
        <h2 className="how-to-use-flow__title">{title}</h2>
        <p className="how-to-use-flow__subtitle">{subtitle}</p>
      </div>

      <div className="how-to-use-flow__steps">
        {steps.map((step, index) => {
          const isLastStep = index === steps.length - 1;

          return (
            <div key={step.number} className="how-to-use-step-wrap">
              <article className="how-to-use-step">
                <div className="how-to-use-step__content">
                  <div className="how-to-use-step__label">
                    Step {step.number}
                  </div>

                  <h3 className="how-to-use-step__title">{step.title}</h3>

                  <p className="how-to-use-step__description">
                    {step.description}
                  </p>

                  {step.links && step.links.length > 0 && (
                    <div className="how-to-use-step__links">
                      {step.links.map((link) => (
                        <a
                          key={link.url}
                          className="how-to-use-step__link"
                          href={link.url}
                          target="_blank"
                          rel="noreferrer"
                        >
                          {link.label}
                        </a>
                      ))}
                    </div>
                  )}
                </div>

                <div className="how-to-use-step__media">
                  <iframe
                    className="how-to-use-step__iframe"
                    src={step.youtubeUrl}
                    title={`${step.title} の実演動画`}
                    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                    allowFullScreen
                  />
                </div>
              </article>

              {!isLastStep && <div className="how-to-use-step__connector" />}
            </div>
          );
        })}
      </div>
    </section>
  );
}

export default function HowToUsePage() {
  const navigate = useNavigate();

  useEffect(() => {
    window.scrollTo({
      top: 0,
      left: 0,
      behavior: "auto",
    });
  }, []);

  return (
    <Layout title="AMOL" mode="landing">
      <main className="how-to-use-page">
        <div className="how-to-use-page__inner">
          <header className="how-to-use-page__header">
            <p className="how-to-use-page__eyebrow">How to use</p>
            <h1 className="how-to-use-page__title">使い方</h1>
            <p className="how-to-use-page__lead">
              出品者Consoleと購入者Mallの使い方を解説します。
              <br />
              一連の操作には出品するテナント役と購入するお客様役が必要です。
            </p>
          </header>

          <div className="how-to-use-page__hero-image-wrap">
            <img
              className="how-to-use-page__hero-image"
              src="/HowToUse.png"
              alt="AMOLのConsoleとMallの使い方"
            />
          </div>

          <StepFlow
            title="出品者Consoleの使い方"
            subtitle="ブランド登録からブロックチェーン連携、出品までの10のステップ"
            steps={sellerSteps}
          />

          <StepFlow
            title="購入者Mallの使い方"
            subtitle="アバター登録から購入、レビュー投稿、トークン交換、お問い合わせまでの5つのステップ"
            steps={buyerSteps}
          />

          <div className="page-actions">
            <Button
              variant="primary"
              onClick={() => navigate("/signin/select")}
            >
              試作品を体験
            </Button>
          </div>
        </div>
      </main>
    </Layout>
  );
}