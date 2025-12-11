// backend/cmd/irys-uploader/index.ts
import "dotenv/config";
import express, { type Request, type Response } from "express";
import bodyParser from "body-parser";
import { Uploader } from "@irys/upload";
import { Arbitrum } from "@irys/upload-ethereum"; // Ethereum でもOKだが安いL2推奨

// Irys Uploader をシングルトン的に初期化
const getIrysUploader = (() => {
  // ※ 型エラー回避のため any でキャッシュ（実際の型は Irys の Upload クライアント）
  let cached: any | null = null;

  return async () => {
    if (!cached) {
      const pk = process.env.IRYS_PRIVATE_KEY;
      if (!pk) {
        throw new Error("IRYS_PRIVATE_KEY is not set");
      }

      console.log("[irys-service] initializing Irys uploader (Arbitrum)...");
      // withWallet は Promise を返すので await する
      cached = await Uploader(Arbitrum).withWallet(pk);
      console.log(
        "[irys-service] Irys uploader initialized. token=",
        // token プロパティがあればログする（なければ undefined になるだけ）
        (cached as any).token
      );
    }
    return cached;
  };
})();

const app = express();
const PORT = process.env.PORT || 3001;

// JSON ボディを受け取る
app.use(bodyParser.json());

// 認証（任意）：Go 側から Authorization: Bearer xxx を付けてくる場合
const API_KEY = process.env.IRYS_SERVICE_API_KEY || "";

app.post("/upload/json", async (req: Request, res: Response) => {
  try {
    if (API_KEY) {
      const auth = req.header("Authorization") || "";
      const expected = `Bearer ${API_KEY}`;
      if (auth !== expected) {
        console.warn("[irys-service] unauthorized request, auth header:", auth);
        return res.status(401).json({ error: "unauthorized" });
      }
    }

    const jsonBody = req.body;
    console.log("[irys-service] /upload/json called, body:", jsonBody);

    const dataToUpload = JSON.stringify(jsonBody);

    const irys = await getIrysUploader();

    console.log("[irys-service] uploading to Irys...");
    const receipt = await irys.upload(dataToUpload, {
      tags: [
        { name: "Content-Type", value: "application/json" },
        { name: "app", value: "Narratives" },
      ],
    });

    const uri = `https://gateway.irys.xyz/${receipt.id}`;
    console.log("[irys-service] upload OK id=", receipt.id, "uri=", uri);

    // ★ Go の HTTPUploader が期待しているレスポンス形式
    return res.json({ uri });
  } catch (e: any) {
    console.error("[irys-service] upload FAILED:", e);
    return res.status(500).json({ error: "upload failed", detail: String(e) });
  }
});

app.get("/healthz", (_req: Request, res: Response) => {
  res.status(200).send("ok");
});

app.listen(PORT, () => {
  console.log(`[irys-service] listening on port ${PORT}`);
});
