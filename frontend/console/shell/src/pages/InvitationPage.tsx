// frontend/shell/src/pages/InvitationPage.tsx
import * as React from "react";
import PageStyle from "../layout/PageStyle/PageStyle";
import { Input } from "../../../shell/src/shared/ui/input";

/**
 * 招待ページ（氏名・かなの入力を担当）
 * - 「姓 → 姓（かな）」「名 → 名（かな）」の順で横並び
 * - PageStyle(single) を使用
 */
export default function InvitationPage() {
  const formRef = React.useRef<HTMLFormElement>(null);

  const [lastName, setLastName] = React.useState("");
  const [lastNameKana, setLastNameKana] = React.useState("");
  const [firstName, setFirstName] = React.useState("");
  const [firstNameKana, setFirstNameKana] = React.useState("");

  const handleBack = () => history.back();
  const handleCreate = () => formRef.current?.requestSubmit();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // ここで API 呼び出し or 次のステップに遷移など
    // e.g., console.log({ lastName, lastNameKana, firstName, firstNameKana });
  };

  return (
    <PageStyle title="メンバー招待" onBack={handleBack} onCreate={handleCreate}>
      <div className="p-4 max-w-3xl mx-auto">
        <form ref={formRef} onSubmit={handleSubmit} className="space-y-4" noValidate>
          {/* 姓 → 姓（かな） */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm text-slate-300 mb-1">姓</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                placeholder="山田"
              />
            </div>
            <div>
              <label className="block text-sm text-slate-300 mb-1">姓（かな）</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={lastNameKana}
                onChange={(e) => setLastNameKana(e.target.value)}
                placeholder="やまだ"
              />
            </div>
          </div>

          {/* 名 → 名（かな） */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm text-slate-300 mb-1">名</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                placeholder="太郎"
              />
            </div>
            <div>
              <label className="block text-sm text-slate-300 mb-1">名（かな）</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={firstNameKana}
                onChange={(e) => setFirstNameKana(e.target.value)}
                placeholder="たろう"
              />
            </div>
          </div>
        </form>
      </div>
    </PageStyle>
  );
}
