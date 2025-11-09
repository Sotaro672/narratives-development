// frontend/member/src/pages/memberDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { Card, CardHeader, CardTitle, CardContent } from "../../../../shared/ui/card";
import { User, Mail, Calendar } from "lucide-react";
import "../styles/member.css";

export default function MemberDetail() {
  const navigate = useNavigate();
  const { memberId } = useParams<{ memberId: string }>();

  // モックデータ（将来API連携に置き換え）
  const [member] = React.useState({
    name: "小林 静香",
    kana: "コバヤシ シズカ",
    email: "designer.lumina@narratives.com",
    updatedAt: "2024年9月27日",
    joinedAt: "2024年6月25日",
  });

  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle
      layout="grid-2"
      title={`メンバー詳細：${memberId ?? "不明ID"}`}
      onBack={handleBack}
    >
      {/* 左カラム：基本情報カード */}
      <div>
        <Card className="member-card">
          <CardHeader className="member-card__header">
            <User className="member-card__icon" size={18} />
            <CardTitle className="member-card__title">基本情報</CardTitle>
          </CardHeader>

          <CardContent className="member-card__body">
            <div className="member-card__grid">
              <div className="member-card__section">
                <div className="member-card__label">氏名</div>
                <div className="member-card__value">
                  <User size={14} className="icon-inline" />
                  {member.name}
                </div>

                <div className="member-card__label">メールアドレス</div>
                <div className="member-card__value">
                  <Mail size={14} className="icon-inline" />
                  {member.email}
                </div>

                <div className="member-card__label">更新日</div>
                <div className="member-card__value">
                  <Calendar size={14} className="icon-inline" />
                  {member.updatedAt}
                </div>
              </div>

              <div className="member-card__section">
                <div className="member-card__label">読み仮名</div>
                <div className="member-card__value">
                  <User size={14} className="icon-inline" />
                  {member.kana}
                </div>

                <div className="member-card__label">参加日</div>
                <div className="member-card__value">
                  <Calendar size={14} className="icon-inline" />
                  {member.joinedAt}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 右カラム：将来拡張用プレースホルダー */}
      <div></div>
    </PageStyle>
  );
}
