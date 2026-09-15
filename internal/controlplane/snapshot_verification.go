package controlplane

// VerifySnapshot exposes the existing v1.5.0 snapshot invariant checks to
// persistence adapters without moving database mechanics into the Control Plane.
func VerifySnapshot(snapshot ProjectSnapshot) error {
	return validateSnapshot(snapshot)
}
