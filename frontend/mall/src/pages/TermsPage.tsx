// frontend/src/pages/TermsPage.tsx
import { useEffect, useState } from "react";

import "../styles/landing-page.css";
import "../styles/terms-page.css";

import Layout from "../components/layout/Layout";

type LegalDocument = {
  title: string;
  eyebrow: string;
  path: string;
};

const legalDocuments: LegalDocument[] = [
  {
    title: "利用規約",
    eyebrow: "Terms",
    path: "/assets/terms.txt",
  },
  {
    title: "プライバシーポリシー",
    eyebrow: "Privacy Policy",
    path: "/assets/privacy-policy.txt",
  },
  {
    title: "特定商取引法に基づく表記",
    eyebrow: "Specified Commercial Transactions",
    path: "/assets/specified-commercial-transactions.txt",
  },
];

type LoadedLegalDocument = LegalDocument & {
  content: string;
};

function LegalDocumentCard({ document }: { document: LoadedLegalDocument }) {
  return (
    <section className="landing-page-card terms-page">
      <header className="terms-page__section">
        <p className="how-to-use-page__eyebrow">{document.eyebrow}</p>
        <h2 className="terms-page__heading">{document.title}</h2>
      </header>

      <pre className="terms-page__content">{document.content}</pre>
    </section>
  );
}

export default function TermsPage() {
  const [documents, setDocuments] = useState<LoadedLegalDocument[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function loadDocuments() {
      try {
        setLoading(true);
        setErrorMessage("");

        const loadedDocuments = await Promise.all(
          legalDocuments.map(async (document) => {
            const response = await fetch(document.path);

            if (!response.ok) {
              throw new Error(`${document.title}の読み込みに失敗しました。`);
            }

            const content = await response.text();

            return {
              ...document,
              content,
            };
          })
        );

        if (!cancelled) {
          setDocuments(loadedDocuments);
        }
      } catch (error) {
        if (!cancelled) {
          const message =
            error instanceof Error
              ? error.message
              : "文書の読み込みに失敗しました。";

          setErrorMessage(message);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadDocuments();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <Layout title="AMOL" mode="landing">
      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <header className="how-to-use-page__header">
            <p className="how-to-use-page__eyebrow">Legal</p>
            <h1 className="how-to-use-page__title">
              利用規約・プライバシーポリシー・特定商取引法に基づく表記
            </h1>
          </header>

          {loading ? (
            <div className="landing-page-card terms-page">
              <p className="landing-page-card__text">読み込み中です。</p>
            </div>
          ) : null}

          {!loading && errorMessage ? (
            <div className="landing-page-card terms-page">
              <p className="landing-page-card__text">{errorMessage}</p>
            </div>
          ) : null}

          {!loading && !errorMessage
            ? documents.map((document) => (
                <LegalDocumentCard key={document.path} document={document} />
              ))
            : null}
        </div>
      </section>
    </Layout>
  );
}