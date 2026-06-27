/**
 * Build signed Solana transactions with @solana/web3.js and submit a bundle to Aegis.
 *
 * Mirrors test/developer/submit_bundle_test.go: a 1-lamport self-transfer plus a
 * client-signed Jito tip transaction, POSTed to POST /v1/bundles.
 *
 * Usage (from repo root, with server running on :8080):
 *   cd demo/js && npm install && npm run submit:bundle
 *
 * Env:
 *   AEGIS_TEST_BASE_URL   — default http://localhost:8080
 *   AEGIS_KEYPAIR_PATH    — default ../../aegis-test-keypair.json
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
  process.env.AEGIS_TEST_BASE_URL ||
  process.env.AEGIS_PUBLIC_API_BASE_URL ||
  "http://localhost:8080";

const KEYPAIR_CANDIDATES = [
  process.env.AEGIS_KEYPAIR_PATH,
  path.join(__dirname, "../../aegis-test-keypair.json"),
  path.join(process.cwd(), "aegis-test-keypair.json"),
].filter(Boolean);

const TRANSFER_LAMPORTS = 1;
const TIP_LAMPORTS = 1000;
const MEMO = "js-demo-bundle";

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
    const secret = Buffer.from(parsed, "base64");
    if (secret.length === 64) {
      return Keypair.fromSecretKey(secret);
    }
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

async function fetchTipAccount() {
  const resp = await apiGet("/v1/tip-accounts");
  const account = resp?.accounts?.[0];
  if (!account) {
    throw new Error(`no tip accounts returned: ${JSON.stringify(resp)}`);
  }
  return new PublicKey(account);
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

function buildSignedTip(keypair, tipAccount, lamports, blockhash) {
  const tx = new Transaction({
    feePayer: keypair.publicKey,
    recentBlockhash: blockhash,
  });

  tx.add(
    SystemProgram.transfer({
      fromPubkey: keypair.publicKey,
      toPubkey: tipAccount,
      lamports,
    })
  );

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

  console.log("Aegis JS demo — submit signed bundle");
  console.log(`  api:     ${BASE_URL}`);
  console.log(`  keypair: ${keypairPath}`);
  console.log(`  signer:  ${keypair.publicKey.toBase58()}`);

  await apiGet("/healthz");

  const blockhash = await fetchBlockhash();
  const tipAccount = await fetchTipAccount();
  console.log(`  blockhash:   ${blockhash}`);
  console.log(`  tip account: ${tipAccount.toBase58()}`);

  const transferTx = buildSignedSelfTransfer(
    keypair,
    blockhash,
    `${MEMO}-transfer`
  );
  const tipTx = buildSignedTip(keypair, tipAccount, TIP_LAMPORTS, blockhash);
  console.log(`  transfer tx (base64, ${transferTx.length} chars)`);
  console.log(`  tip tx     (base64, ${tipTx.length} chars, ${TIP_LAMPORTS} lamports)`);

  const submit = await apiPost("/v1/bundles", {
    transactions: [transferTx, tipTx],
    encoding: "base64",
    memo: MEMO,
  });

  console.log("\nSubmit response:");
  console.log(`  transaction_id:  ${submit.transaction_id}`);
  console.log(`  submission_kind: ${submit.submission_kind}`);
  console.log(`  bundle_id:       ${submit.bundle_id || submit.result}`);
  console.log(`  signatures:      ${(submit.signatures || []).join(", ") || submit.signature}`);

  const bundleId = submit.bundle_id || submit.result;
  if (bundleId) {
    const bundleStatus = await apiGet(`/v1/bundles/${bundleId}`);
    console.log("\nBundle status:");
    console.log(`  bundle.status: ${bundleStatus?.bundle?.status ?? "(unknown)"}`);
    if (bundleStatus?.jito_status) {
      console.log(`  jito_status:   ${JSON.stringify(bundleStatus.jito_status)}`);
    }
  }

  console.log("\nPolling lifecycle...");
  const final = await pollTransaction(submit.transaction_id);

  console.log("\nTerminal state:");
  console.log(`  status:    ${final.status}`);
  console.log(`  stage:     ${final.stage}`);
  console.log(`  signature: ${final.signature}`);

  if (final.status === "failed") {
    process.exitCode = 1;
    console.error("\nBundle failed on-chain or during lifecycle tracking.");
  } else {
    console.log("\nSuccess — bundle landed and reached a healthy terminal state.");
  }
}

main().catch((err) => {
  console.error(err.message || err);
  process.exit(1);
});
