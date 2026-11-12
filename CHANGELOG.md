# Historial de cambios

<!-- version list -->

## v0.9.0 (2026-11-12)

### Correcciones

- configure semantic-release for the Go layout ([`9856e54`](https://github.com/Gunmer/kamino/commit/9856e54e9fdd6aba7fd6f2a9c9668054525158d8))
- restore golden fixture and Python readme ([`8b3286c`](https://github.com/Gunmer/kamino/commit/8b3286c185714353976edddfac449df639ede9b5))
- return to column zero after each line ([`767670b`](https://github.com/Gunmer/kamino/commit/767670be2fa8f831b343fe5f61d4a21d5b65a1af))

### Mantenimiento

- verify with the real toolchain ([`6841bec`](https://github.com/Gunmer/kamino/commit/6841bec0b31d1fa159ba820669ee94f1fb089f57))

### Documentación

- add migrate-to-go change ([`b58b456`](https://github.com/Gunmer/kamino/commit/b58b456c14161ddc7fe34b165c03c86d32f41fe9))

### Funcionalidades

- migrate to Go ([`755e93c`](https://github.com/Gunmer/kamino/commit/755e93c44b37e3abd0e051d8754b4b8154a4b36e))

  **Cambio incompatible.** la distribución pasa a un binario Go estático; uv y pip dejan de ser vías de instalación y la TUI solo admite ansi-dark y ansi-light.
- add Go distribution, core packages and TUI ([`f1b2d43`](https://github.com/Gunmer/kamino/commit/f1b2d430feec169aba957918ec58bb32d166c99e))


## v0.8.0 (2026-11-12)

### Funcionalidades

- add D0 clone picker ([`ba1d53f`](https://github.com/Gunmer/kamino/commit/ba1d53f5bf89fd1897ac34608f5b4cc10924cb70))
- remove menu keyboard shortcuts ([`5e28982`](https://github.com/Gunmer/kamino/commit/5e28982e8243e8a091e71c7630e6b74343001e72))

  **Cambio incompatible.** `c`, `d`, `u` and `q` no longer trigger menu actions. Use arrow keys plus `Enter`; exit by selecting Abandonar Kamino.
- add TUI-managed user configuration ([`a05f909`](https://github.com/Gunmer/kamino/commit/a05f909ebf21100702e28c4d345e2e96748518dc))

  **Cambio incompatible.** clone writes to ~/.config/kamino/vault by default and snapshots move to ~/.config/kamino/vault/snapshot; existing snapshots under ~/Library/Application Support/kamino/snapshots are not migrated.


## v0.7.0 (2026-11-09)

### Mantenimiento

- rename backup-safety and restore-conflicts capabilities ([`04d398b`](https://github.com/Gunmer/kamino/commit/04d398b1134ae332add1124d6eb080f12798f041))

### Documentación

- archive renombrar-clone-deploy ([`2b092c9`](https://github.com/Gunmer/kamino/commit/2b092c9ad65c92f9379ac0b69110172532001755))

### Funcionalidades

- unify interactive flows with keyboard navigation ([`9809d1c`](https://github.com/Gunmer/kamino/commit/9809d1cdd4fa6cf1bde47cb76898989f1908dbb6))
- rename backup/restore to clone/deploy ([`8e5e5c3`](https://github.com/Gunmer/kamino/commit/8e5e5c39e2f239f0eeaabd1041d4ebb125b65596))

  **Cambio incompatible.** `kamino backup` and `kamino restore` are removed; use `kamino clone` and `kamino deploy` instead.

### Pruebas

- stub update query for up-to-date result ([`2f31690`](https://github.com/Gunmer/kamino/commit/2f316901db4d7b3daf86cd26b198ff114d80ec0d))


## v0.6.0 (2026-11-08)




## v0.5.0 (2026-11-08)

### Documentación

- archive expanded catalog change ([`79469a1`](https://github.com/Gunmer/kamino/commit/79469a11665e37b848d9bfd3285e04b7cb8a7073))

### Funcionalidades

- add block-level conflict resolution and TUI ([`b57168e`](https://github.com/Gunmer/kamino/commit/b57168e9ac4fd7da8970df9a71bfded0dffaae8b))


## v0.4.0 (2026-11-07)

### Funcionalidades

- add macOS Apple Silicon installer ([`b0e5c16`](https://github.com/Gunmer/kamino/commit/b0e5c16ee35d7a94f99ce0cc3410bcd2dd39ce13))
- expand app coverage and harden backups ([`fbfe042`](https://github.com/Gunmer/kamino/commit/fbfe042a364445f96c4a82f9d1140719e182e6f3))

### Pruebas

- isolate inherited config paths ([`a296987`](https://github.com/Gunmer/kamino/commit/a296987ff522ff1eb4d6fe02ae190a0bef14a8e4))


## v0.3.0 (2026-11-04)

### Mantenimiento

- add SDD agents ([`c644d87`](https://github.com/Gunmer/kamino/commit/c644d8751dcb8ceb1edf87b29dc048a84eb63476))
- add sdd ([`89d1cb2`](https://github.com/Gunmer/kamino/commit/89d1cb2184a69d4ab2dbc74ffa6a2fe4a3f2dd9b))

### Funcionalidades

- add short option aliases ([`c58c02e`](https://github.com/Gunmer/kamino/commit/c58c02e9b7549b8b86a407cdcf00ff102bb390da))

### Pruebas

- handle ANSI output in help tests ([`cdc3839`](https://github.com/Gunmer/kamino/commit/cdc38397e91d02b971c01574af64430c124bc805))


## v0.2.0 (2026-11-02)

### Correcciones

- address R3 review findings — dispatch via ctx.invoke, separate concerns, CommandChoice alias ([`5bdd257`](https://github.com/Gunmer/kamino/commit/5bdd257cadb67b5c99ce9156a638dbdaf08e336b))

### Funcionalidades

- show TUI command menu when run without arguments ([`e0d1c06`](https://github.com/Gunmer/kamino/commit/e0d1c06898651091818348d3b55b3e94bb9b2060))


## v0.1.1 (2026-11-02)

### Correcciones

- automate versioning and embed catalog in ARM64 binary ([`0efd45b`](https://github.com/Gunmer/kamino/commit/0efd45b7b38deeb1fceecbbe460b61808f023d20))
- embed catalog data in packaged binary and smoke-test it ([`4d89b4f`](https://github.com/Gunmer/kamino/commit/4d89b4fc7c4c354ccbd6cc3354abe69eae775c4f))

### Integración continua

- fix workflow ([`2c46c8f`](https://github.com/Gunmer/kamino/commit/2c46c8f6b8e57eb74c49076a2f3a0cbae35361fe))
- remove uv cache ([`8a84433`](https://github.com/Gunmer/kamino/commit/8a8443391107fa0ca796a5d4ac82ed0c6c53a82f))
- fix uv dependencies ([`0ce0d3f`](https://github.com/Gunmer/kamino/commit/0ce0d3f391f303c86ae683967a3c5ec06aa14b5e))
- improve workflows ([`451cae5`](https://github.com/Gunmer/kamino/commit/451cae55ff7d49c31c4c836f921c48ab43dbd5ea))
- fix macos image ([`bb40b60`](https://github.com/Gunmer/kamino/commit/bb40b60bf45d1f23cb0da21a6fdae8e8015e980d))

### Documentación

- add AGENTS.md with project conventions ([`818d0f5`](https://github.com/Gunmer/kamino/commit/818d0f5a4db7011bf7936f06aa21540b8f4ae571))


## v0.1.0 (2026-10-31)

- Versión inicial
