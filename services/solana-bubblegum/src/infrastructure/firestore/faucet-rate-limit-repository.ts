// services/solana-bubblegum/src/infrastructure/firestore/faucet-rate-limit-repository.ts


import crypto from "node:crypto";



import {
  Timestamp,
} from "@google-cloud/firestore";



import type {
  CompleteFaucetRequestInput,
  FaucetRateLimitPort,
  ReserveFaucetSlotResult,
} from "../../application/ports/faucet-rate-limit-port.js";



import {
  firestore,
} from "./firestore-client.js";



const WINDOW_MS =
  8 * 60 * 60 * 1000;



const MAX_REQUESTS_PER_WINDOW =
  2;



type FaucetReservationStatus =
  | "pending"
  | "succeeded"
  | "failed"
  | "rate_limited";



type FaucetReservation = {
  id: string;
  requestedAtMs: number;
  status: FaucetReservationStatus;
  completedAtMs?: number;
  retryAfterSeconds?: number;
};



type FaucetState = {
  requests?: unknown[];
};



function isRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null
  );
}



function normalizeStatus(
  value: unknown,
): FaucetReservationStatus {
  switch (value) {
    case "pending":
    case "succeeded":
    case "failed":
    case "rate_limited":
      return value;


    default:
      return "failed";
  }
}



function normalizeReservation(
  value: unknown,
): FaucetReservation | null {
  if (!isRecord(value)) {
    return null;
  }



  const id =
    value.id;



  const requestedAtMs =
    value.requestedAtMs;



  if (
    typeof id !== "string" ||
    id.length === 0 ||
    typeof requestedAtMs !== "number" ||
    !Number.isFinite(requestedAtMs)
  ) {
    return null;
  }



  const reservation: FaucetReservation = {
    id,
    requestedAtMs,
    status: normalizeStatus(
      value.status,
    ),
  };



  if (
    typeof value.completedAtMs ===
      "number" &&
    Number.isFinite(
      value.completedAtMs,
    )
  ) {
    reservation.completedAtMs =
      value.completedAtMs;
  }



  if (
    typeof value.retryAfterSeconds ===
      "number" &&
    Number.isFinite(
      value.retryAfterSeconds,
    ) &&
    value.retryAfterSeconds >= 0
  ) {
    reservation.retryAfterSeconds =
      value.retryAfterSeconds;
  }



  return reservation;
}



function isCountedReservation(
  reservation: FaucetReservation,
): boolean {
  return (
    reservation.status !==
    "rate_limited"
  );
}



export class FirestoreFaucetRateLimitRepository
  implements FaucetRateLimitPort {
  async reserveRequestSlot(
    now: Date,
  ): Promise<ReserveFaucetSlotResult> {
    const ref =
      firestore
        .collection("system")
        .doc("devnetReserveFaucet");



    return firestore.runTransaction(
      async (transaction) => {
        const snapshot =
          await transaction.get(
            ref,
          );



        const data =
          snapshot.exists
            ? snapshot.data() as FaucetState
            : {};



        const nowMs =
          now.getTime();



        const windowStart =
          nowMs - WINDOW_MS;



        const requests =
          Array.isArray(data.requests)
            ? data.requests
                .map(
                  normalizeReservation,
                )
                .filter(
                  (
                    request,
                  ): request is FaucetReservation =>
                    request !== null,
                )
            : [];



        const recent =
          requests
            .filter(
              (request) =>
                request.requestedAtMs >
                windowStart,
            )
            .sort(
              (a, b) =>
                a.requestedAtMs -
                b.requestedAtMs,
            );



        const active =
          recent.filter(
            isCountedReservation,
          );



        if (
          active.length >=
          MAX_REQUESTS_PER_WINDOW
        ) {
          const oldest =
            active[0];



          return {
            allowed: false,
            nextEligibleAt: new Date(
              oldest.requestedAtMs +
                WINDOW_MS +
                1000,
            ),
          };
        }



        const reservation:
          FaucetReservation = {
            id: crypto.randomUUID(),
            requestedAtMs: nowMs,
            status: "pending",
          };



        transaction.set(
          ref,
          {
            requests: [
              ...recent,
              reservation,
            ],
            updatedAt:
              Timestamp.fromDate(now),
          },
          {
            merge: true,
          },
        );



        return {
          allowed: true,
          reservationId:
            reservation.id,
        };
      },
    );
  }



  async completeRequestSlot(
    input: CompleteFaucetRequestInput,
  ): Promise<void> {
    const ref =
      firestore
        .collection("system")
        .doc("devnetReserveFaucet");



    await firestore.runTransaction(
      async (transaction) => {
        const snapshot =
          await transaction.get(
            ref,
          );



        if (!snapshot.exists) {
          throw new Error(
            `faucet_rate_limit: state not found reservationId=${input.reservationId}`,
          );
        }



        const data =
          snapshot.data() as FaucetState;



        const requests =
          Array.isArray(data.requests)
            ? data.requests
                .map(
                  normalizeReservation,
                )
                .filter(
                  (
                    request,
                  ): request is FaucetReservation =>
                    request !== null,
                )
            : [];



        const reservationIndex =
          requests.findIndex(
            (request) =>
              request.id ===
              input.reservationId,
          );



        if (
          reservationIndex < 0
        ) {
          throw new Error(
            `faucet_rate_limit: reservation not found reservationId=${input.reservationId}`,
          );
        }



        const reservation =
          requests[
            reservationIndex
          ];



        const updated:
          FaucetReservation = {
            ...reservation,
            status:
              input.outcome,
            completedAtMs:
              input.completedAt.getTime(),
          };



        if (
          input.outcome ===
            "rate_limited" &&
          typeof input.retryAfterSeconds ===
            "number" &&
          Number.isFinite(
            input.retryAfterSeconds,
          ) &&
          input.retryAfterSeconds >= 0
        ) {
          updated.retryAfterSeconds =
            Math.ceil(
              input.retryAfterSeconds,
            );
        } else {
          delete updated.retryAfterSeconds;
        }



        requests[
          reservationIndex
        ] = updated;



        const windowStart =
          input.completedAt.getTime() -
          WINDOW_MS;



        const recent =
          requests
            .filter(
              (request) =>
                request.requestedAtMs >
                windowStart,
            )
            .sort(
              (a, b) =>
                a.requestedAtMs -
                b.requestedAtMs,
            );



        transaction.set(
          ref,
          {
            requests: recent,
            updatedAt:
              Timestamp.fromDate(
                input.completedAt,
              ),
          },
          {
            merge: true,
          },
        );
      },
    );
  }
}