package observability

import "github.com/prometheus/client_golang/prometheus"

var (
	ConvoFetchDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "conversation_briefing_conversation_fetch_duration_milliseconds",
		Help:    "Time to fetch conversations from DB",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
	})

	BriefingInsertDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "conversation_briefing_briefing_insert_duration_milliseconds",
		Help:    "Time to insert newly generated briefing into DB",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
	})

	LLMResponseDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "conversation_briefing_llm_response_duration_milliseconds",
		Help:    "Time to get a response from the LLM",
		Buckets: []float64{500, 1000, 2000, 5000, 10000, 30000, 60000},
	})

	DBReadsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "conversation_briefing_db_reads_total",
		Help: "Total DB read operations by operation",
	}, []string{"operation"})

	DBWritesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "conversation_briefing_db_writes_total",
		Help: "Total DB write operations by operation",
	}, []string{"operation"})

	CacheHitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "conversation_briefing_cache_hits_total",
		Help: "Total cache hits by operation",
	}, []string{"operation"})

	CacheMissesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "conversation_briefing_cache_misses_total",
		Help: "Total cache misses by operation",
	}, []string{"operation"})

	ErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "conversation_briefing_errors_total",
		Help: "Total errors by operation",
	}, []string{"operation"})
)

func RegisterMetrics() {
	prometheus.MustRegister(
		ConvoFetchDuration,
		BriefingInsertDuration,
		LLMResponseDuration,
		DBReadsTotal,
		DBWritesTotal,
		CacheHitsTotal,
		CacheMissesTotal,
		ErrorsTotal,
	)
}
