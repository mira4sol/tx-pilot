# TX Pilot Lifecycle Log Evidence

Live mainnet Jito submissions captured by `TestBountyLifecycleLog`. Every
signature, bundle id and slot below is reproduced in full (untruncated) so
each transaction can be independently verified on a Solana explorer.

- Generated: 2026-06-27T14:40:34Z
- Submissions: 10 (target 8 success / 2 failure)
- Lifecycle log entries exported: 50

- Observed: 8 success / 2 failed

## Overview

| #   | Status    | sub slot  | proc slot | conf slot | fin slot  | Tip (lamports) | Failure                                                        |
| --- | --------- | --------- | --------- | --------- | --------- | -------------- | -------------------------------------------------------------- |
| 1   | failed    | 429255986 | 0         | 0         | 0         | 70000          | expired_blockhash (Blockhash expired before the bundle landed) |
| 2   | failed    | 429256011 | 0         | 0         | 0         | 75000          | expired_blockhash (Blockhash expired before the bundle landed) |
| 3   | finalized | 429256015 | 429256028 | 429256028 | 429256028 | 80000          | -                                                              |
| 4   | finalized | 429256024 | 429256064 | 429256064 | 429256064 | 70000          | -                                                              |
| 5   | finalized | 429256082 | 429256105 | 429256105 | 429256105 | 80000          | -                                                              |
| 6   | finalized | 429256099 | 429256141 | 429256141 | 429256141 | 70000          | -                                                              |
| 7   | finalized | 429256159 | 429256173 | 429256173 | 429256173 | 65000          | -                                                              |
| 8   | finalized | 429256183 | 429256201 | 429256201 | 429256201 | 65000          | -                                                              |
| 9   | finalized | 429256200 | 429256244 | 429256244 | 429256244 | 70000          | -                                                              |
| 10  | finalized | 429256235 | 429256280 | 429256280 | 429256280 | 70000          | -                                                              |

## Full transaction records

### 1. failed

- transaction_id: `tx_3c4fe90d-7066-4543-92a8-ea2b36612644`
- bundle_id: `068738840f420080f6365dd039206a421b4032f72b1a369960da599975a4fc7e`
- signature: `2oCkft3n24WWYYinfGnNvdA4hSS8syonSnpcQmhGBBiAyeAEu9vLz6NqAWGqrfdETGsHTiz8GgpaWDeBdfxaKd1u`
- explorer: https://explorer.solana.com/tx/2oCkft3n24WWYYinfGnNvdA4hSS8syonSnpcQmhGBBiAyeAEu9vLz6NqAWGqrfdETGsHTiz8GgpaWDeBdfxaKd1u
- solscan: https://solscan.io/tx/2oCkft3n24WWYYinfGnNvdA4hSS8syonSnpcQmhGBBiAyeAEu9vLz6NqAWGqrfdETGsHTiz8GgpaWDeBdfxaKd1u
- inject_expired_blockhash: true
- tip_lamports: 70000
- slots: submitted=429255986 processed=0 confirmed=0 finalized=0
- commitment_progression: created -> submitted -> failed
- submitted_at: 2026-06-27T14:37:42Z
- processed_at: -
- confirmed_at: -
- finalized_at: -
- failed_at: 2026-06-27T14:37:45Z
- failure_kind: expired_blockhash
- failure_title: Blockhash expired before the bundle landed

### 2. failed

- transaction_id: `tx_adc03f41-0a16-4518-ac8c-080f114ea45b`
- bundle_id: `cd71ac7199dcbaa8cfc746d92959d8b92ef175f185ece30a53b4ead9506a6cbc`
- signature: `61e4vcc9R1tegxdjKUo2jDVDHyDNrUb2PHkySECu7V2K2Qfw9KcsBFdqMzZA4H4jTdGenN96ATZBa67tRTFsZpPK`
- explorer: https://explorer.solana.com/tx/61e4vcc9R1tegxdjKUo2jDVDHyDNrUb2PHkySECu7V2K2Qfw9KcsBFdqMzZA4H4jTdGenN96ATZBa67tRTFsZpPK
- solscan: https://solscan.io/tx/61e4vcc9R1tegxdjKUo2jDVDHyDNrUb2PHkySECu7V2K2Qfw9KcsBFdqMzZA4H4jTdGenN96ATZBa67tRTFsZpPK
- inject_expired_blockhash: true
- tip_lamports: 75000
- slots: submitted=429256011 processed=0 confirmed=0 finalized=0
- commitment_progression: created -> submitted -> failed
- submitted_at: 2026-06-27T14:37:51Z
- processed_at: -
- confirmed_at: -
- finalized_at: -
- failed_at: 2026-06-27T14:37:55Z
- failure_kind: expired_blockhash
- failure_title: Blockhash expired before the bundle landed

### 3. finalized

- transaction_id: `tx_5e10ba4c-2d4e-4ac2-b0be-c1d9c51c91d0`
- bundle_id: `c8319ae19261c6a38dcd66cc75f1d4bb336d4e969f5a882ffd61a4e541bf19af`
- signature: `4TYZ9MzPHjSLudYQmZmtGMMqQxGt9swEjf7psKmiyGEbBy5wQz4dB8TLSUjHQrnjkfsQAboUs2aHaT9inYSCamhP`
- explorer: https://explorer.solana.com/tx/4TYZ9MzPHjSLudYQmZmtGMMqQxGt9swEjf7psKmiyGEbBy5wQz4dB8TLSUjHQrnjkfsQAboUs2aHaT9inYSCamhP
- solscan: https://solscan.io/tx/4TYZ9MzPHjSLudYQmZmtGMMqQxGt9swEjf7psKmiyGEbBy5wQz4dB8TLSUjHQrnjkfsQAboUs2aHaT9inYSCamhP
- inject_expired_blockhash: false
- tip_lamports: 80000
- slots: submitted=429256015 processed=429256028 confirmed=429256028 finalized=429256028
- commitment_progression: created -> submitted -> processed -> confirmed
- submitted_at: 2026-06-27T14:38:00Z
- processed_at: 2026-06-27T14:38:05Z
- confirmed_at: 2026-06-27T14:38:10Z
- finalized_at: 2026-06-27T14:38:15Z
- failed_at: -

### 4. finalized

- transaction_id: `tx_745d85a9-0111-4beb-888b-5583e322234b`
- bundle_id: `38b86a456cd4f1024aaf9c58e23caa158eecf0efcfb03353bed566b76f2be98d`
- signature: `5ztheu4YFYHXfaoPmUT4X49x5ZZ697WFUNP5z8LJNk2CfyCnLHnXb4WBKhrvBeCe3YF8yXyRK3FLLwAGjcseVGdT`
- explorer: https://explorer.solana.com/tx/5ztheu4YFYHXfaoPmUT4X49x5ZZ697WFUNP5z8LJNk2CfyCnLHnXb4WBKhrvBeCe3YF8yXyRK3FLLwAGjcseVGdT
- solscan: https://solscan.io/tx/5ztheu4YFYHXfaoPmUT4X49x5ZZ697WFUNP5z8LJNk2CfyCnLHnXb4WBKhrvBeCe3YF8yXyRK3FLLwAGjcseVGdT
- inject_expired_blockhash: false
- tip_lamports: 70000
- slots: submitted=429256024 processed=429256064 confirmed=429256064 finalized=429256064
- commitment_progression: created -> submitted -> processed
- submitted_at: 2026-06-27T14:38:15Z
- processed_at: 2026-06-27T14:38:20Z
- confirmed_at: 2026-06-27T14:38:25Z
- finalized_at: 2026-06-27T14:38:30Z
- failed_at: -

### 5. finalized

- transaction_id: `tx_53ce0125-9836-4c37-ad07-22e311279554`
- bundle_id: `f22734ebf4ebf10a36c8477b50834150ce55a00d38120fbd95c5a89474ef24fc`
- signature: `52AWQWNNF2dwsxt3xWLn7xWvE8gU2oUhg8jBrwFkPS76FvxDSmBbyH9fbYXWAbvYLQDTAMp7WNPXEpT1k9EFRq9u`
- explorer: https://explorer.solana.com/tx/52AWQWNNF2dwsxt3xWLn7xWvE8gU2oUhg8jBrwFkPS76FvxDSmBbyH9fbYXWAbvYLQDTAMp7WNPXEpT1k9EFRq9u
- solscan: https://solscan.io/tx/52AWQWNNF2dwsxt3xWLn7xWvE8gU2oUhg8jBrwFkPS76FvxDSmBbyH9fbYXWAbvYLQDTAMp7WNPXEpT1k9EFRq9u
- inject_expired_blockhash: false
- tip_lamports: 80000
- slots: submitted=429256082 processed=429256105 confirmed=429256105 finalized=429256105
- commitment_progression: created -> submitted -> processed
- submitted_at: 2026-06-27T14:38:28Z
- processed_at: 2026-06-27T14:38:35Z
- confirmed_at: 2026-06-27T14:38:40Z
- finalized_at: 2026-06-27T14:38:45Z
- failed_at: -

### 6. finalized

- transaction_id: `tx_67fe4968-0132-422b-a785-4e326e310645`
- bundle_id: `dd8bd76ce12285fed2949879be7ca5fb441322514e09deee7bad9a5caa635264`
- signature: `2aeMfBnDFsDTTwiM198MzHT4nweGdLkTUsFHmLez4Hrz7ETPjBrdZV1kzuaXW4P1Xcr1UGt3R1dPAMkzT1HV3uWG`
- explorer: https://explorer.solana.com/tx/2aeMfBnDFsDTTwiM198MzHT4nweGdLkTUsFHmLez4Hrz7ETPjBrdZV1kzuaXW4P1Xcr1UGt3R1dPAMkzT1HV3uWG
- solscan: https://solscan.io/tx/2aeMfBnDFsDTTwiM198MzHT4nweGdLkTUsFHmLez4Hrz7ETPjBrdZV1kzuaXW4P1Xcr1UGt3R1dPAMkzT1HV3uWG
- inject_expired_blockhash: false
- tip_lamports: 70000
- slots: submitted=429256099 processed=429256141 confirmed=429256141 finalized=429256141
- commitment_progression: created -> submitted -> processed
- submitted_at: 2026-06-27T14:38:45Z
- processed_at: 2026-06-27T14:38:50Z
- confirmed_at: 2026-06-27T14:38:55Z
- finalized_at: 2026-06-27T14:39:00Z
- failed_at: -

### 7. finalized

- transaction_id: `tx_49a804eb-b0b4-42de-b0d7-2ed39b69474e`
- bundle_id: `8a6116087e950428832fd441b6130bc18ae5d4dee9215be6c212043b5285e464`
- signature: `41AxuJA1H8fqVTkwgr3StwH8UUUnsYjMTDADugdLNn8xT23MP8KtbrD8twBSrPWDdR1e4zyZzZjdPgd7imeBjW5w`
- explorer: https://explorer.solana.com/tx/41AxuJA1H8fqVTkwgr3StwH8UUUnsYjMTDADugdLNn8xT23MP8KtbrD8twBSrPWDdR1e4zyZzZjdPgd7imeBjW5w
- solscan: https://solscan.io/tx/41AxuJA1H8fqVTkwgr3StwH8UUUnsYjMTDADugdLNn8xT23MP8KtbrD8twBSrPWDdR1e4zyZzZjdPgd7imeBjW5w
- inject_expired_blockhash: false
- tip_lamports: 65000
- slots: submitted=429256159 processed=429256173 confirmed=429256173 finalized=429256173
- commitment_progression: created -> submitted -> confirmed
- submitted_at: 2026-06-27T14:38:57Z
- processed_at: 2026-06-27T14:39:00Z
- confirmed_at: 2026-06-27T14:39:10Z
- finalized_at: 2026-06-27T14:39:15Z
- failed_at: -

### 8. finalized

- transaction_id: `tx_9447e20f-8dc4-4655-996d-8c92a63f2285`
- bundle_id: `a7e31350cd74f3fadffd0c786fec7b96141b4cc0a7f166577424f108a412c24b`
- signature: `5gFsTpggRRUySNZJmVmfnfHKyvuX3KkM1AwuMr7VehuqxCxFi8RZ3dN9Lo1GyqNuzTsWq8dM92ibAxKpHEVxYHpX`
- explorer: https://explorer.solana.com/tx/5gFsTpggRRUySNZJmVmfnfHKyvuX3KkM1AwuMr7VehuqxCxFi8RZ3dN9Lo1GyqNuzTsWq8dM92ibAxKpHEVxYHpX
- solscan: https://solscan.io/tx/5gFsTpggRRUySNZJmVmfnfHKyvuX3KkM1AwuMr7VehuqxCxFi8RZ3dN9Lo1GyqNuzTsWq8dM92ibAxKpHEVxYHpX
- inject_expired_blockhash: false
- tip_lamports: 65000
- slots: submitted=429256183 processed=429256201 confirmed=429256201 finalized=429256201
- commitment_progression: created -> submitted -> processed -> confirmed
- submitted_at: 2026-06-27T14:39:07Z
- processed_at: 2026-06-27T14:39:15Z
- confirmed_at: 2026-06-27T14:39:20Z
- finalized_at: 2026-06-27T14:39:25Z
- failed_at: -

### 9. finalized

- transaction_id: `tx_c6b5aaa8-bb79-4230-ad4d-4602803f59f2`
- bundle_id: `034eb114a54bf738329d0da84a3bd03eddef07b492ba699f944172c5400ba154`
- signature: `52GJgrhupuWhiX5U9NxKQcDTzcVGBvjdmQLcCkKEkpZvR58GpoEEdAYrgW9QbvGjVD1puTc4XV4H32vkW1FK3D8J`
- explorer: https://explorer.solana.com/tx/52GJgrhupuWhiX5U9NxKQcDTzcVGBvjdmQLcCkKEkpZvR58GpoEEdAYrgW9QbvGjVD1puTc4XV4H32vkW1FK3D8J
- solscan: https://solscan.io/tx/52GJgrhupuWhiX5U9NxKQcDTzcVGBvjdmQLcCkKEkpZvR58GpoEEdAYrgW9QbvGjVD1puTc4XV4H32vkW1FK3D8J
- inject_expired_blockhash: false
- tip_lamports: 70000
- slots: submitted=429256200 processed=429256244 confirmed=429256244 finalized=429256244
- commitment_progression: created -> submitted -> processed -> confirmed
- submitted_at: 2026-06-27T14:39:27Z
- processed_at: 2026-06-27T14:39:30Z
- confirmed_at: 2026-06-27T14:39:40Z
- finalized_at: 2026-06-27T14:39:45Z
- failed_at: -

### 10. finalized

- transaction_id: `tx_9cc04a03-f57b-4258-830d-f1aa43e834ac`
- bundle_id: `98a81486f00cb5c90bb36b8f39492addf5973b9745e8d20e6d66a67cb19c296f`
- signature: `5LF8EFWoQaaRBGQNqEy84bxYV4uVjHSc4DFRDD5g17UjsmwTUCgMAV6ZwTqkykhKSHiq8qnEUzJfgV9xCmUh2myT`
- explorer: https://explorer.solana.com/tx/5LF8EFWoQaaRBGQNqEy84bxYV4uVjHSc4DFRDD5g17UjsmwTUCgMAV6ZwTqkykhKSHiq8qnEUzJfgV9xCmUh2myT
- solscan: https://solscan.io/tx/5LF8EFWoQaaRBGQNqEy84bxYV4uVjHSc4DFRDD5g17UjsmwTUCgMAV6ZwTqkykhKSHiq8qnEUzJfgV9xCmUh2myT
- inject_expired_blockhash: false
- tip_lamports: 70000
- slots: submitted=429256235 processed=429256280 confirmed=429256280 finalized=429256280
- commitment_progression: created -> submitted -> processed -> confirmed
- submitted_at: 2026-06-27T14:39:41Z
- processed_at: 2026-06-27T14:39:45Z
- confirmed_at: 2026-06-27T14:39:55Z
- finalized_at: 2026-06-27T14:40:00Z
- failed_at: -

## Explorer verification

Open any `explorer` link above, or paste the full signature into
[Solana Explorer](https://explorer.solana.com/) / [Solscan](https://solscan.io/),
and cross-reference the confirmed/finalized slot recorded here.
