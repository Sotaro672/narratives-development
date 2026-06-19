// frontend/console/sales/src/presentation/pages/salesDetail.tsx
import { useCallback, useState } from "react";

import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/presentation/components/AdminCard";
import LogCard from "../../../log/presentation/LogCard";
import SalesOwnersCard from "../components/salesOwnersCard";
import InputCard from "../components/inputCard";
import type { SubmitPayload } from "../components/inputCard";

import { useSalesDetail } from "../hook/useSalesDetail";

const initialInputPayload: SubmitPayload = {
  title: "",
  text: "",
  images: [],
};

export default function SalesDetail() {
  const { vm, handlers } = useSalesDetail();
  const [inputPayload, setInputPayload] =
    useState<SubmitPayload>(initialInputPayload);
  const [isSavingInput, setIsSavingInput] = useState(false);
  const [isSendingInput, setIsSendingInput] = useState(false);

  const {
    sales,
    assigneeId,
    assigneeName,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
    owners,
  } = vm;

  const { onBack } = handlers;

  const handleInputChange = useCallback((payload: SubmitPayload) => {
    setInputPayload(payload);
  }, []);

  const buildSubmitPayload = useCallback((): SubmitPayload => {
    return {
      title: inputPayload.title.trim(),
      text: inputPayload.text.trim(),
      images: inputPayload.images,
    };
  }, [inputPayload]);

  const handleSave = useCallback(async () => {
    if (isSavingInput || isSendingInput) return;

    setIsSavingInput(true);

    try {
      const payload = buildSubmitPayload();

      // TODO: 告知の下書き保存 API / usecase と接続する
      console.log("save sales announcement", {
        tokenBlueprintId: sales?.tokenBlueprintId,
        ...payload,
      });
    } finally {
      setIsSavingInput(false);
    }
  }, [buildSubmitPayload, isSavingInput, isSendingInput, sales]);

  const handleSend = useCallback(async () => {
    if (isSavingInput || isSendingInput) return;

    setIsSendingInput(true);

    try {
      const payload = buildSubmitPayload();

      // TODO: 告知の送信 API / usecase と接続する
      console.log("send sales announcement", {
        tokenBlueprintId: sales?.tokenBlueprintId,
        ...payload,
      });
    } finally {
      setIsSendingInput(false);
    }
  }, [buildSubmitPayload, isSavingInput, isSendingInput, sales]);

  if (!sales) {
    return (
      <PageStyle layout="single" title="営業" onBack={onBack}>
        <p className="p-4 text-sm text-muted-foreground">
          表示可能な営業情報がありません。
        </p>
      </PageStyle>
    );
  }

  return (
    <PageStyle
      layout="grid-2"
      title="営業"
      onBack={onBack}
      onSave={handleSave}
      isSaving={isSavingInput}
      onSend={handleSend}
      isSending={isSendingInput}
    >
      <div className="space-y-4">
        <div style={{ marginTop: 16 }}>
          <SalesOwnersCard owners={owners} />
        </div>

        <InputCard
          title="入力"
          saving={isSavingInput}
          sending={isSendingInput}
          onChange={handleInputChange}
        />
      </div>

      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          mode="view"
          assigneeId={assigneeId}
          assigneeName={assigneeName}
          createdByName={createdByName}
          createdAt={createdAt}
          updatedByName={updatedByName}
          updatedAt={updatedAt}
        />

        <LogCard title="更新ログ" />
      </div>
    </PageStyle>
  );
}