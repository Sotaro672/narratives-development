// frontend/shell/src/layout/PageFrame.tsx
import { Outlet, useLocation } from "react-router-dom";
import Header from "./Header/Header";
import Sidebar from "./Sidebar/Sidebar";
import Main from "./Main/Main";
import { Filter, RotateCw } from "lucide-react";
import { useMemo } from "react";

/**
 * フレーム要素のみ（データは各ページが <Outlet /> で差し込む）
 * - 上部ヘッダー
 * - 左サイドバー
 * - 右側のページタイトル行
 * - 角丸カード（枠）
 * - 問い合わせページのみカラム見出し
 * - 右上リフレッシュボタン
 */
export default function PageFrame() {
  const location = useLocation();
  const p = location.pathname;

  // Sidebar.tsx の定義に完全準拠（全て単数形に統一）
  const title = useMemo(() => {
    if (p.startsWith("/inquiry")) return "問い合わせ管理";

    // 商品グループ: /product, /productBlueprint, /production, /inventory
    if (
      p.startsWith("/product") ||
      p.startsWith("/productBlueprint") ||
      p.startsWith("/production") ||
      p.startsWith("/inventory")
    ) {
      return "商品管理";
    }

    // トークングループ: /token, /tokenBlueprint, /mint, /operations
    if (
      p.startsWith("/token") ||
      p.startsWith("/tokenBlueprint") ||
      p.startsWith("/mint") ||
      p.startsWith("/operation")
    ) {
      return "トークン管理";
    }

    // 出品: /list
    if (p.startsWith("/list")) return "出品管理";

    // 注文: /order
    if (p.startsWith("/order")) return "注文管理";

    // 広告: /ads
    if (p.startsWith("/ads")) return "広告管理";

    // 組織グループ: /company, /member, /brand, /permission
    if (
      p.startsWith("/company") ||
      p.startsWith("/member") ||
      p.startsWith("/brand") ||
      p.startsWith("/permission")
    ) {
      return "組織管理";
    }

    // 財務: /finance（トップ）
    if (p.startsWith("/finance")) return "財務管理";

    // 財務サブ: /transaction, /account
    if (p.startsWith("/transaction")) return "入出金履歴";
    if (p.startsWith("/account")) return "口座";

    return "";
  }, [p]);

  // 問い合わせ画面のカラム枠（行データは描画しない）
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

  const showInquiryTableHead = p.startsWith("/inquiry");

  return (
    <div className="frame">
      <Header />

      <div className="main">
        <aside className="left">
          <Sidebar isOpen />
        </aside>

        <Main>
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

          {/* 角丸カード（枠）。Outlet はボディ内へ差し込み */}
          <div className="card">
            {showInquiryTableHead ? (
              <div className="table">
                <div className="thead">
                  <div className="tr">
                    {inquiryColumns.map((col) => (
                      <div key={col.label} className="th">
                        <span>{col.label}</span>
                        {col.filterable && (
                          <Filter className="icon-sm th-filter" aria-hidden />
                        )}
                      </div>
                    ))}
                  </div>
                </div>

                {/* 明細は描画しない。必要時は各ページが Outlet で挿入 */}
                <div className="tbody empty">
                  <Outlet />
                </div>
              </div>
            ) : (
              <div className="card-body">
                <Outlet />
              </div>
            )}
          </div>
        </Main>
      </div>
    </div>
  );
}
