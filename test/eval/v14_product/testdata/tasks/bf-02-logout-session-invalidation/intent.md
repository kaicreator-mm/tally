# Intent

Sessions that are created before logout keep authenticating afterwards.
After a session token is revoked by logout, any request carrying that token
must be rejected with an authentication error; other sessions must keep
working.

Find the session invalidation defect in the session store and fix it. Keep
the existing public function signatures unchanged. A regression check must
fail on the unfixed auth flow and pass after the fix, and the check must not
depend on implementation internals (source independence).
