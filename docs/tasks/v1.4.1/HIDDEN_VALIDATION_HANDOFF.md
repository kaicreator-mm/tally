# v1.4.1 Hidden Validation Handoff

The actual Hidden Validation Pack is intentionally distributed **outside the repository**.

- Expected archive: `Tally_v1.4.1_Hidden_Validation_Pack.zip`
- Expected SHA-256: `67863529795f31b4a45848ee50e7746c49ef5ae91267b2ceb9392d8a400e53e0`

## Timing

Do not mount, open, search, extract or summarize the hidden pack until:

1. all 57 visible tasks have reached an honest terminal state;
2. all visible phase gates are complete;
3. the candidate repository HEAD is frozen;
4. implementation agents have stopped editing.

Then:

1. verify the archive SHA-256;
2. extract it outside the repository;
3. run its `prompts/HIDDEN_REVIEW_PROMPT.md` with a HIGH-capability independent validator;
4. run `run_hidden_validation.py`;
5. write `docs/tasks/v1.4.1/reports/V1.4.1_HIDDEN_VALIDATION.md`;
6. if the hidden verdict is FAIL, return defects to a new remediation cycle and rerun affected hidden scenarios.

Hidden tests are not implementation references.
