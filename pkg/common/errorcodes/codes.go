package errorcodes

const (
	CodeValidationError   = "VALIDATION_ERROR"
	CodeUnauthorized      = "UNAUTHORIZED"
	CodeForbidden         = "FORBIDDEN"
	CodeNotFound          = "NOT_FOUND"
	CodeConflict          = "CONFLICT"
	CodeIdempotency       = "IDEMPOTENCY_CONFLICT"
	CodeProductHasStock   = "PRODUCT_HAS_STOCK"
	CodeWarehouseHasStock = "WAREHOUSE_HAS_STOCK"
	CodeInsufficientStock = "INSUFFICIENT_STOCK"
	CodeMovementNotDraft       = "MOVEMENT_NOT_DRAFT"
	CodeCategoryHasActiveProducts = "CATEGORY_HAS_ACTIVE_PRODUCTS"
	CodeInternalError     = "INTERNAL_ERROR"
	CodeNotImplemented    = "NOT_IMPLEMENTED"
)

var (
	ErrValidationError = New(CodeValidationError, "Request body or params is invalid", 400)
	ErrUnauthorized    = New(CodeUnauthorized, "Missing or invalid token", 401)
	ErrForbidden       = New(CodeForbidden, "Insufficient permissions", 403)
	ErrNotFound        = New(CodeNotFound, "Entity not found", 404)
	ErrConflict        = New(CodeConflict, "Resource conflict", 409)
	ErrIdempotency     = New(CodeIdempotency, "Idempotency key has been processed", 409)

	ErrProductHasStock = New(CodeProductHasStock, "Cannot delete product with remaining stock", 422)
	ErrWarehouseStock  = New(CodeWarehouseHasStock, "Cannot deactivate warehouse with stock", 422)
	ErrInsufficient    = New(CodeInsufficientStock, "Outbound quantity exceeds available stock", 422)
	ErrMovementDraft   = New(CodeMovementNotDraft, "Cannot modify confirmed or cancelled movement", 422)
	ErrCategoryHasActiveProducts = New(CodeCategoryHasActiveProducts, "Cannot delete category while active products reference it", 422)

	ErrInternal = New(CodeInternalError, "Unexpected server error", 500)

	ErrNotImplemented = New(CodeNotImplemented, "Not implemented yet", 501)
)
