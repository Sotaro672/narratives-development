import { Outlet, useLocation } from "react-router-dom";
import Header from "./Header/Header";
import Sidebar from "./Sidebar/Sidebar";
import { Filter, RotateCw } from "lucide-react";
import { useMemo } from "react";

/**
 * 画像のレイアウトから「フレーム要素のみ」を抽出:
 * - 上部ヘッダー
 * - 左サイドバー
 * - 右側のページタイトル行（例: 問い合わせ管理）
 * - 角丸のカード（枠）
 * - カラム見出しのみ（行データは描画しない）
 * - 右上のリフレッシュボタン
 *
 * 行データや詳細UIは各ページが <Outlet /> に差し込む想定。
 */
export default function PageFrame() {
  const location = useLocation();

  // シンプルにパスでページタイトルを決める（必要に応じて拡張）
  const title = useMemo(() => {
    if (location.pathname.startsWith("/inquiries")) return "問い合わせ管理";
    if (location.pathname.startsWith("/listings")) return "商品管理";
    if (location.pathname.startsWith("/mint")) return "トークン管理";
    if (location.pathname.startsWith("/preview")) return "出品管理";
    if (location.pathname.startsWith("/orders")) return "注文管理";
    if (location.pathname.startsWith("/ads")) return "広告管理";
    if (location.pathname.startsWith("/org")) return "組織管理";
    if (location.pathname.startsWith("/accounts")) return "財務管理";
    if (location.pathname.startsWith("/transactions")) return "入出金履歴";
    return "";
  }, [location.pathname]);

  // 問い合わせ画面のカラム枠（画像の通り）
  const inquiryColumns = [
    { label: "問い合わせID", filterable: false },
    { label: "件名", filterable: false },
    { label: "ユーザー", filterable: false },
    { label: "ステータス", filterable: true },
    { label: "タイプ", filterable: false },
    { label: "担当者", filterable: true },
    { label: "問い合わせ日", filterable: false },
    { label: "応答日", filterable: false },
  ];

  // 現在のページが「問い合わせ」なら上のカラム、他ページは空（枠のみ）
  const columns =
    location.pathname.startsWith("/inquiries") ? inquiryColumns : [];

  return (
    <div className="frame">
      <Header />

      <div className="main">
        <aside className="left">
          <Sidebar isOpen />
        </aside>

        <section className="content">
          {/* ページタイトル行 */}
          {title && (
            <div className="content-header">
              <h2 className="content-title">{title}</h2>

              <div className="content-actions">
                <button
                  type="button"
                  className="icon-btn ghost"
                  aria-label="更新"
                  title="更新"
                >
                  <RotateCw className="icon-md" aria-hidden />
                </button>
              </div>
            </div>
          )}

          {/* 角丸カード（枠のみ）。Outlet はボディ内に差し込まれる */}
          <div className="card">
            {/* テーブルヘッダー枠（カラム名のみ。行データは出さない） */}
            {columns.length > 0 && (
              <div className="table">
                <div className="thead">
                  <div className="tr">
                    {columns.map((col) => (
                      <div key={col.label} className="th">
                        <span>{col.label}</span>
                        {col.filterable && (
                          <Filter className="icon-sm th-filter" aria-hidden />
                        )}
                      </div>
                    ))}
                  </div>
                </div>

                {/* 行データは描画しない（枠のみ）。必要ならここに <Outlet /> で注入 */}
                <div className="tbody empty">
                  {/* 各ページが明細を描画したい場合はここに入る */}
                  <Outlet />
                </div>
              </div>
            )}

            {/* カラムを持たないページの場合も、カード枠の中に Outlet を差し込む */}
            {columns.length === 0 && (
              <div className="card-body">
                <Outlet />
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}
