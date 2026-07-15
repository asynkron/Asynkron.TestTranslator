# Istanbul coverage-final.json — Real Third-Party Fixtures

These are **unmodified**, byte-exact `coverage-final.json` files produced by the
Istanbul / nyc JavaScript coverage tooling and committed into public GitHub
repositories. They are used **read-only** as proving material for a converter.
No file contents were altered. Each is a JSON object keyed by absolute source-file
path, with per-file `path`, `statementMap`/`s`, `fnMap`/`f`, and `branchMap`/`b`.

All source repos are under permissive/OSI licenses (MIT, Apache-2.0, ISC).

| Local file | Repo | License (SPDX) | Bytes | Notes |
|---|---|---|---|---|
| lockr.json | tsironis/lockr | MIT | 20465 | TS library, 13 files, has branches |
| macaroons_js.json | nitram509/macaroons.js | Apache-2.0 | 4058 | JS lib, 13 files, empty branchMaps |
| nestjs-s3.json | ntegral/nestjs-s3 | ISC | 10036 | NestJS module, 9 files, has branches |
| react-lazyimg-component.json | zhansingsong/react-lazyimg-component | MIT | 44408 | small (3 files), React component |
| scienft-contracts.json | ScieNFT/contracts | MIT | 90523 | solidity-coverage output, 6 files (largest) |
| vue-cion-design-system.json | visualjerk/vue-cion-design-system | MIT | 79338 | large, 38 source files, Vue design system |

---

## Per-file provenance

### lockr.json
- Source repo: tsironis/lockr — https://github.com/tsironis/lockr
- Raw URL (commit-pinned): https://raw.githubusercontent.com/tsironis/lockr/ed23f98dcbd85bad8f9c5d96bf49e266c8f9deb6/coverage/coverage-final.json
- Original path: `coverage/coverage-final.json`
- License: MIT (LICENSE file: "MIT License Copyright (c) 2021 Dimitris Tsironis")
- Contents: Istanbul coverage for the lockr TypeScript localStorage helper library (13 source files, includes populated branchMaps).

### macaroons_js.json
- Source repo: nitram509/macaroons.js — https://github.com/nitram509/macaroons.js
- Raw URL (commit-pinned): https://raw.githubusercontent.com/nitram509/macaroons.js/d1cb54da166daaaca6a614f1191525914a2b05f7/coverage/coverage-final.json
- Original path: `coverage/coverage-final.json`
- License: Apache-2.0 (LICENSE file: Apache License Version 2.0)
- Contents: Istanbul coverage for the macaroons.js crypto library (13 files; branchMaps present but empty across files).

### nestjs-s3.json
- Source repo: ntegral/nestjs-s3 — https://github.com/ntegral/nestjs-s3
- Raw URL (commit-pinned): https://raw.githubusercontent.com/ntegral/nestjs-s3/a97e29699817989b95cf971af8f9283653a5be1d/coverage/coverage-final.json
- Original path: `coverage/coverage-final.json`
- License: ISC (declared in package.json `"license": "ISC"`; no standalone LICENSE file in repo)
- Contents: Istanbul coverage for a NestJS AWS S3 module (9 files, includes branches).

### react-lazyimg-component.json
- Source repo: zhansingsong/react-lazyimg-component — https://github.com/zhansingsong/react-lazyimg-component
- Raw URL (commit-pinned): https://raw.githubusercontent.com/zhansingsong/react-lazyimg-component/9d26ecc3732086a6b9d3173a91c011f5c8e215a0/coverage/coverage-final.json
- Original path: `coverage/coverage-final.json`
- License: MIT (LICENSE file: "MIT License", 2018-present singsong)
- Contents: Small Istanbul coverage report (3 source files) for a React lazy-image component.

### scienft-contracts.json
- Source repo: ScieNFT/contracts — https://github.com/ScieNFT/contracts
- Raw URL (commit-pinned): https://raw.githubusercontent.com/ScieNFT/contracts/29558837166b6024b346c7f639c4304c6f7ffc5c/coverage/coverage-final.json
- Original path: `coverage/coverage-final.json`
- License: MIT (LICENSE file: "MIT License Copyright (c) 2023 ScieNFT")
- Contents: solidity-coverage (Istanbul-format) output for Solidity smart contracts (6 files; largest fixture at ~90 KB).

### vue-cion-design-system.json
- Source repo: visualjerk/vue-cion-design-system — https://github.com/visualjerk/vue-cion-design-system
- Raw URL (commit-pinned): https://raw.githubusercontent.com/visualjerk/vue-cion-design-system/9574c03976f89a5a86761afea1a7e348fd8e6ca2/coverage/coverage-final.json
- Original path: `coverage/coverage-final.json`
- License: MIT (LICENSE file: "MIT License Copyright (c) 2019 Jörg Bayreuther")
- Contents: Large Istanbul coverage report (38 source files) for a Vue design system.
