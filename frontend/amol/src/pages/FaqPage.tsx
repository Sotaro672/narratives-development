// frontend/src/pages/FaqPage.tsx
import Layout from "../components/layout/Layout";

import "../styles/page-layout.css";
import "../styles/faq-page.css";

const faqItems = [
  {
    question: "なぜブロックチェーンを使うのですか？",
    answer:
      "商品の取引履歴や所有履歴を改ざんできない形で記録するためです。これにより、商品の真正性と所有権の透明性を高めることができます。",
  },
  {
    question: "真贋証明はどのように確認できますか？",
    answer:
      "商品に付与されたQRコードをスキャンすることで、製品情報、コンテンツ、コメント、所有履歴などにアクセスでき、本物であることを確認できます。",
  },
  {
    question: "ブロックチェーントークンは何に使われますか？",
    answer:
      "商品の真正性や所有履歴を証明するために使われます。また、商品に紐づくコンテンツやストーリーを閲覧したり、所有者同士で交流したりする体験にも活用できます。",
  },
  {
    question: "導入コストは高くありませんか？",
    answer:
      "ブロックチェーントークンの発行コストは1つあたり1円未満に抑えられるため、大量の商品にも現実的に導入できます。",
  },
  {
    question: "転売対策にはどのように役立ちますか？",
    answer:
      "商品の譲渡履歴や所有履歴を確認できるため、不透明な流通を抑制できます。また、購入者同士がコメントを通じてつながることで、健全なコマースコミュニティの形成を促します。",
  },
  {
    question: "口コミの信頼性はどのように担保されますか？",
    answer:
      "トークンを所有しているアカウントだけが商品や関連コンテンツに口コミできる仕組みにすることで、未購入者によるやらせ投稿を防ぎやすくなります。",
  },
  {
    question: "購入者にとってのメリットは何ですか？",
    answer:
      "本物の商品を所有していることを証明できるだけでなく、商品のストーリーやコンテンツを楽しみ、同じ趣味を持つユーザーと交流できる点がメリットです。",
  },
  {
    question: "販売者にとってのメリットは何ですか？",
    answer:
      "販売後も誰が商品を所有しているかを把握しやすくなり、本当に興味のあるユーザーに新商品や関連情報を届けやすくなります。",
  },
];

export default function FaqPage() {
  return (
    <Layout title="FAQ" mode="landing">
      <section className="faq-page">
        <div className="faq-page__inner">
          <div className="faq-page__header">
            <p className="faq-page__eyebrow">FAQ</p>
            <h1 className="faq-page__title">よくある質問</h1>
            <p className="faq-page__lead">
              ブロックチェーンを活用した真贋証明、所有履歴、口コミ基盤について、
              よくある質問をまとめています。
            </p>
          </div>

          <div className="faq-page__list">
            {faqItems.map((item, index) => (
              <article key={item.question} className="faq-page-card">
                <div className="faq-page-card__number">
                  Q{String(index + 1).padStart(2, "0")}
                </div>

                <div className="faq-page-card__body">
                  <h2 className="faq-page-card__question">
                    {item.question}
                  </h2>
                  <p className="faq-page-card__answer">{item.answer}</p>
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>
    </Layout>
  );
}