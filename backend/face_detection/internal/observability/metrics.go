package observability

import "github.com/prometheus/client_golang/prometheus"

var (
	EmbeddingGenerationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:		"face_detection_embedding_generation_duration_seconds",
		Help: 		"Time to generate face embeddings from image bytes",
		Buckets: 	[]float64{0.05, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})

	ProfileEmbFetchDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:		"face_detection_profile_emb_fetch_duration_seconds",
		Help: 		"Time to fetch profile face embeddings from DB",
		Buckets: 	[]float64{0.05, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})

	VisitorEmbFetchDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:		"face_detection_visitor_emb_fetch_duration_seconds",
		Help: 		"Time to fetch visitor face embeddings from DB",
		Buckets: 	[]float64{0.05, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})

	ProfileComparisonDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:		"face_detection_profile_comparison_duration_seconds",
		Help: 		"Time to compare profile face embeddings from DB to recieved face emb",
		Buckets: 	[]float64{0.05, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})


	VisitorComparisonDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:		"face_detection_visitor_comparison_duration_seconds",
		Help: 		"Time to compare visitor face embeddings from DB to recieved face emb",
		Buckets: 	[]float64{0.05, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})

	BriefingFetchDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:		"face_detection_briefing_fetch_duration_seconds",
		Help: 		"Time to fetch visitor briefing from DB",
		Buckets: 	[]float64{0.05, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})

	ProfileRegisterDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:		"face_detection_profile_register_duration_seconds",
		Help: 		"Time to register a new profile with face embeddings in DB",
		Buckets: 	[]float64{0.05, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})

	DBReadsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:	"face_detection_db_reads_total",
		Help:	"Total DB read operations by operation",
	}, []string{"operation"})
	
	DBWritesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:	"face_detection_db_writes_total",
		Help:	"Total DB write operations by operation",
	}, []string{"operation"})
	
	CacheHitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:	"face_detection_cache_hits_total",
		Help:	"Total cache hits by operation",
	}, []string{"operation"})
	
	CacheMissesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:	"face_detection_cache_misses_total",
		Help:	"Total cache misses by operation",
	}, []string{"operation"})
	
	ErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:	"face_detection_errors_total",
		Help:	"Total errors by operation",
	}, []string{"operation"})
)

func Register() {
	prometheus.MustRegister(
		EmbeddingGenerationDuration,
		ProfileEmbFetchDuration,
		VisitorEmbFetchDuration,
		ProfileComparisonDuration,
		VisitorComparisonDuration,
		BriefingFetchDuration,
		ProfileRegisterDuration,
		DBReadsTotal,
		DBWritesTotal,
		CacheHitsTotal,
		CacheMissesTotal,
		ErrorsTotal,
	)
}