package constants

import "time"

// EndpointTimeout is the default per-request deadline for handler → service calls.
const EndpointTimeout = 30 * time.Second
