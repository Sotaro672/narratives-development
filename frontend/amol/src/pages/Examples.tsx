// frontend/amol/src/pages/Examples.tsx
import "../styles/page-layout.css";
import "../styles/vision-page.css";

import Layout from "../components/layout/Layout";

const examples = [
  {
    title: "アパレルブランド",
    description:
      "商品ごとに電子名札を付与し、正規品であることを購入者が確認できるようにします。偽物対策やブランド価値の保護に活用できます。",
  },
  {
    title: "酒類・地域産品",
    description:
      "生産者、産地、製造情報を商品と紐づけ、購入者へ信頼できる情報を届けます。地域ブランドや限定商品の真正性証明に活用できます。",
  },
  {
    title: "限定商品・コレクション商品",
    description:
      "数量限定品やコラボ商品に電子名札を付与し、所有証明や二次流通時の真正性確認に活用できます。",
  },
];

export default function Examples() {
  return (
    <Layout title="AMOL" mode="landing">
      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <section id="examples" className="landing-page-vision">
            <div className="landing-page-vision__header">
              <h2 className="landing-page-vision__title">想定導入事例</h2>
              <p className="landing-page-vision__lead">
                AMOLを導入できる代表的なユースケース
              </p>
              <p className="landing-page-vision__description">
                AMOLは、商品の真正性や所有を証明したい事業者向けに、
                ブロックチェーン技術を活用した電子名札を提供します。
              </p>
            </div>

            <div className="landing-page-vision__steps">
              {examples.map((item, index) => (
                <div key={item.title} className="landing-page-vision-step-wrap">
                  <article
                    className={[
                      "landing-page-vision-step",
                      index === 0 ? "landing-page-vision-step--current" : "",
                    ]
                      .filter(Boolean)
                      .join(" ")}
                  >
                    <div className="landing-page-vision-step__content">
                      <div className="landing-page-vision-step__meta">
                        <span className="landing-page-vision-step__step">
                          Example {index + 1}
                        </span>
                      </div>

                      <h3 className="landing-page-vision-step__title">
                        {item.title}
                      </h3>
                      <p className="landing-page-vision-step__description">
                        {item.description}
                      </p>
                    </div>
                  </article>

                  {index < examples.length - 1 && (
                    <div className="landing-page-vision-step__connector" />
                  )}
                </div>
              ))}
            </div>
          </section>
        </div>
      </section>
    </Layout>
  );
}