// frontend/src/pages/SignUpSelectPage.tsx
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

import "../styles/page-layout.css";
import "../styles/sign-up-select-page.css";

type SignUpTarget = {
  title: string;
  description: string;
  url?: string;
  internalPath?: string;
  buttonLabel: string;
  variant: "primary" | "secondary";
  deviceTags: string[];
};

const signUpTargets: SignUpTarget[] = [
  {
    title: "Console",
    description:
      "商品を登録・管理し、ブロックチェーントークンを発行できる出品者向けのプラットフォーム。",
    url: "https://narratives-console-dev.web.app",
    buttonLabel: "Consoleで新規登録",
    variant: "primary",
    deviceTags: ["PC専用"],
  },
  {
    title: "Mall",
    description:
      "商品を閲覧・購入し、QRコードからトークンコンテンツにアクセスできる購入者向けのモール。",
    internalPath: "/signup",
    buttonLabel: "Mallで新規登録",
    variant: "primary",
    deviceTags: ["PC対応", "モバイル対応"],
  },
];

export default function SignUpSelectPage() {
  const navigate = useNavigate();

  const handleSignUpTargetClick = (target: SignUpTarget) => {
    if (target.internalPath) {
      navigate(target.internalPath);
      return;
    }

    if (target.url) {
      window.open(target.url, "_blank", "noopener,noreferrer");
    }
  };

  return (
    <Layout title="新規登録" mode="landing">
      <main className="sign-up-select-page">
        <div className="sign-up-select-page__inner">
          <header className="sign-up-select-page__header">
            <p className="sign-up-select-page__eyebrow">Sign up</p>
            <p className="sign-up-select-page__lead">
              新規登録するサービスを選択してください。
            </p>
          </header>

          <div className="sign-up-select-page__grid">
            {signUpTargets.map((target) => (
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
                  onClick={() => handleSignUpTargetClick(target)}
                >
                  {target.buttonLabel}
                </Button>
              </article>
            ))}
          </div>

          <p className="sign-up-select-page__note">
            ※ Consoleは別ウィンドウで開きます。
          </p>
        </div>
      </main>
    </Layout>
  );
}