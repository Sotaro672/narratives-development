//frontend\src\pages\AuthActionPage.tsx
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  applyActionCode,
  checkActionCode,
  confirmPasswordReset,
  getAuth,
  verifyPasswordResetCode,
} from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/form.css";

import Layout from "../components/layout/Layout";
import Input from "../components/ui/Input";
import Button from "../components/ui/Button";

export default function AuthActionPage() {
  const navigate = useNavigate();
  const [mode, setMode] = useState<string | null>(null);
  const [oobCode, setOobCode] = useState<string | null>(null);
  const [continueUrl, setContinueUrl] = useState<string | null>(null);
  const [message, setMessage] = useState("処理中です...");
  const [completed, setCompleted] = useState(false);

  const [resetEmail, setResetEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const handleAction = async () => {
      try {
        const auth = getAuth();
        const params = new URLSearchParams(window.location.search);
        const nextMode = params.get("mode");
        const nextOobCode = params.get("oobCode");
        const nextContinueUrl = params.get("continueUrl");

        setMode(nextMode);
        setOobCode(nextOobCode);
        setContinueUrl(nextContinueUrl);

        if (!nextMode || !nextOobCode) {
          setMessage("無効なメールリンクです。");
          setCompleted(true);
          return;
        }

        if (
          nextMode === "verifyEmail" ||
          nextMode === "verifyAndChangeEmail"
        ) {
          await applyActionCode(auth, nextOobCode);
          setMessage("メールアドレスの確認が完了しました。");
          setCompleted(true);

          setTimeout(() => {
            if (nextContinueUrl) {
              window.location.href = nextContinueUrl;
              return;
            }

            navigate("/signin", { replace: true });
          }, 1200);

          return;
        }

        if (nextMode === "recoverEmail") {
          await checkActionCode(auth, nextOobCode);
          await applyActionCode(auth, nextOobCode);
          setMessage("メールアドレスの復元が完了しました。");
          setCompleted(true);

          setTimeout(() => {
            navigate("/signin", { replace: true });
          }, 1200);

          return;
        }

        if (nextMode === "resetPassword") {
          const authEmail = await verifyPasswordResetCode(auth, nextOobCode);
          setResetEmail(authEmail);
          setMessage("新しいパスワードを入力してください。");
          return;
        }

        setMessage(`未対応のメールアクションです。mode=${nextMode}`);
        setCompleted(true);
      } catch (error) {
        console.error(error);
        setMessage("メールアクションの処理に失敗しました。");
        setCompleted(true);
      }
    };

    void handleAction();
  }, [navigate]);

  const handleResetPassword = async () => {
    try {
      if (!oobCode) {
        window.alert("無効なリセットコードです。");
        return;
      }

      if (!newPassword) {
        window.alert("新しいパスワードを入力してください。");
        return;
      }

      if (!confirmPassword) {
        window.alert("確認用パスワードを入力してください。");
        return;
      }

      if (newPassword !== confirmPassword) {
        window.alert("新しいパスワードと確認用パスワードが一致しません。");
        return;
      }

      setSubmitting(true);

      const auth = getAuth();
      await confirmPasswordReset(auth, oobCode, newPassword);

      setMessage("パスワードの再設定が完了しました。");
      setCompleted(true);

      setTimeout(() => {
        if (continueUrl) {
          window.location.href = continueUrl;
          return;
        }

        navigate("/signin", { replace: true });
      }, 1200);
    } catch (error) {
      console.error(error);

      const firebaseError = error as { code?: string };

      switch (firebaseError.code) {
        case "auth/weak-password":
          window.alert(
            "新しいパスワードが弱すぎます。より強いパスワードを設定してください。"
          );
          break;
        case "auth/expired-action-code":
          window.alert(
            "このリンクの有効期限が切れています。もう一度やり直してください。"
          );
          break;
        case "auth/invalid-action-code":
          window.alert("無効なリンクです。もう一度やり直してください。");
          break;
        default:
          window.alert("パスワードの再設定に失敗しました。");
          break;
      }
    } finally {
      setSubmitting(false);
    }
  };

  const isResetPasswordMode = mode === "resetPassword" && !completed;

  return (
    <Layout title="確認" mode="signin">
      <section className="page-section">
        <p className="page-description">{message}</p>

        {isResetPasswordMode && (
          <div className="form-block">
            <Input
              id="reset-email"
              name="resetEmail"
              label="対象メールアドレス"
              type="email"
              value={resetEmail}
              onChange={() => {}}
              disabled
            />

            <Input
              id="new-password"
              name="newPassword"
              label="新しいパスワード"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="新しいパスワードを入力"
              autoComplete="new-password"
              disabled={submitting}
            />

            <Input
              id="confirm-password"
              name="confirmPassword"
              label="新しいパスワード（確認用）"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="確認用パスワードを入力"
              autoComplete="new-password"
              disabled={submitting}
            />

            <div className="page-actions">
              <Button
                variant="primary"
                size="md"
                onClick={handleResetPassword}
                disabled={submitting}
              >
                {submitting ? "更新中..." : "パスワードを更新"}
              </Button>
            </div>
          </div>
        )}

        {completed && !isResetPasswordMode && (
          <div className="page-actions">
            <Button
              variant="primary"
              size="md"
              onClick={() => navigate("/signin")}
            >
              サインインへ
            </Button>
          </div>
        )}
      </section>
    </Layout>
  );
}