# Aegis Lifecycle Log Evidence

Live mainnet Jito submissions captured by `TestBountyLifecycleLog`. Every
signature, bundle id and slot below is reproduced in full (untruncated) so
each transaction can be independently verified on a Solana explorer.

- Generated: 2026-06-27T16:35:34Z
- Submissions: 10 (target 8 success / 2 failure)
- Lifecycle log entries exported: 11

- Observed: 8 success / 2 failed

## Overview

| # | Status | sub slot | proc slot | conf slot | fin slot | Tip (lamports) | Failure |
|---|--------|----------|-----------|-----------|----------|----------------|---------|
| 1 | failed | 429273197 | 0 | 0 | 0 | 19310 | expired_blockhash (Blockhash expired before the bundle landed) |
| 2 | failed | 429273221 | 0 | 0 | 0 | 15000 | expired_blockhash (Blockhash expired before the bundle landed) |
| 3 | finalized | 429273229 | 429273241 | 429273241 | 429273241 | 15000 | - |
| 4 | finalized | 429273224 | 429273265 | 429273265 | 429273265 | 15000 | - |
| 5 | finalized | 429273277 | 429273288 | 429273288 | 429273288 | 60000 | - |
| 6 | finalized | 429273306 | 429273320 | 429273320 | 429273320 | 51317 | - |
| 7 | finalized | 429273302 | 429273341 | 429273341 | 429273341 | 50852 | - |
| 8 | finalized | 429273324 | 429273372 | 429273372 | 429273372 | 180000 | - |
| 9 | finalized | 429273384 | 429273396 | 429273396 | 429273396 | 200000 | - |
| 10 | finalized | 429273407 | 429273420 | 429273420 | 429273420 | 155348 | - |

## Full transaction records

### 1. failed

- transaction_id: `tx_01f6cf4b-7a5d-4c43-a8e4-b3f05731ac80`
- bundle_id: `06f92cde6bd04b5283d69cd052a0e80bf8a1484f3d199dd1c7b1d065c51dbce0`
- signature: `5ZBptHhuvd7QnwgU8gzfrw3wwhoHrNvHDoEHQuwFZkej9bJFLkjiyDrwFv5E8opZwmTyNuvai8ZHfMWZhg3uUn6R`
- explorer: https://explorer.solana.com/tx/5ZBptHhuvd7QnwgU8gzfrw3wwhoHrNvHDoEHQuwFZkej9bJFLkjiyDrwFv5E8opZwmTyNuvai8ZHfMWZhg3uUn6R
- solscan: https://solscan.io/tx/5ZBptHhuvd7QnwgU8gzfrw3wwhoHrNvHDoEHQuwFZkej9bJFLkjiyDrwFv5E8opZwmTyNuvai8ZHfMWZhg3uUn6R
- inject_expired_blockhash: true
- tip_lamports: 19310
- slots: submitted=429273197 processed=0 confirmed=0 finalized=0
- commitment_progression: created -> submitted -> failed
- submitted_at: 2026-06-27T16:33:13Z
- processed_at: -
- confirmed_at: -
- finalized_at: -
- failed_at: 2026-06-27T16:33:16Z
- failure_kind: expired_blockhash
- failure_title: Blockhash expired before the bundle landed

### 2. failed

- transaction_id: `tx_0f8ec26f-971d-4a98-9dbd-b3c8fc820759`
- bundle_id: `76d4db147f764f2fb0885edf878960c85398aff6d417ab9835ea4c83cd34503e`
- signature: `286gN5xXyk7Lyt65Sw6nptjzoU3oYBg4op1jgEKrFA6NAPv2Umuao28GSwgrDydyTGgrzEvY7g8aS9qPyi3vRu69`
- explorer: https://explorer.solana.com/tx/286gN5xXyk7Lyt65Sw6nptjzoU3oYBg4op1jgEKrFA6NAPv2Umuao28GSwgrDydyTGgrzEvY7g8aS9qPyi3vRu69
- solscan: https://solscan.io/tx/286gN5xXyk7Lyt65Sw6nptjzoU3oYBg4op1jgEKrFA6NAPv2Umuao28GSwgrDydyTGgrzEvY7g8aS9qPyi3vRu69
- inject_expired_blockhash: true
- tip_lamports: 15000
- slots: submitted=429273221 processed=0 confirmed=0 finalized=0
- commitment_progression: created -> submitted -> failed
- submitted_at: 2026-06-27T16:33:22Z
- processed_at: -
- confirmed_at: -
- finalized_at: -
- failed_at: 2026-06-27T16:33:25Z
- failure_kind: expired_blockhash
- failure_title: Blockhash expired before the bundle landed

### 3. finalized

- transaction_id: `tx_e3c9705f-063b-4550-b0de-e94f7104c080`
- bundle_id: `67548a0323610e2a092d47d833401875f8ceb5c748fccccb560b88149f1d8528`
- signature: `4VpZyWePkHaXNfsEkgQnf2s28q9Bq3tCfy93LRgUvt9qpXzVzxc22fr9DKyUYjDwNG4RBWP5Xx8LCQKtbFVR3NdY`
- explorer: https://explorer.solana.com/tx/4VpZyWePkHaXNfsEkgQnf2s28q9Bq3tCfy93LRgUvt9qpXzVzxc22fr9DKyUYjDwNG4RBWP5Xx8LCQKtbFVR3NdY
- solscan: https://solscan.io/tx/4VpZyWePkHaXNfsEkgQnf2s28q9Bq3tCfy93LRgUvt9qpXzVzxc22fr9DKyUYjDwNG4RBWP5Xx8LCQKtbFVR3NdY
- inject_expired_blockhash: false
- tip_lamports: 15000
- slots: submitted=429273229 processed=429273241 confirmed=429273241 finalized=429273241
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-27T16:33:32Z
- processed_at: 2026-06-27T16:33:35Z
- confirmed_at: 2026-06-27T16:33:45Z
- finalized_at: 2026-06-27T16:33:50Z
- failed_at: -

### 4. finalized

- transaction_id: `tx_9e7700a4-aaa8-4091-b283-8930311401e1`
- bundle_id: `598cabb039c9884694e5c8ec81041291492cb63cbcf57523c0819dca8538e1ce`
- signature: `3RKQJfTax66jRo1SWZYUyjF4jECcyUFwVezPDPWL8SeuiaYtn5XSGktjP3BiKTtXNJjrFi3zS7WJHkS9BKTTYdHa`
- explorer: https://explorer.solana.com/tx/3RKQJfTax66jRo1SWZYUyjF4jECcyUFwVezPDPWL8SeuiaYtn5XSGktjP3BiKTtXNJjrFi3zS7WJHkS9BKTTYdHa
- solscan: https://solscan.io/tx/3RKQJfTax66jRo1SWZYUyjF4jECcyUFwVezPDPWL8SeuiaYtn5XSGktjP3BiKTtXNJjrFi3zS7WJHkS9BKTTYdHa
- inject_expired_blockhash: false
- tip_lamports: 15000
- slots: submitted=429273224 processed=429273265 confirmed=429273265 finalized=429273265
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-27T16:33:41Z
- processed_at: 2026-06-27T16:33:45Z
- confirmed_at: 2026-06-27T16:33:55Z
- finalized_at: 2026-06-27T16:34:00Z
- failed_at: -

### 5. finalized

- transaction_id: `tx_5904c5c7-d42f-40ac-961a-bf3560e5b60b`
- bundle_id: `38189ff31713ccb04c601ed99ae06a081d3fecdc8f7dc1b2422d589955ea8564`
- signature: `5EHn8DeqXjVeg4DarZ7sWiabP33Z9kkPNBQXh8EtLmwK86vNqrEQCAvNFQhwAq4ZuxFfiuyD1fMwAQCp5hoYHE9n`
- explorer: https://explorer.solana.com/tx/5EHn8DeqXjVeg4DarZ7sWiabP33Z9kkPNBQXh8EtLmwK86vNqrEQCAvNFQhwAq4ZuxFfiuyD1fMwAQCp5hoYHE9n
- solscan: https://solscan.io/tx/5EHn8DeqXjVeg4DarZ7sWiabP33Z9kkPNBQXh8EtLmwK86vNqrEQCAvNFQhwAq4ZuxFfiuyD1fMwAQCp5hoYHE9n
- inject_expired_blockhash: false
- tip_lamports: 60000
- slots: submitted=429273277 processed=429273288 confirmed=429273288 finalized=429273288
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-27T16:33:51Z
- processed_at: 2026-06-27T16:33:55Z
- confirmed_at: 2026-06-27T16:34:00Z
- finalized_at: 2026-06-27T16:34:05Z
- failed_at: -

### 6. finalized

- transaction_id: `tx_15a8eead-544e-48dd-ae26-2279585f3079`
- bundle_id: `a7b3ebd5a8ec611feb9fa55131255e6f57a5e89936196fd0a7124ebffbbd0a0a`
- signature: `3w9UKhSKaGaQq9xP5XGJ67Y657eA7qD6CX5y2ui9u3UcJYtDdEZhdWX4pTkS9NRKaB3QGSpsg5pYErCGSDoPWcNt`
- explorer: https://explorer.solana.com/tx/3w9UKhSKaGaQq9xP5XGJ67Y657eA7qD6CX5y2ui9u3UcJYtDdEZhdWX4pTkS9NRKaB3QGSpsg5pYErCGSDoPWcNt
- solscan: https://solscan.io/tx/3w9UKhSKaGaQq9xP5XGJ67Y657eA7qD6CX5y2ui9u3UcJYtDdEZhdWX4pTkS9NRKaB3QGSpsg5pYErCGSDoPWcNt
- inject_expired_blockhash: false
- tip_lamports: 51317
- slots: submitted=429273306 processed=429273320 confirmed=429273320 finalized=429273320
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-27T16:34:04Z
- processed_at: 2026-06-27T16:34:06Z
- confirmed_at: 2026-06-27T16:34:15Z
- finalized_at: 2026-06-27T16:34:20Z
- failed_at: -

### 7. finalized

- transaction_id: `tx_8f29c1a0-3c57-4a60-823e-bbc160c06d1c`
- bundle_id: `28b95d857a7063c514fa6551bf8bd1cdb421360888d1287a6bbef28073f1736e`
- signature: `Fjxk6BDQeFHH539WpeqkYqYRDFzeziUW4cJVQ8np5jNzXw9GpuCut7hoCz1LE5sbTmnnQqfNtR2wQHWxpiQtXC4`
- explorer: https://explorer.solana.com/tx/Fjxk6BDQeFHH539WpeqkYqYRDFzeziUW4cJVQ8np5jNzXw9GpuCut7hoCz1LE5sbTmnnQqfNtR2wQHWxpiQtXC4
- solscan: https://solscan.io/tx/Fjxk6BDQeFHH539WpeqkYqYRDFzeziUW4cJVQ8np5jNzXw9GpuCut7hoCz1LE5sbTmnnQqfNtR2wQHWxpiQtXC4
- inject_expired_blockhash: false
- tip_lamports: 50852
- slots: submitted=429273302 processed=429273341 confirmed=429273341 finalized=429273341
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-27T16:34:12Z
- processed_at: 2026-06-27T16:34:15Z
- confirmed_at: 2026-06-27T16:34:25Z
- finalized_at: 2026-06-27T16:34:30Z
- failed_at: -

### 8. finalized

- transaction_id: `tx_baf5d052-2264-4145-9358-b3911a59b34e`
- bundle_id: `8504e08dd5441d5b5f6c34e177cb2491ae6dade28ef921c77bcc35368e589b35`
- signature: `5HenCr5d3P6dvP9rTX98V6VpmXAGsELkvGgvCX9tZjWWM73jjsLwugbA2nH6yiThk69YNddxWDBoGxXwy3MXaCwz`
- explorer: https://explorer.solana.com/tx/5HenCr5d3P6dvP9rTX98V6VpmXAGsELkvGgvCX9tZjWWM73jjsLwugbA2nH6yiThk69YNddxWDBoGxXwy3MXaCwz
- solscan: https://solscan.io/tx/5HenCr5d3P6dvP9rTX98V6VpmXAGsELkvGgvCX9tZjWWM73jjsLwugbA2nH6yiThk69YNddxWDBoGxXwy3MXaCwz
- inject_expired_blockhash: false
- tip_lamports: 180000
- slots: submitted=429273324 processed=429273372 confirmed=429273372 finalized=429273372
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-27T16:34:24Z
- processed_at: 2026-06-27T16:34:27Z
- confirmed_at: 2026-06-27T16:34:35Z
- finalized_at: 2026-06-27T16:34:40Z
- failed_at: -

### 9. finalized

- transaction_id: `tx_cb0efca0-1edc-4972-ba80-fe1ddb170c76`
- bundle_id: `f09c1b51f3707d628688327a37440861558ff4dc041d0b62988dea3205b73560`
- signature: `5hddnC5rH8HUhKb2wnppqUSRy44ox5YBw12KaAi369SE22aNPYhoNS46GpEWGiPvoQ7XrRmdHqLLf5PgC14XL93v`
- explorer: https://explorer.solana.com/tx/5hddnC5rH8HUhKb2wnppqUSRy44ox5YBw12KaAi369SE22aNPYhoNS46GpEWGiPvoQ7XrRmdHqLLf5PgC14XL93v
- solscan: https://solscan.io/tx/5hddnC5rH8HUhKb2wnppqUSRy44ox5YBw12KaAi369SE22aNPYhoNS46GpEWGiPvoQ7XrRmdHqLLf5PgC14XL93v
- inject_expired_blockhash: false
- tip_lamports: 200000
- slots: submitted=429273384 processed=429273396 confirmed=429273396 finalized=429273396
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-27T16:34:33Z
- processed_at: 2026-06-27T16:34:36Z
- confirmed_at: 2026-06-27T16:34:45Z
- finalized_at: 2026-06-27T16:34:50Z
- failed_at: -

### 10. finalized

- transaction_id: `tx_94006627-f6a0-4322-9ff6-a2f48acf5573`
- bundle_id: `55d84e0e721db1a3d9d2f14158ca5ff55e33ebc9a310f87e281dd120b8c647eb`
- signature: `66QaXPbouXXUkYaC2JayiWa9kKBz66Xagh6XhhqswfwqcT7HDY449DiWiwtp8tVSWMo6aBQSCYzyDCiGZJcVKjeF`
- explorer: https://explorer.solana.com/tx/66QaXPbouXXUkYaC2JayiWa9kKBz66Xagh6XhhqswfwqcT7HDY449DiWiwtp8tVSWMo6aBQSCYzyDCiGZJcVKjeF
- solscan: https://solscan.io/tx/66QaXPbouXXUkYaC2JayiWa9kKBz66Xagh6XhhqswfwqcT7HDY449DiWiwtp8tVSWMo6aBQSCYzyDCiGZJcVKjeF
- inject_expired_blockhash: false
- tip_lamports: 155348
- slots: submitted=429273407 processed=429273420 confirmed=429273420 finalized=429273420
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-27T16:34:43Z
- processed_at: 2026-06-27T16:34:46Z
- confirmed_at: 2026-06-27T16:34:55Z
- finalized_at: 2026-06-27T16:35:00Z
- failed_at: -

## Explorer verification

Open any `explorer` link above, or paste the full signature into
[Solana Explorer](https://explorer.solana.com/) / [Solscan](https://solscan.io/),
and cross-reference the confirmed/finalized slot recorded here.
