// Task fixture baseline (bf-02). Representative buggy source snapshot of the
// session store under repair. Input encoding for the frozen oracle is a
// ';'-separated operation sequence:
//
//	create:<token>|<user>  logout:<token>  auth:<token>
//
// Output: ';'-separated results, one per operation:
//
//	create -> "created"   logout -> "logged-out"
//	auth   -> "ok:<user>" or "reject"
package auth

type store struct {
	valid map[string]string
}

func newStore() *store { return &store{valid: map[string]string{}} }

func (s *store) apply(op string) string {
	switch {
	case len(op) >= 7 && op[:7] == "create:":
		body := op[7:]
		sep := -1
		for i := 0; i < len(body); i++ {
			if body[i] == '|' {
				sep = i
				break
			}
		}
		s.valid[body[:sep]] = body[sep+1:]
		return "created"
	case len(op) >= 7 && op[:7] == "logout:":
		// BUG: revocation is a no-op; the token stays valid.
		_ = op[7:]
		return "logged-out"
	case len(op) >= 5 && op[:5] == "auth:":
		if user, ok := s.valid[op[5:]]; ok {
			return "ok:" + user
		}
		return "reject"
	}
	return "reject"
}
