import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Lock, Mail, LogIn } from "lucide-react";

/**
 * SignInPage
 * ------------------------------------------------------------------------
 * Solid State Console 管理者ログインページ。
 * Firebase Auth / GraphQL 認証API / OAuth など任意の認証方式に接続可能。
 * 現時点ではダミーのバリデーション＋ダッシュボード遷移を実装。
 * ------------------------------------------------------------------------
 */
export default function SignInPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      // TODO: 認証APIへ置き換え
      if (email === "" || password === "") {
        throw new Error("メールアドレスとパスワードを入力してください。");
      }

      // ダミー認証処理
      if (email === "admin@example.com" && password === "password") {
        // 認証成功 → ダッシュボードへ
        navigate("/");
      } else {
        throw new Error("メールアドレスまたはパスワードが正しくありません。");
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-slate-100 dark:bg-slate-900">
      <div className="w-full max-w-sm bg-white dark:bg-slate-800 rounded-xl shadow-lg p-8">
        <div className="flex flex-col items-center mb-6">
          <div className="flex items-center gap-2 mb-1">
            <LogIn className="w-6 h-6 text-sky-600" />
            <h1 className="text-lg font-semibold text-slate-800 dark:text-slate-100">
              Solid State Console
            </h1>
          </div>
          <p className="text-slate-500 text-sm dark:text-slate-400">
            管理者ログイン
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="email"
              className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1"
            >
              メールアドレス
            </label>
            <div className="relative">
              <Mail className="absolute left-3 top-2.5 w-4 h-4 text-slate-400" />
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full pl-9 pr-3 py-2 rounded-md border border-slate-300 dark:border-slate-600 
                  bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100
                  focus:ring-2 focus:ring-sky-500 focus:outline-none text-sm"
                placeholder="admin@example.com"
              />
            </div>
          </div>

          <div>
            <label
              htmlFor="password"
              className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1"
            >
              パスワード
            </label>
            <div className="relative">
              <Lock className="absolute left-3 top-2.5 w-4 h-4 text-slate-400" />
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full pl-9 pr-3 py-2 rounded-md border border-slate-300 dark:border-slate-600 
                  bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100
                  focus:ring-2 focus:ring-sky-500 focus:outline-none text-sm"
                placeholder="********"
              />
            </div>
          </div>

          {error && (
            <div className="text-red-500 text-sm mt-2 text-center">{error}</div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full flex items-center justify-center gap-2 bg-sky-600 hover:bg-sky-700
              text-white font-medium py-2 px-4 rounded-md focus:outline-none focus:ring-2
              focus:ring-sky-500 focus:ring-offset-2 transition disabled:opacity-60"
          >
            {loading ? (
              <div className="loading-spinner" />
            ) : (
              <>
                <LogIn className="w-4 h-4" />
                ログイン
              </>
            )}
          </button>
        </form>

        <div className="text-center text-xs text-slate-400 mt-6">
          © {new Date().getFullYear()} Solid State Console
        </div>
      </div>
    </div>
  );
}
