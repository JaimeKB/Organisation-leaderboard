package metrics

// registered lists every known Metric implementation. Adding a new metric
// (e.g. lines added/deleted, PR count) is a new file implementing Metric
// plus one entry here.
var registered = []Metric{
	CommitCountMetric{},
}

// Enabled returns the metrics shown on the leaderboard, in display order.
func Enabled() []Metric {
	return registered
}

// Lookup returns the Metric for a given key, e.g. from a ?sort= query param.
func Lookup(key string) (Metric, bool) {
	for _, m := range registered {
		if m.Key() == key {
			return m, true
		}
	}
	return nil, false
}
