// frontend/amol/src/pages/VisionPage.tsx
import "../styles/page-layout.css";
import "../styles/vision-page.css";

import Layout from "../components/layout/Layout";

const team = [
  {
    name: "奥岡 曹太朗",
    role: "開発者・代表",
    bio: "生産者の信頼を技術で守る",
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
    title: "電子名札が付与された商品のみが流通するMall",
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
    title: "電子名札付商品を交換できるフリマ",
    description:
      "ユーザー間で電子名札付商品を交換できるフリマ。本物であることが証明された商品を出品できるため、再販価格が保障されます。上代価格と再販価格の差額が実質的な負担額となります。",
    isCurrent: false,
  },
];

export default function VisionPage() {
  return (
    <Layout title="AMOL" mode="landing">
      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <section id="team" className="landing-page-team">
            <div className="landing-page-team__header">
              <h2 className="landing-page-team__title">代表者</h2>
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
                生産者の信頼を国境を越えて守られる経済圏を構築します。
              </p>
              <p className="landing-page-vision__description">
                自分の作成した製品がどこで誰の役に立っているのか、誰を豊かにしているのかを国境を越えて追跡できる経済圏を構築します。
                偽物かもしれないという不当な疑義をかけられることなく全ての生産者が正当に評価し、競い合える舞台を３つのステップを通して構築します。
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