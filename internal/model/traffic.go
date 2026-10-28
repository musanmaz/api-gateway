package model

// ServiceTrafficBucket is an aggregate; no request paths, IPs, user agents or secrets
// are sent to the management plane.
type ServiceTrafficBucket struct {
 ServiceID string `json:"serviceId"`
 Minute int64 `json:"minute"`
 Requests uint64 `json:"requests"`
 Status2xx uint64 `json:"status2xx"`
 Status3xx uint64 `json:"status3xx"`
 Status4xx uint64 `json:"status4xx"`
 Status5xx uint64 `json:"status5xx"`
 Limited uint64 `json:"limited"`
 WAFMatches uint64 `json:"wafMatches"`
 WAFBlocked uint64 `json:"wafBlocked"`
 Durations [10]uint64 `json:"durations"`
}
