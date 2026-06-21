// frontend\amol\src\pages\ExamplesPage.tsx
import { useEffect } from "react";
import { Link } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/examples-page.css";

import Layout from "../components/layout/Layout";

const examples = [
  {
    title: "アパレル",
    to: "/use-cases/apparel",
    description:
      "色、採寸、素材、GSM、洗濯表示を登録できます。ブランドテーマを表現するコンテンツをトークンに掲載し、コーデのアイデアをお客様と共有することができます。",
    image: "/shirts.png",
    imageAlt: "首元のブランドタグにQRコードが付いた赤いTシャツのイラスト",
  },
  {
    title: "酒類",
    to: "/use-cases/alcohol",
    description:
      "生産者、産地、素材、製造年数を登録できます。酒米の生育状況や醸造過程をトークンに掲載し、お客様に蔵出しを楽しみにしてもらう運用ができます。",
    image: "/sake.png",
    imageAlt: "ラベルにQRコードが印字された一升瓶のイラスト",
  },
  {
    title: "化粧品",
    to: "/use-cases/cosmetics",
    description: "現在実装中",
    image: "/cosmetics.png",
    imageAlt: "QRコード付きラベルが付いた化粧品ボトルのイラスト",
  },
];

export default function Examples() {
  useEffect(() => {
    window.scrollTo({
      top: 0,
      left: 0,
      behavior: "auto",
    });
  }, []);

  return (
    <Layout title="AMOL" mode="landing">
      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <section id="examples" className="examples-page">
            <div className="examples-page__header">
              <h2 className="examples-page__title">想定導入事例</h2>
              <p className="examples-page__description">
                現在想定している想定導入事例を業界ごとに提示いたします。
              </p>
            </div>

            <div className="examples-page__grid">
              {examples.map((item) => (
                <Link
                  key={item.title}
                  to={item.to}
                  className="examples-page-card examples-page-card--link"
                >
                  <h3 className="examples-page-card__title">{item.title}</h3>

                  <div className="examples-page-card__image-wrap">
                    <img
                      src={item.image}
                      alt={item.imageAlt}
                      className="examples-page-card__image"
                      loading="lazy"
                    />
                  </div>

                  <p className="examples-page-card__description">
                    {item.description}
                  </p>
                </Link>
              ))}
            </div>
          </section>
        </div>
      </section>
    </Layout>
  );
}