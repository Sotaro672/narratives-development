import { useNavigate } from "react-router-dom";
import { AlertTriangle, ArrowLeft } from "lucide-react";

/**
 * NotFoundPage
 * ------------------------------------------------------------------------
 * Solid State Console 全体で共通使用する404ページ。
 * 存在しないルートや無効なURLアクセス時に表示される。
 * ------------------------------------------------------------------------
 */
export default function NotFoundPage() {
  const navigate = useNavigate();

  return (
    <div className="flex flex-col items-center justify-center min-h-[80vh] text-center">
      <div className="flex flex-col items-center mb-8">
        <AlertTriangle className="w-12 h-12 text-amber-500 mb-3" />
        <h1 className="text-3xl font-semibold text-slate-800 dark:text-slate-100 mb-2">
          ページが見つかりません
        </h1>
        <p className="text-slate-600 dark:text-slate-300 max-w-md">
          アクセスしようとしたページは存在しないか、移動した可能性があります。
        </p>
      </div>

      <button
        onClick={() => navigate("/")}
        className="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-sky-600 text-white hover:bg-sky-700 transition-colors"
      >
        <ArrowLeft className="w-4 h-4" />
        ダッシュボードに戻る
      </button>

      <div className="mt-12 text-sm text-slate-400">
        Solid State Console © {new Date().getFullYear()}
      </div>
    </div>
  );
}
