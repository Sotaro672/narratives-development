// frontend/console/shell/src/pages/accountConnect.tsx

import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  listBrands,
  type BrandRow,
} from "../features/brand/application/brandService";

import { auth } from "../auth/infrastructure/config/firebaseClient";
import { API_BASE } from "../shared/http/apiBase";
import { fetchJSON } from "../shared/http/fetchJSON";
import { Button } from "../shared/ui/button";
import { Card, CardContent } from "../shared/ui/card";

import "../styles/account.css";

type AccountDTO = {
  id: string;
  companyId: string;
  brandId: string;
  stripeAccountId: string;
  memberId: string;
  bankName: string;
  branchName: string;
  accountNumber: number;
  accountType: string;
  currency: string;
  status: "active" | "inactive" | "suspended" | "deleted";
  createdAt: string;
  createdBy?: string;
  updatedAt: string;
  updatedBy?: string;
};

type ConnectAccountResponse = {
  account: AccountDTO;
  onboardingUrl: string;
  expiresAt: string;
};

const ACCOUNT_CONNECT_URL = `${API_BASE}/accounts/connect`;

function getErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return "Stripe口座との接続に失敗しました。";
}

export default function AccountConnectPage() {
  const navigate = useNavigate();

  const [brands, setBrands] = useState<BrandRow[]>([]);
  const [selectedBrandId, setSelectedBrandId] = useState("");
  const [contactEmail, setContactEmail] = useState(auth.currentUser?.email ?? "");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [completed, setCompleted] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setCompleted(params.get("completed") === "1");
  }, []);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        setLoading(true);
        setError(null);

        const rows = await listBrands();

        if (cancelled) {
          return;
        }

        setBrands(rows);

        const activeRows = rows.filter((brand) => brand.isActive);

        if (activeRows.length === 1) {
          setSelectedBrandId(activeRows[0].id);
        }
      } catch (caughtError: unknown) {
        if (cancelled) {
          return;
        }

        setBrands([]);
        setError(
          caughtError instanceof Error
            ? caughtError.message
            : "ブランド情報の取得に失敗しました。",
        );
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, []);

  const selectedBrand = useMemo(
    () => brands.find((brand) => brand.id === selectedBrandId) ?? null,
    [brands, selectedBrandId],
  );

  const activeBrands = useMemo(
    () => brands.filter((brand) => brand.isActive),
    [brands],
  );

  const handleCreateBrand = () => {
    navigate("/brand/create");
  };

  const handleAccountManagement = () => {
    navigate("/account");
  };

  const handleConnect = async () => {
    if (submitting) {
      return;
    }

    const brandId = selectedBrandId.trim();

    if (!brandId) {
      setError("接続するブランドを選択してください。");
      return;
    }

    const normalizedEmail = contactEmail.trim();

    if (!normalizedEmail) {
      setError("Stripe口座に使用するメールアドレスを入力してください。");
      return;
    }

    try {
      setSubmitting(true);
      setError(null);

      const origin = window.location.origin;

      const response = await fetchJSON<ConnectAccountResponse>(
        ACCOUNT_CONNECT_URL,
        {
          method: "POST",
          mode: "cors",
          auth: "required",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            brandId,
            contactEmail: normalizedEmail,
            country: "JP",
            returnUrl: `${origin}/account/connect?completed=1`,
            refreshUrl: `${origin}/account/connect`,
          }),
        },
      );

      const onboardingUrl = response?.onboardingUrl?.trim();

      if (!onboardingUrl) {
        throw new Error("Stripe onboarding URLを取得できませんでした。");
      }

      window.location.assign(onboardingUrl);
    } catch (caughtError: unknown) {
      setError(getErrorMessage(caughtError));
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="account-connect-page">
        <div className="account-connect-container">
          <Card>
            <CardContent>
              <div className="account-connect-loading">
                ブランド情報を読み込んでいます...
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  if (!error && brands.length === 0) {
    return (
      <div className="account-connect-page">
        <div className="account-connect-container">
          <Card>
            <CardContent>
              <div className="account-connect-no-brand">
                <h1 className="account-connect-title">口座接続</h1>

                <p className="account-connect-no-brand-description">
                  Stripe口座を接続するには、先にブランドを登録してください。
                </p>

                <Button
                  type="button"
                  variant="solid"
                  size="lg"
                  onClick={handleCreateBrand}
                >
                  ブランドを登録する
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="account-connect-page">
      <div className="account-connect-container">
        <Card>
          <CardContent>
            <div className="account-connect-content">
              <h1 className="account-connect-title">口座接続</h1>

              <p className="account-connect-description">
                ブランドの売上受取口座としてStripeを接続します。
                1ブランドにつき1つのStripe口座を接続できます。
              </p>

              {completed && (
                <div className="account-connect-completed">
                  Stripeの口座登録画面から戻りました。
                  接続状態はStripe側の審査・入力状況によって反映まで時間がかかる場合があります。
                </div>
              )}

              {error && (
                <div
                  role="alert"
                  className="account-connect-error"
                >
                  {error}
                </div>
              )}

              {brands.length > 0 && (
                <>
                  <div className="account-connect-field">
                    <label
                      htmlFor="account-connect-brand"
                      className="account-connect-label"
                    >
                      接続するブランド
                    </label>

                    <select
                      id="account-connect-brand"
                      className="account-connect-select"
                      value={selectedBrandId}
                      onChange={(event) => {
                        setSelectedBrandId(event.target.value);

                        if (error) {
                          setError(null);
                        }
                      }}
                      disabled={submitting}
                    >
                      <option value="">
                        ブランドを選択してください
                      </option>

                      {brands.map((brand) => (
                        <option
                          key={brand.id}
                          value={brand.id}
                          disabled={!brand.isActive}
                        >
                          {brand.name}
                          {!brand.isActive ? "（停止中）" : ""}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div className="account-connect-field">
                    <label
                      htmlFor="account-connect-email"
                      className="account-connect-label"
                    >
                      Stripe連絡先メールアドレス
                    </label>

                    <input
                      id="account-connect-email"
                      type="email"
                      className="account-connect-input"
                      value={contactEmail}
                      onChange={(event) => {
                        setContactEmail(event.target.value);

                        if (error) {
                          setError(null);
                        }
                      }}
                      disabled={submitting}
                      required
                    />
                  </div>

                  <div className="account-connect-summary">
                    <div className="account-connect-summary-title">
                      接続対象
                    </div>

                    <div>
                      ブランド：
                      {selectedBrand ? selectedBrand.name : "未選択"}
                    </div>

                    <div>国・地域：日本</div>
                  </div>

                  {activeBrands.length === 0 && (
                    <div className="account-connect-warning">
                      接続可能なアクティブブランドがありません。
                    </div>
                  )}

                  <div className="account-connect-actions">
                    <Button
                      type="button"
                      variant="solid"
                      size="lg"
                      disabled={
                        submitting ||
                        !selectedBrandId ||
                        activeBrands.length === 0
                      }
                      onClick={() => {
                        void handleConnect();
                      }}
                    >
                      {submitting
                        ? "Stripeへ接続中..."
                        : "Stripeと接続する"}
                    </Button>

                    <Button
                      type="button"
                      variant="outline"
                      size="lg"
                      disabled={submitting}
                      onClick={handleAccountManagement}
                    >
                      口座管理へ
                    </Button>
                  </div>
                </>
              )}

              {error && brands.length === 0 && (
                <div className="account-connect-actions">
                  <Button
                    type="button"
                    variant="outline"
                    size="lg"
                    onClick={() => {
                      window.location.reload();
                    }}
                  >
                    再読み込み
                  </Button>

                  <Button
                    type="button"
                    variant="solid"
                    size="lg"
                    onClick={handleCreateBrand}
                  >
                    ブランド管理へ
                  </Button>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}