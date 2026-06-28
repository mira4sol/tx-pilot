# TX Pilot Lifecycle Log Evidence

Live mainnet Jito submissions captured by `TestBountyLifecycleLog`. Every
signature, bundle id and slot below is reproduced in full (untruncated) so
each transaction can be independently verified on a Solana explorer.

- Generated: 2026-06-28T00:40:24Z
- Submissions: 10 (target 8 success / 2 failure)
- Lifecycle log entries exported: 21

- Observed: 8 success / 2 failed

## Overview

| # | Status | sub slot | proc slot | conf slot | fin slot | Tip (lamports) | Failure |
|---|--------|----------|-----------|-----------|----------|----------------|---------|
| 1 | failed | 429345640 | 0 | 0 | 0 | 594758 | expired_blockhash (Blockhash expired before the bundle landed) |
| 2 | failed | 429345666 | 0 | 0 | 0 | 650000 | expired_blockhash (Blockhash expired before the bundle landed) |
| 3 | finalized | 429345672 | 429345681 | 429345681 | 429345681 | 650000 | - |
| 4 | finalized | 429345700 | 429345709 | 429345709 | 429345709 | 600782 | - |
| 5 | finalized | 429345696 | 429345739 | 429345739 | 429345739 | 133365 | - |
| 6 | finalized | 429345719 | 429345760 | 429345760 | 429345760 | 133365 | - |
| 7 | finalized | 429345747 | 429345787 | 0 | 429345787 | 80000 | - |
| 8 | finalized | 429345843 | 429345854 | 429345854 | 429345854 | 80000 | - |
| 9 | finalized | 429345866 | 429345875 | 429345875 | 429345875 | 120157 | - |
| 10 | finalized | 429345893 | 429345904 | 429345904 | 429345904 | 126165 | - |

## Full transaction records

### 1. failed

- transaction_id: `tx_a166eb41-3c2b-4ff1-91bb-65a8350dcc4a`
- bundle_id: `63df60cb365f4e941cdfcdb99426c71bcdc308918f8af56a331684124a434285`
- signature: `4kMNWWrMsbadCCvKGS8wZz2xBeYwVA34Y6uUpCSHzsbaHqDnin1XGHtEN9rTZC9JJQzNqMFU66s7Gg8S8Q96LmKw`
- explorer: https://explorer.solana.com/tx/4kMNWWrMsbadCCvKGS8wZz2xBeYwVA34Y6uUpCSHzsbaHqDnin1XGHtEN9rTZC9JJQzNqMFU66s7Gg8S8Q96LmKw
- solscan: https://solscan.io/tx/4kMNWWrMsbadCCvKGS8wZz2xBeYwVA34Y6uUpCSHzsbaHqDnin1XGHtEN9rTZC9JJQzNqMFU66s7Gg8S8Q96LmKw
- inject_expired_blockhash: true
- tip_lamports: 594758
- slots: submitted=429345640 processed=0 confirmed=0 finalized=0
- commitment_progression: created -> submitted -> failed
- submitted_at: 2026-06-28T00:37:46Z
- processed_at: -
- confirmed_at: -
- finalized_at: -
- failed_at: 2026-06-28T00:37:49Z
- failure_kind: expired_blockhash
- failure_title: Blockhash expired before the bundle landed

### 2. failed

- transaction_id: `tx_6631d9d1-bdca-4830-a684-7ed3c6603ae9`
- bundle_id: `0f3704af0989829bf898c96b5d2a54171f2022a67903b87dc68d1c74b1cd2cb6`
- signature: `3GarTyd45UeQ63a2r4BLrexC3bFdiQ24p9tcJDTBKRYUPridW4meUtWXCKFd6Ciqhq75HupFK5LWDMM9pF7SUtnZ`
- explorer: https://explorer.solana.com/tx/3GarTyd45UeQ63a2r4BLrexC3bFdiQ24p9tcJDTBKRYUPridW4meUtWXCKFd6Ciqhq75HupFK5LWDMM9pF7SUtnZ
- solscan: https://solscan.io/tx/3GarTyd45UeQ63a2r4BLrexC3bFdiQ24p9tcJDTBKRYUPridW4meUtWXCKFd6Ciqhq75HupFK5LWDMM9pF7SUtnZ
- inject_expired_blockhash: true
- tip_lamports: 650000
- slots: submitted=429345666 processed=0 confirmed=0 finalized=0
- commitment_progression: created -> submitted -> failed
- submitted_at: 2026-06-28T00:37:55Z
- processed_at: -
- confirmed_at: -
- finalized_at: -
- failed_at: 2026-06-28T00:37:59Z
- failure_kind: expired_blockhash
- failure_title: Blockhash expired before the bundle landed

### 3. finalized

- transaction_id: `tx_85b57a8b-aa8c-4ff2-ab57-035be1b9fcfc`
- bundle_id: `4012355dca326ba41bf13c584bf238214f54733a82e45b47ace5c48dd70be5a1`
- signature: `4vTxfp83jL6cgQzL5jaJkUjj4opCaymRa9dBk1rKsCnLkNEYRoyCKjLjJ7Y1DzM478fM9wMwH1RCiiEh435oiRyr`
- explorer: https://explorer.solana.com/tx/4vTxfp83jL6cgQzL5jaJkUjj4opCaymRa9dBk1rKsCnLkNEYRoyCKjLjJ7Y1DzM478fM9wMwH1RCiiEh435oiRyr
- solscan: https://solscan.io/tx/4vTxfp83jL6cgQzL5jaJkUjj4opCaymRa9dBk1rKsCnLkNEYRoyCKjLjJ7Y1DzM478fM9wMwH1RCiiEh435oiRyr
- inject_expired_blockhash: false
- tip_lamports: 650000
- slots: submitted=429345672 processed=429345681 confirmed=429345681 finalized=429345681
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-28T00:38:04Z
- processed_at: 2026-06-28T00:38:09Z
- confirmed_at: 2026-06-28T00:38:14Z
- finalized_at: 2026-06-28T00:38:19Z
- failed_at: -

### 4. finalized

- transaction_id: `tx_40e35906-80de-4d38-8d14-eb10d1fb5e28`
- bundle_id: `662785d7892e921c4e938ee87b08e893c5f843884a1a93bbdb9ca1801e17a164`
- signature: `493gYgxGnmnDV7s1ApekFBYVHVHpvDukWnq5qX3yuvowYrd6fQWb1gxdPZhyoHg8EtxqQGQCB3We42s6vppaK1aL`
- explorer: https://explorer.solana.com/tx/493gYgxGnmnDV7s1ApekFBYVHVHpvDukWnq5qX3yuvowYrd6fQWb1gxdPZhyoHg8EtxqQGQCB3We42s6vppaK1aL
- solscan: https://solscan.io/tx/493gYgxGnmnDV7s1ApekFBYVHVHpvDukWnq5qX3yuvowYrd6fQWb1gxdPZhyoHg8EtxqQGQCB3We42s6vppaK1aL
- inject_expired_blockhash: false
- tip_lamports: 600782
- slots: submitted=429345700 processed=429345709 confirmed=429345709 finalized=429345709
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-28T00:38:14Z
- processed_at: 2026-06-28T00:38:19Z
- confirmed_at: 2026-06-28T00:38:24Z
- finalized_at: 2026-06-28T00:38:29Z
- failed_at: -

### 5. finalized

- transaction_id: `tx_d5f4e96c-6ebf-4598-9b35-d8131bc2b5ba`
- bundle_id: `abb83aefedc53ce7f2cb732a16c2c3138357c276cf14e6de12a72222847bab6a`
- signature: `5U6rPJzhLTuBqg4prz2RSi32aGg2GhgkBY4yNnKqrTF6fMYSZWRTr4MVHDTPpEpTw3EjPGwuSuyFRrnyoXKV4nD3`
- explorer: https://explorer.solana.com/tx/5U6rPJzhLTuBqg4prz2RSi32aGg2GhgkBY4yNnKqrTF6fMYSZWRTr4MVHDTPpEpTw3EjPGwuSuyFRrnyoXKV4nD3
- solscan: https://solscan.io/tx/5U6rPJzhLTuBqg4prz2RSi32aGg2GhgkBY4yNnKqrTF6fMYSZWRTr4MVHDTPpEpTw3EjPGwuSuyFRrnyoXKV4nD3
- inject_expired_blockhash: false
- tip_lamports: 133365
- slots: submitted=429345696 processed=429345739 confirmed=429345739 finalized=429345739
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-28T00:38:25Z
- processed_at: 2026-06-28T00:38:29Z
- confirmed_at: 2026-06-28T00:38:39Z
- finalized_at: 2026-06-28T00:38:44Z
- failed_at: -

### 6. finalized

- transaction_id: `tx_a54f9938-3d0f-41ff-afc2-47c548e0c7de`
- bundle_id: `d2dc56682915c4e28ca5ac780f776275b2c64ade34d2be91b5f75b235f135c66`
- signature: `4cipEuzcYECUfGWsmBFA9t7UB2r8FGNpyPX1Z8ef7AU3b5tQjc4dYCPY1REzxNgJof6PZU4dPoiUEjuxZLDM3TMB`
- explorer: https://explorer.solana.com/tx/4cipEuzcYECUfGWsmBFA9t7UB2r8FGNpyPX1Z8ef7AU3b5tQjc4dYCPY1REzxNgJof6PZU4dPoiUEjuxZLDM3TMB
- solscan: https://solscan.io/tx/4cipEuzcYECUfGWsmBFA9t7UB2r8FGNpyPX1Z8ef7AU3b5tQjc4dYCPY1REzxNgJof6PZU4dPoiUEjuxZLDM3TMB
- inject_expired_blockhash: false
- tip_lamports: 133365
- slots: submitted=429345719 processed=429345760 confirmed=429345760 finalized=429345760
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-28T00:38:34Z
- processed_at: 2026-06-28T00:38:39Z
- confirmed_at: 2026-06-28T00:38:44Z
- finalized_at: 2026-06-28T00:39:07Z
- failed_at: -

### 7. finalized

- transaction_id: `tx_0279c9e0-7add-41ee-8fcd-c674b5ccff63`
- bundle_id: `a220319e12472d6f24a4d0e7b8e15b450ff2512bdb6acce1e0cbb61322279fe0`
- signature: `26MngwPjvKmuC84zeSm5ZCbuMQFKdypMafuXMKJUNUhP9kEvbF5id1aQkwXLpMNx3Z7KhufR9TmY3u3aFVrCiAcV`
- explorer: https://explorer.solana.com/tx/26MngwPjvKmuC84zeSm5ZCbuMQFKdypMafuXMKJUNUhP9kEvbF5id1aQkwXLpMNx3Z7KhufR9TmY3u3aFVrCiAcV
- solscan: https://solscan.io/tx/26MngwPjvKmuC84zeSm5ZCbuMQFKdypMafuXMKJUNUhP9kEvbF5id1aQkwXLpMNx3Z7KhufR9TmY3u3aFVrCiAcV
- inject_expired_blockhash: false
- tip_lamports: 80000
- slots: submitted=429345747 processed=429345787 confirmed=0 finalized=429345787
- commitment_progression: created -> submitted -> processed -> finalized
- submitted_at: 2026-06-28T00:38:46Z
- processed_at: 2026-06-28T00:39:07Z
- confirmed_at: -
- finalized_at: 2026-06-28T00:39:07Z
- failed_at: -

### 8. finalized

- transaction_id: `tx_76d4ddd2-c37f-45b9-ae40-80f264d64bf4`
- bundle_id: `bf982be337fa1ee4d5cbdb22c643d25dca9613f34e62b4db324ccc9b2d3f4c67`
- signature: `5eEfR4D8FKg1Ur1MRCHFy93qTxJYBekDuBKqcd9SKPHijU8MLhE4ZyYZLYFEGSnFMBWYWt2s4oiTR8ELCrZX636y`
- explorer: https://explorer.solana.com/tx/5eEfR4D8FKg1Ur1MRCHFy93qTxJYBekDuBKqcd9SKPHijU8MLhE4ZyYZLYFEGSnFMBWYWt2s4oiTR8ELCrZX636y
- solscan: https://solscan.io/tx/5eEfR4D8FKg1Ur1MRCHFy93qTxJYBekDuBKqcd9SKPHijU8MLhE4ZyYZLYFEGSnFMBWYWt2s4oiTR8ELCrZX636y
- inject_expired_blockhash: false
- tip_lamports: 80000
- slots: submitted=429345843 processed=429345854 confirmed=429345854 finalized=429345854
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-28T00:39:14Z
- processed_at: 2026-06-28T00:39:17Z
- confirmed_at: 2026-06-28T00:39:27Z
- finalized_at: 2026-06-28T00:39:32Z
- failed_at: -

### 9. finalized

- transaction_id: `tx_16e43e90-3579-4c9e-b2eb-0b1eb5525b39`
- bundle_id: `d1fe3d624cc67af7e19711c3b28955075e8993460fe9b1375d84c72935928ce9`
- signature: `51LkdKoebk8vPLEp25LTucseepM3km1cB9XqttHCHhibCEnQgaNVSd4GPsjhZQpNwChbFFxp6sFJ1Lyhjic1hHws`
- explorer: https://explorer.solana.com/tx/51LkdKoebk8vPLEp25LTucseepM3km1cB9XqttHCHhibCEnQgaNVSd4GPsjhZQpNwChbFFxp6sFJ1Lyhjic1hHws
- solscan: https://solscan.io/tx/51LkdKoebk8vPLEp25LTucseepM3km1cB9XqttHCHhibCEnQgaNVSd4GPsjhZQpNwChbFFxp6sFJ1Lyhjic1hHws
- inject_expired_blockhash: false
- tip_lamports: 120157
- slots: submitted=429345866 processed=429345875 confirmed=429345875 finalized=429345875
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-28T00:39:22Z
- processed_at: 2026-06-28T00:39:27Z
- confirmed_at: 2026-06-28T00:39:32Z
- finalized_at: 2026-06-28T00:39:37Z
- failed_at: -

### 10. finalized

- transaction_id: `tx_0b370b70-da12-4270-bad1-59c99281a5f9`
- bundle_id: `b1925823c1ad73a8385694ea75bb7a72564e64e3c6078351e2d296e4c53c572d`
- signature: `5BQhVBHcoUFDhjTYspuXYCpLd8FhFdmoriJCun8tW7o3HxU5h1bGaszBt5rRYRzY6h7mBqk3xGwQsgLPeU7scVVg`
- explorer: https://explorer.solana.com/tx/5BQhVBHcoUFDhjTYspuXYCpLd8FhFdmoriJCun8tW7o3HxU5h1bGaszBt5rRYRzY6h7mBqk3xGwQsgLPeU7scVVg
- solscan: https://solscan.io/tx/5BQhVBHcoUFDhjTYspuXYCpLd8FhFdmoriJCun8tW7o3HxU5h1bGaszBt5rRYRzY6h7mBqk3xGwQsgLPeU7scVVg
- inject_expired_blockhash: false
- tip_lamports: 126165
- slots: submitted=429345893 processed=429345904 confirmed=429345904 finalized=429345904
- commitment_progression: created -> submitted -> processed -> confirmed -> finalized
- submitted_at: 2026-06-28T00:39:33Z
- processed_at: 2026-06-28T00:39:37Z
- confirmed_at: 2026-06-28T00:39:47Z
- finalized_at: 2026-06-28T00:39:52Z
- failed_at: -

## Explorer verification

Open any `explorer` link above, or paste the full signature into
[Solana Explorer](https://explorer.solana.com/) / [Solscan](https://solscan.io/),
and cross-reference the confirmed/finalized slot recorded here.
