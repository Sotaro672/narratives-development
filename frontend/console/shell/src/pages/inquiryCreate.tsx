// frontend/console/shell/src/pages/inquiryCreate.tsx

import { useNavigate } from "react-router-dom";

import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";

export default function InquiryCreate() {
  const navigate = useNavigate();

  return (
    <PageStyle
      layout="single"
      title="AMOLに問い合わせ"
      onBack={() => navigate("/inquiry")}
      onSave={undefined}
    >
      <div />
    </PageStyle>
  );
}