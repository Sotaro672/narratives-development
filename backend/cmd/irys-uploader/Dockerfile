# backend/cmd/irys-uploader/Dockerfile
# Irys uploader 用（TS をビルドして JS を実行）

FROM node:20-alpine AS builder

WORKDIR /app

# ① 必要なファイルだけコピー
#   - backend/cmd/irys-uploader/package.json
#   - backend/cmd/irys-uploader/package-lock.json（あれば）
#   - backend/cmd/irys-uploader/tsconfig.irys.json
#   - backend/cmd/irys-uploader/index.ts
COPY cmd/irys-uploader/package*.json ./ 
COPY cmd/irys-uploader/tsconfig.irys.json ./tsconfig.irys.json
COPY cmd/irys-uploader/index.ts ./index.ts

# ② 依存関係をインストール
RUN npm install

# ③ TypeScript をコンパイルして dist/index.js を作る
RUN npx tsc -p tsconfig.irys.json

# -------------------------------------------------
# 実行用コンテナ（軽量）
# -------------------------------------------------
FROM node:20-alpine

WORKDIR /app

ENV NODE_ENV=production
# Cloud Run が上書きするが、ローカル実行用に 8080 をデフォルトでセット
ENV PORT=8080

# ランタイムに必要なものだけコピー
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/dist ./dist

EXPOSE 8080

# ビルド済み JS を node で起動
CMD ["node", "dist/index.js"]
