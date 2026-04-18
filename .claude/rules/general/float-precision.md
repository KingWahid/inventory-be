# Float Precision Rules

## The Rule

Always use `float64` in domain types. Never use `float32`.

`float32` cannot represent values like `0.35` exactly — it becomes `0.34999999403953552246`. This corrupts sensor readings, GPS coordinates, financial values, and thresholds.

## Domain Types (`pkg/services/types/`)

```go
// Correct
type DeviceConfig struct {
    KFactor              *float64 `json:"k_factor,omitempty"`
    CriticalThresholdBar *float64 `json:"critical_threshold_bar,omitempty"`
}

// Wrong
type DeviceConfig struct {
    KFactor              *float32 `json:"k_factor,omitempty"`
}
```

## API Boundary Conversion

OpenAPI/AsyncAPI stubs may generate `float32`. Convert at the boundary only:

```go
// Request: float32 (stub) -> float64 (domain)
if r.Value != nil {
    val := float64(*r.Value)
    result.Value = &val
}

// Response: float64 (domain) -> float32 (stub)
if r.Value != nil {
    val := float32(*r.Value)
    result.Value = &val
}
```

## MQTT/Wire Protocol

Accept `float32` from devices for bandwidth efficiency. Convert to `float64` immediately at the service boundary:

```go
err := h.svc.HandleFlowData(ctx, deviceID,
    float64(msg.Payload.FlowRate),
    float64(msg.Payload.Volume))
```
