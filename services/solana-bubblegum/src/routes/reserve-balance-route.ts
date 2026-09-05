// services/solana-bubblegum/src/routes/reserve-balance-route.ts

import { Router } from "express";

import { getBubblegumRuntime } from "../bootstrap/container.js";
import { env } from "../config/env.js";

const LAMPORTS_PER_SOL = 1_000_000_000;

export const reserveBalanceRouter = Router();

reserveBalanceRouter.get("/reserve-balance", async (_req, res, next) => {
  try {
    const runtime = await getBubblegumRuntime();
    const balance = await runtime.umi.rpc.getBalance(runtime.reserve.publicKey, {
      commitment: "finalized",
    });

    const balanceLamports = balance.basisPoints;

    res.status(200).json({
      cluster: env.solanaCluster,
      address: String(runtime.reserve.publicKey),
      balanceLamports: balanceLamports.toString(),
      balanceSol: Number(balanceLamports) / LAMPORTS_PER_SOL,
    });
  } catch (error) {
    next(error);
  }
});