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
        </div>
      </section>
    </Layout>
  );
}