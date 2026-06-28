/**
 * Build a signed Solana transfer with @solana/web3.js and submit it to TX Pilot.
 *
 * Usage (from repo root, with server running on :8080):
 *   cd demo/js && npm install && npm run submit
 *
 * Env:
 *   TX_PILOT_TEST_BASE_URL   — default http://localhost:8080
 *   TX_PILOT_KEYPAIR_PATH    — default ../../tx-pilot-test-keypair.json
 */

const fs = require("fs");
const path = require("path");
const {
  Keypair,
  PublicKey,
  SystemProgram,
  Transaction,
  TransactionInstruction,
} = require("@solana/web3.js");

const MEMO_PROGRAM_ID = new PublicKey(
  "MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr"
);

const BASE_URL =
  process.env.TX_PILOT_TEST_BASE_URL ||
  process.env.TX_PILOT_PUBLIC_API_BASE_URL ||
  "http://localhost:8080";

const KEYPAIR_CANDIDATES = [
  process.env.TX_PILOT_KEYPAIR_PATH,
  path.join(__dirname, "../../tx-pilot-test-keypair.json"),
  path.join(process.cwd(), "tx-pilot-test-keypair.json"),
].filter(Boolean);

const TRANSFER_LAMPORTS = 1;
const MEMO = "js-demo-submit";

function resolveKeypairPath() {
  for (const candidate of KEYPAIR_CANDIDATES) {
    const resolved = path.resolve(candidate);
    if (fs.existsSync(resolved)) {
      return resolved;
    }
  }
  throw new Error(
    `keypair not found; run "make test-keypair" from repo root (tried: ${KEYPAIR_CANDIDATES.join(", ")})`
  );
}

function loadKeypair(filePath) {
  const parsed = JSON.parse(fs.readFileSync(filePath, "utf8"));

  if (Array.isArray(parsed)) {
    return Keypair.fromSecretKey(Uint8Array.from(parsed));
  }

  if (typeof parsed === "string") {
    // Go's json.Unmarshal into []byte expects a base64 JSON string.
    const secret = Buffer.from(parsed, "base64");
    if (secret.length === 64) {
      return Keypair.fromSecretKey(secret);
    }
    // Fallback: solana-keygen base58 export.
    const bs58 = require("bs58");
    return Keypair.fromSecretKey(bs58.decode(parsed));
  }

  throw new Error(`unsupported keypair format in ${filePath}`);
}

async function apiGet(pathname) {
  const res = await fetch(`${BASE_URL}${pathname}`);
  const body = await res.text();
  if (!res.ok) {
    throw new Error(`GET ${pathname} ${res.status}: ${body}`);
  }
  return JSON.parse(body);
}

async function apiPost(pathname, payload) {
  const res = await fetch(`${BASE_URL}${pathname}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body = await res.text();
  if (!res.ok) {
    throw new Error(`POST ${pathname} ${res.status}: ${body}`);
  }
  return JSON.parse(body);
}

function memoInstruction(text) {
  return new TransactionInstruction({
    keys: [],
    programId: MEMO_PROGRAM_ID,
    data: Buffer.from(text, "utf8"),
  });
}

async function fetchBlockhash() {
  const resp = await apiGet("/v1/blockhash");
  const blockhash = resp?.value?.blockhash;
  if (!blockhash) {
    throw new Error(`unexpected blockhash response: ${JSON.stringify(resp)}`);
  }
  return blockhash;
}

function buildSignedSelfTransfer(keypair, blockhash, memo) {
  const tx = new Transaction({
    feePayer: keypair.publicKey,
    recentBlockhash: blockhash,
  });

  tx.add(
    SystemProgram.transfer({
      fromPubkey: keypair.publicKey,
      toPubkey: keypair.publicKey,
      lamports: TRANSFER_LAMPORTS,
    })
  );

  if (memo) {
    tx.add(memoInstruction(memo));
  }

  tx.sign(keypair);
  return Buffer.from(tx.serialize()).toString("base64");
}

async function pollTransaction(txId, timeoutMs = 120_000) {
  const deadline = Date.now() + timeoutMs;
  let last = null;

  while (Date.now() < deadline) {
    try {
      last = await apiGet(`/v1/transactions/${txId}`);
    } catch (err) {
      if (!String(err.message).includes("404")) {
        throw err;
      }
      await sleep(500);
      continue;
    }

    const status = last.status;
    process.stdout.write(`  poll status=${status} stage=${last.stage}\n`);
    if (status === "confirmed" || status === "finalized" || status === "failed") {
      return last;
    }
    await sleep(2000);
  }

  throw new Error(`poll timeout for ${txId}; last=${JSON.stringify(last)}`);
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function main() {
  const keypairPath = resolveKeypairPath();
  const keypair = loadKeypair(keypairPath);

  console.log("TX Pilot JS demo — submit signed transaction");
  console.log(`  api:     ${BASE_URL}`);
  console.log(`  keypair: ${keypairPath}`);
  console.log(`  signer:  ${keypair.publicKey.toBase58()}`);

  await apiGet("/healthz");

  const blockhash = await fetchBlockhash();
  console.log(`  blockhash: ${blockhash}`);

  const encoded = buildSignedSelfTransfer(keypair, blockhash, MEMO);
  console.log(`  signed tx (base64, ${encoded.length} chars)`);

  const submit = await apiPost("/v1/transactions", {
    transaction: encoded,
    encoding: "base64",
    memo: MEMO,
  });

  console.log("\nSubmit response:");
  console.log(`  transaction_id:  ${submit.transaction_id}`);
  console.log(`  submission_kind:   ${submit.submission_kind}`);
  console.log(`  result:            ${submit.result}`);
  console.log(`  signature:         ${submit.signature || submit.signatures?.[0] || "(none)"}`);
  console.log(`  tip_lamports:      ${submit.tip_lamports}`);

  console.log("\nPolling lifecycle...");
  const final = await pollTransaction(submit.transaction_id);

  console.log("\nTerminal state:");
  console.log(`  status:    ${final.status}`);
  console.log(`  stage:     ${final.stage}`);
  console.log(`  signature: ${final.signature}`);

  if (final.status === "failed") {
    process.exitCode = 1;
    console.error("\nTransaction failed on-chain or during lifecycle tracking.");
  } else {
    console.log("\nSuccess — transaction landed and reached a healthy terminal state.");
  }
}

main().catch((err) => {
  console.error(err.message || err);
  process.exit(1);
});
