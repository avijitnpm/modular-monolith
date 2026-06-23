package entitlements

const Unlimited int64 = -1

type PlanLimits struct {
	Users       int64
	Documents   int64
	APIRequests int64
	Storage     int64
}

var Plans = map[string]PlanLimits{
	"free": {
		Users:       1,
		Documents:   50,
		APIRequests: 1000,
		Storage:     104857600,
	},
	"pro": {
		Users:       10,
		Documents:   5000,
		APIRequests: 100000,
		Storage:     10737418240,
	},
	"enterprise": {
		Users:       Unlimited,
		Documents:   Unlimited,
		APIRequests: Unlimited,
		Storage:     Unlimited,
	},
}

type Entitlement struct {
	Metric    string `json:"metric"`
	Used      int64  `json:"used"`
	Limit     int64  `json:"limit"`
	Remaining int64  `json:"remaining"`
	Allowed   bool   `json:"allowed"`
}

var metrics = []string{"users", "documents", "api_requests", "storage"}

func limitForPlan(plan string, metric string) int64 {
	p, ok := Plans[plan]
	if !ok {
		p = Plans["free"]
	}
	switch metric {
	case "users":
		return p.Users
	case "documents":
		return p.Documents
	case "api_requests":
		return p.APIRequests
	case "storage":
		return p.Storage
	default:
		return 0
	}
}
