//services\solana-bubblegum\src\routes\devnet-reserve-refill.ts
import type {
  Request,
  Response,
} from "express";

import {
  devnetReserveRefillUsecase,
} from "../bootstrap/container.js";

export async function devnetReserveRefillHandler(
  _req: Request,
  res: Response,
): Promise<void> {
  try {
    const result =
      await devnetReserveRefillUsecase
        .execute();

    res.status(200).json({
      data: result,
    });
  } catch (error) {
    console.error(
      "[devnet-reserve-refill]",
      error,
    );

    res.status(500).json({
      error:
        error instanceof Error
          ? error.message
          : "devnet reserve refill failed",
    });
  }
}