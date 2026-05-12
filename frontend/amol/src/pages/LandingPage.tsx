// frontend/src/pages/LandingPage.tsx
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/landing-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

const team = [
  {
    name: "奥岡 曹太朗",
    role: "開発者",
    bio: "商流で世界を繋げることで平和を築いていくことを目標に、ブロックチェーン技術でお客様の信用と信頼を守ってまいります。",
    career: [
      "2020 東京農業大学 生物応用化学科（農芸化学科）卒業",
      "2022 京都大学大学院農学研究科地域環境科学専攻 土壌学研究室 修士課程修了",
      "2022 NTCインターナショナル株式会社 勤務（国際協力コンサルティング）",
      "2024 株式会社 Mover & Company 勤務（IT保守開発コンサルティング）",
      "2026 株式会社AMOL 設立",
    ],
    image: "/founder.jpg",
  },
];

const visionSteps = [
  {
    step: "Step 1",
    title: "ブロックチェーントークンが付与された商品のみが流通するMall",
    description:
      "本物だけが流通する安心・安全なECモールプラットフォーム。すべての商品にブロックチェーントークンが付与され、真正性が保証されます。",
    isCurrent: true,
  },
  {
    step: "Step 2",
    title: "本物の所有を証明できるSNS",
    description:
      "自分が本当にその商品を持っていることを証明できるSNS。所有している商品についてのみハッシュタグを付けた投稿ができる、信頼性の高いソーシャルプラットフォーム。",
    isCurrent: false,
  },
  {
    step: "Step 3",
    title: "ブロックチェーントークンを交換できるフリマ",
    description:
      "ユーザー間でブロックチェーントークンを交換できるフリマ。本物であることが証明された商品を出品できるため、再販価格が保障されます。上代価格と再販価格の差額が実質的な負担額となります。",
    isCurrent: false,
  },
];

export default function LandingPage() {
  const navigate = useNavigate();

  return (
    <Layout title="AMOL" mode="landing">
      <section className="landing-page-hero">
        <div className="landing-page-hero__inner">
          <div className="landing-page-hero__content">
            <p className="landing-page-hero__eyebrow">
              ブロックチェーンで本物だけを届ける
            </p>

            <h1 className="landing-page-hero__title">AMOL</h1>

            <div className="page-actions">
              <Button
                variant="primary"
                onClick={() => navigate("/signup/select")}
              >
                新規登録
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
            <article className="landing-page-feature-card">
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

            <article className="landing-page-feature-card">
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

          <section id="team" className="landing-page-team">
            <div className="landing-page-team__header">
              <h2 className="landing-page-team__title">チーム</h2>
            </div>

            <div className="landing-page-team__content">
              {team.map((member) => (
                <article key={member.name} className="landing-page-team-card">
                  <div className="landing-page-team-card__image-wrapper">
                    <img
                      src={member.image}
                      alt={member.name}
                      className="landing-page-team-card__image"
                      loading="lazy"
                    />
                  </div>

                  <div className="landing-page-team-card__body">
                    <h3 className="landing-page-team-card__name">
                      {member.name}
                    </h3>
                    <p className="landing-page-team-card__role">
                      {member.role}
                    </p>
                    <p className="landing-page-team-card__bio">
                      {member.bio}
                    </p>

                    <ul className="landing-page-team-card__career">
                      {member.career.map((item) => (
                        <li
                          key={item}
                          className="landing-page-team-card__career-item"
                        >
                          {item}
                        </li>
                      ))}
                    </ul>
                  </div>
                </article>
              ))}
            </div>
          </section>

          <section id="vision" className="landing-page-vision">
            <div className="landing-page-vision__header">
              <h2 className="landing-page-vision__title">ビジョン</h2>
              <p className="landing-page-vision__lead">
                ブロックチェーントークン経済圏を構築する
              </p>
              <p className="landing-page-vision__description">
                3つのステップを通じて、ブロックチェーン技術を活用した信頼性の高い経済圏を構築します。
                商品の真正性保証から始まり、SNSでのコミュニティ形成、そしてフリマでの安全な二次流通まで、
                ユーザーにとって価値ある体験を提供し続けます。
              </p>
            </div>

            <div className="landing-page-vision__steps">
              {visionSteps.map((item, index) => (
                <div key={item.step} className="landing-page-vision-step-wrap">
                  <article
                    className={[
                      "landing-page-vision-step",
                      item.isCurrent ? "landing-page-vision-step--current" : "",
                    ]
                      .filter(Boolean)
                      .join(" ")}
                  >
                    <div className="landing-page-vision-step__content">
                      <div className="landing-page-vision-step__meta">
                        <span className="landing-page-vision-step__step">
                          {item.step}
                        </span>
                        {item.isCurrent && (
                          <span className="landing-page-vision-step__badge">
                            今ここ
                          </span>
                        )}
                      </div>

                      <h3 className="landing-page-vision-step__title">
                        {item.title}
                      </h3>
                      <p className="landing-page-vision-step__description">
                        {item.description}
                      </p>
                    </div>
                  </article>

                  {index < visionSteps.length - 1 && (
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