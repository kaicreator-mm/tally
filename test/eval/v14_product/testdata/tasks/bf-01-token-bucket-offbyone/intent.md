# Intent

The API rate limiter admits one extra immediate request right after the
bucket is emptied. With capacity=3 and no refill elapsed, the first three
immediate requests must be allowed and the fourth must be rejected until
tokens refill.

Find the boundary defect in the token bucket limiter and fix it. Keep the
existing public function signature and package path unchanged. A regression
check must fail on the unfixed limiter and pass after the fix.
