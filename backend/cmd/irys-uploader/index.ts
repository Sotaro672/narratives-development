// backend/cmd/irys-uploader/index.ts
import "dotenv/config";
import express, { Request, Response } from "express";
import bodyParser from "body-parser";
import { Uploader } from "@irys/upload";
import { Arbitrum } from "@irys/upload-ethereum";
import { SecretManagerServiceClient } from "@google-cloud/secret-manager";

// --------------------------------------------------
// Secret Manager から Irys 用秘密鍵を取得（キャッシュ付き）
// --------------------------------------------------
const secretClient = new SecretManagerServiceClient();

let cachedPrivateKey: string | null = null;

async function loadIrysPrivateKey(): Promise<string> {
  if (cachedPrivateKey) {
    return cachedPrivateKey;
  }

  const raw = process.env.IRYS_PRIVATE_KEY;
  if (!raw) {
    throw new Error("IRYS_PRIVATE_KEY is not set");
  }

  // Secret Manager のリソースパスがそのまま入っている場合:
  //   projects/PROJECT_ID/secrets/SECRET_ID/versions/latest
  if (raw.startsWith("projects/")) {
    console.log("[irys-service] loading private key from Secret Manager:", raw);
    const [version] = await secretClient.accessSecretVersion({ name: raw });
    const data = version.payload?.data?.toString("utf8") ?? "";
    const trimmed = data.trim();
    if (!trimmed) {
      throw new Error("IRYS_PRIVATE_KEY secret payload is empty");
    }
    cachedPrivateKey = trimmed;
  } else {
    // 直接秘密鍵文字列が入っている場合（ローカル開発など）
    console.log("[irys-service] using IRYS_PRIVATE_KEY from env (raw string)");
    cachedPrivateKey = raw.trim();
  }

  return cachedPrivateKey;
}

// --------------------------------------------------
// Irys Uploader シングルトン
// --------------------------------------------------
const getIrysUploader = (() => {
  // 型は素直に any にして細かい型エラーを避ける
  let cached: any | null = null;

  return async () => {
    if (!cached) {
      const pk = await loadIrysPrivateKey();

      const network = process.env.IRYS_NETWORK || "arbitrum";
      const token = process.env.IRYS_TOKEN || "arb";
      console.log(
        `[irys-service] init uploader network=${network}, token=${token}`
      );

      // ★ 型レベルの不整合を any キャストで回避
      //    実行時には問題なく動く想定
      const uploader = await (Uploader as any)(Arbitrum as any).withWallet(pk);

      cached = uploader;
    }
    return cached;
  };
})();

// --------------------------------------------------
// Express アプリ
// --------------------------------------------------
const app = express();

// Cloud Run 互換: PORT は環境変数を優先（未設定なら 8080）
const PORT = Number(process.env.PORT) || 8080;

// 全リクエストをログ（Cloud Run デバッグ用）
app.use((req, _res, next) => {
  console.log(`[irys-service] incoming ${req.method} ${req.url}`);
  next();
});

// JSON ボディを受け取る
app.use(bodyParser.json());

// 任意: Go 側から Authorization: Bearer xxx を付けてくる場合用
const API_KEY = process.env.IRYS_SERVICE_API_KEY || "";

// ルート: 単純なヘルスチェック
app.get("/", (_req: Request, res: Response) => {
  console.log("[irys-service] GET / (root health)");
  res.status(200).send("ok(irys-root)");
});

// Cloud Run / 自前のヘルスチェック用
app.get("/healthz", (_req: Request, res: Response) => {
  console.log("[irys-service] GET /healthz");
  res.status(200).send("ok");
});

// JSON アップロード API
app.post("/upload/json", async (req: Request, res: Response) => {
  try {
    // 認証チェック（API_KEY を設定している場合のみ）
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

    // Go 側の HTTPUploader が期待している形式
    return res.json({ uri });
  } catch (e: any) {
    console.error("[irys-service] upload FAILED:", e);
    return res
      .status(500)
      .json({ error: "upload failed", detail: String(e?.message ?? e) });
  }
});

// 404 ログ（デバッグ用）
app.use((req: Request, res: Response) => {
  console.warn("[irys-service] 404 Not Found:", req.method, req.url);
  res.status(404).send("not found");
});

// サーバ起動
app.listen(PORT, () => {
  console.log(`[irys-service] listening on port ${PORT}`);
});
