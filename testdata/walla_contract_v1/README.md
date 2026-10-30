# Walla synthetic file contract

This fixture contains metadata only. It is a byte-for-byte copy of the XML and JSON in `kofac-wbs/Walla-DDI-v1-합성검증-20261030`.

- Walla source HEAD: `7371d128c3793617872447eb2a1a7ae6d36bce97`, including uncommitted source changes recorded by the producer.
- XML SHA256: `21c6d6bad4ba8e7a013b1f4a58a0d316710d504efb285ff41ce7e98898aacc7b`
- JSON SHA256: `f431a29ddf05b79d36bf758cb2a7aa9c774de905597d3e7322c0a793760899b0`

The producer called its current model and serializer functions. Its accompanying CSV is a manually authored expectation, not a database export. No customer responses are included here.

The regression test compares every tree node, attribute, text value and raw XML after XLSX conversion. A separate in-memory variation tests literal character codes, a missing-value reason, foreign metadata and dropped-comment warnings; those additions are not part of the Walla output.
