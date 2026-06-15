// frontend/src/pages/SignInSelectPage.tsx
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

import "../styles/page-layout.css";
import "../styles/sign-up-select-page.css";

type SignInTarget = {
  title: string;
  description: string;
  url?: string;
  internalPath?: string;
  buttonLabel: string;
  variant: "primary" | "secondary";
  deviceTags: string[];
};

const signInTargets: SignInTarget[] = [
  {
    title: "コンソール",
    description:
      "商品を登録・管理し、ブロックチェーントークンを発行できる出品者向けのプラットフォーム。",
    url: "https://narratives-console-dev.web.app",
    buttonLabel: "Consoleにログイン",
    variant: "primary",
    deviceTags: ["PC専用"],
  },
  {
    title: "検品スキャナー",
    description:
      "QRコードをスキャンして検品結果を入力できる、Console付属の検品専用アプリ。",
    url: "https://amol-inspector.web.app/",
    buttonLabel: "検品スキャナーにログイン",
    variant: "secondary",
    deviceTags: ["モバイル専用"],
  },
  {
    title: "モール",
    description:
      "商品を閲覧・購入し、QRコードからトークンコンテンツにアクセスできる購入者向けのモール。",
    internalPath: "/signin",
    buttonLabel: "モールにログイン",
    variant: "primary",
    deviceTags: ["PC対応", "モバイル対応"],
  },
];

export default function SignInSelectPage() {
  const navigate = useNavigate();

  const handleSignInTargetClick = (target: SignInTarget) => {
    if (target.internalPath) {
      navigate(target.internalPath);
      return;
    }

    if (target.url) {
      window.open(target.url, "_blank", "noopener,noreferrer");
    }
  };

  return (
    <Layout title="ログイン" mode="landing">
      <main className="sign-up-select-page">
        <div className="sign-up-select-page__inner">
          <header className="sign-up-select-page__header">
            <p className="sign-up-select-page__eyebrow">Sign in</p>
            <p className="sign-up-select-page__lead">
              ログインするサービスを選択してください。
            </p>
          </header>

          <div className="sign-up-select-page__grid sign-up-select-page__grid--three">
            {signInTargets.map((target) => (
              <article key={target.title} className="sign-up-select-card">
                <div className="sign-up-select-card__body">
                  <div className="sign-up-select-card__header">
                    <h2 className="sign-up-select-card__title">
                      {target.title}
                    </h2>

                    <div className="sign-up-select-card__tags">
                      {target.deviceTags.map((tag) => (
                        <span key={tag} className="sign-up-select-card__tag">
                          {tag}
                        </span>
                      ))}
                    </div>
                  </div>

                  <p className="sign-up-select-card__description">
                    {target.description}
                  </p>
                </div>

                <Button
                  variant={target.variant}
                  onClick={() => handleSignInTargetClick(target)}
                >
                  {target.buttonLabel}
                </Button>
              </article>
            ))}
          </div>

          <p className="sign-up-select-page__note">
            ※ ConsoleとInspectorは別ウィンドウで開きます。
          </p>
        </div>
      </main>
    </Layout>
  );
}