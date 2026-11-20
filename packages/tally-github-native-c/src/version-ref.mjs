/** Ordinary GitHub main baselines and existing version/* refs are supported.
 * A selected ref is still pinned to a verified current GitHub commit SHA.
 */
const MANAGED_REF=/^(?:main|version\/[A-Za-z0-9_.-]+)$/;
export function isValidVersionRef(ref) {
  return typeof ref==='string' && MANAGED_REF.test(ref);
}
