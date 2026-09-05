// services/solana-bubblegum/src/app.ts

import express from "express";

import { errorHandler } from "./http/error-handler.js";
import { estimateRouter } from "./routes/estimate-route.js";
import { healthRouter } from "./routes/health-route.js";
import { mintRouter } from "./routes/mint-route.js";
import { ownedAssetsRouter } from "./routes/owned-assets-route.js";
import { reserveBalanceRouter } from "./routes/reserve-balance-route.js";
import { transferRouter } from "./routes/transfer-route.js";

export const app = express();

app.disable("x-powered-by");
app.use(express.json({ limit: "32kb" }));

app.use(healthRouter);
app.use(ownedAssetsRouter);
app.use(transferRouter);
app.use(estimateRouter);
app.use(mintRouter);
app.use(reserveBalanceRouter);

app.use((_req, res) => {
  res.status(404).json({ error: "not found" });
});

app.use(errorHandler);