package worker

// Queue-related helpers beyond the basic BRPop loop in pool.go will live
// here as the build pipeline grows (e.g. retry logic, dead-letter queue).