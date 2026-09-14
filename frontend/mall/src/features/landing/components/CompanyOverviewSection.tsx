// frontend/mall/src/features/landing/components/CompanyOverviewSection.tsx

type CompanyOverviewRow = {
  label: string;
  value: string;
};

type CompanyOverviewSectionProps = {
  companyOverviewRows: CompanyOverviewRow[];
};

export default function CompanyOverviewSection({
  companyOverviewRows,
}: CompanyOverviewSectionProps) {
  return (
    <section
      id="company-overview"
      className="landing-page-section landing-page-company-overview"
    >
      <div className="landing-page-section__inner">
        <header className="landing-page-company-overview__header">
          <p className="landing-page-sales-support__eyebrow">
            Company
          </p>

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
  );
}