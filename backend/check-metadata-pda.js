// backend/check-metadata-pda.js
const { Connection, PublicKey, clusterApiUrl } = require("@solana/web3.js");

const MINT_ADDRESS = "81h789tCu7AHyRjBVQHG6LGWSUh6Gb6ZZRRX3ySyYbjc";
const TOKEN_METADATA_PROGRAM_ID = new PublicKey(
  "metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s"
);

(async function main() {
  const mint = new PublicKey(MINT_ADDRESS);

  const [metadataPda] = PublicKey.findProgramAddressSync(
    [Buffer.from("metadata"), TOKEN_METADATA_PROGRAM_ID.toBuffer(), mint.toBuffer()],
    TOKEN_METADATA_PROGRAM_ID
  );

  console.log("mint:", mint.toBase58());
  console.log("metadata PDA:", metadataPda.toBase58());

  const connection = new Connection(clusterApiUrl("devnet"), "confirmed");
  const acc = await connection.getAccountInfo(metadataPda);

  if (!acc) {
    console.log("metadata account: NOT FOUND on devnet");
    process.exitCode = 2;
    return;
  }

  console.log("metadata account: FOUND on devnet");
  console.log("owner(program):", acc.owner.toBase58());
  console.log("lamports:", acc.lamports);
  console.log("data length:", acc.data.length);

  if (acc.owner.equals(TOKEN_METADATA_PROGRAM_ID)) {
    console.log("owner check: OK (Token Metadata Program)");
  } else {
    console.log("owner check: NG (unexpected owner)");
  }
})().catch((e) => {
  console.error("error:", e);
  process.exitCode = 1;
});
