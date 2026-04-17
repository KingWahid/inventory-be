# How to Use Storage

A developer guide for uploading, downloading, and managing files using the storage client. The storage layer supports MinIO (local development) and Supabase/S3 (production) through a unified interface.

## Table of Contents

- [Overview](#overview)
- [The Client Interface](#the-client-interface)
- [Injecting the Storage Client](#injecting-the-storage-client)
- [Uploading Files](#uploading-files)
- [Downloading Files](#downloading-files)
- [Presigned URLs](#presigned-urls)
- [Checking if a File Exists](#checking-if-a-file-exists)
- [Deleting Files](#deleting-files)
- [File Path Conventions](#file-path-conventions)
- [Error Handling](#error-handling)
- [Using Storage in Background Processes](#using-storage-in-background-processes)
- [Local Development with MinIO](#local-development-with-minio)
- [Testing](#testing)
- [Related Guides](#related-guides)

---

## Overview

The storage system provides a single `storage.Client` interface backed by three implementations:

| Backend | When Used | How Selected |
|---------|-----------|-------------|
| **S3/MinIO** | `STORAGE_BACKEND=s3` + S3 credentials set | Default for local dev (`docker-compose.yaml` sets MinIO) |
| **Supabase** | Supabase URL + service key set (no S3 config) | Alternative production backend |
| **No-op** | Neither configured | App starts, but storage calls fail at runtime |

The backend is selected at startup in `pkg/common/storage/MODULE.go`. Your code never needs to know which backend is active — use the `Client` interface everywhere.

**Package:** `pkg/common/storage/`

---

## The Client Interface

Different environments use different storage backends — MinIO in local development, Supabase or S3 in production. The `Client` interface abstracts this away so all code uses the same API regardless of backend. FX selects the implementation at startup based on environment variables (`STORAGE_BACKEND`, S3 credentials, Supabase credentials). Your service code never needs to know which backend is active, and switching backends requires zero code changes — only environment variable updates.

```go
type Client interface {
    // Upload uploads a file to the specified bucket and path.
    Upload(ctx context.Context, bucket, path string, data []byte, contentType string) error

    // Download downloads a file from the specified bucket and path.
    Download(ctx context.Context, bucket, path string) ([]byte, error)

    // GetSignedURL generates a presigned PUT URL for uploading.
    GetSignedURL(ctx context.Context, bucket, path string, expiresIn time.Duration) (string, error)

    // GetSignedDownloadURL generates a presigned GET URL for downloading.
    GetSignedDownloadURL(ctx context.Context, bucket, path string, expiresIn time.Duration) (string, error)

    // Delete removes a file. Idempotent — deleting a non-existent file succeeds.
    Delete(ctx context.Context, bucket, path string) error

    // Exists checks if a file exists at the given path.
    Exists(ctx context.Context, bucket, path string) (bool, error)
}
```

### Bucket parameter

- Pass the bucket name explicitly, or pass `""` to use the default bucket (`S3_BUCKET` env var, typically `documents`).
- The S3 client's `bucket()` helper falls back to the default bucket when the caller passes an empty string.

### Path parameter

- Leading slashes are trimmed automatically (`/invoices/123.pdf` → `invoices/123.pdf`).
- Use forward slashes as directory separators.

---

## Injecting the Storage Client

The storage client is provided via FX dependency injection. Add it to your service's `Params` struct:

### In a service (`pkg/services/*/MODULE.go`)

```go
type ServiceParams struct {
    fx.In

    // ... other dependencies ...
    StorageClient storage.Client
}

func ProvideService(p ServiceParams) (ServiceResult, error) {
    svc := newServiceWithDependencies(
        // ... other deps ...
        p.StorageClient,
    )
    return ServiceResult{Service: svc}, nil
}
```

### In a worker handler (`workers/asynq/handlers.go`)

```go
type Handlers struct {
    JobService    bgprocess.BackgroundProcessService
    StorageClient storage.Client
}
```

The storage module is already included in the FX composition for `service-common`, `service-billing`, and `workers`. If your service doesn't have it yet, add `commonfx.StorageModule` to `services/*/cmd/main.go`.

---

## Uploading Files

### Direct upload (server-side)

Use when the server generates the file (e.g., PDF generation in a background process):

```go
func (s *service) uploadInvoicePDF(ctx context.Context, invoiceID uuid.UUID, pdfData []byte) error {
    path := fmt.Sprintf("invoices/%s/invoice.pdf", invoiceID.String())

    err := s.storageClient.Upload(ctx, "", path, pdfData, "application/pdf")
    if err != nil {
        return common.NewCustomError("failed to upload invoice PDF").
            WithErrorCode(errorcodes.StorageUploadFailed).
            WithHTTPCode(http.StatusInternalServerError).
            WithError(err)
    }

    return nil
}
```

### Presigned upload (client-side)

Use when the client (browser/app) uploads directly to storage:

```go
func (s *service) requestUploadURL(ctx context.Context, purpose, fileName string) (string, error) {
    path := fmt.Sprintf("%s/%s/%s", purpose, uuid.New().String(), fileName)

    signedURL, err := s.storageClient.GetSignedURL(ctx, "", path, 15*time.Minute)
    if err != nil {
        return "", err
    }

    return signedURL, nil
}
```

The client then PUTs the file directly to the signed URL — the file never passes through your service.

---

## Downloading Files

### Direct download (server-side)

```go
func (s *service) downloadFile(ctx context.Context, bucket, path string) ([]byte, error) {
    data, err := s.storageClient.Download(ctx, bucket, path)
    if err != nil {
        if common.IsNotFoundError(err) {
            return nil, common.NewCustomError("file not found").
                WithErrorCode(errorcodes.StorageFileNotFound).
                WithHTTPCode(http.StatusNotFound)
        }
        return nil, err
    }
    return data, nil
}
```

### Presigned download URL (client-side)

Use when you want to give the client a temporary download link:

```go
func (s *service) getDownloadURL(ctx context.Context, path string) (string, error) {
    return s.storageClient.GetSignedDownloadURL(ctx, "", path, 15*time.Minute)
}
```

**S3/MinIO note:** `GetSignedURL` generates a PUT presigned URL (for uploads), while `GetSignedDownloadURL` generates a GET presigned URL (for downloads). These are distinct operations.

**Supabase note:** Both methods use the same `/object/sign/` endpoint — Supabase's signed URLs work for both upload and download.

---

## Checking if a File Exists

```go
exists, err := s.storageClient.Exists(ctx, "", path)
if err != nil {
    return err
}
if !exists {
    // Handle missing file
}
```

---

## Deleting Files

```go
err := s.storageClient.Delete(ctx, "", path)
if err != nil {
    return err
}
```

Delete is **idempotent** — deleting a file that doesn't exist succeeds without error (both S3 and Supabase backends).

---

## File Path Conventions

Files are organized by purpose:

| Purpose | Path Pattern | Example |
|---------|-------------|---------|
| Profile images | `profiles/{user_id}/{filename}` | `profiles/abc-123/avatar.png` |
| Invoice PDFs | `invoices/{invoice_id}/{filename}` | `invoices/def-456/invoice.pdf` |
| Payment proofs | `invoices/{invoice_id}/attachments/{filename}` | `invoices/def-456/attachments/proof.jpg` |
| Reports | `reports/{report_id}/{filename}` | `reports/ghi-789/monthly-report.csv` |
| Pending uploads | `{purpose}/{uuid}/{filename}` | `profile_image/jkl-012/photo.png` |

**Rules:**
- Use UUIDs in paths to avoid collisions
- Use lowercase, hyphen-separated directory names
- Include the entity ID for easy lookup and cleanup
- Never include user-provided data in paths without sanitization

---

## Error Handling

All storage errors are wrapped in `common.CustomError` with specific error codes.

**The no-op client:** If neither S3 nor Supabase credentials are configured, the storage module creates a `noopClient` that returns an `InitializationError` (503 Service Unavailable) for every operation. This is a deliberate design choice — the app still starts (so you can develop features that don't need storage), but any actual storage call fails loudly with a clear message: "storage not configured; set S3 or Supabase env." It's a safety net that prevents silent data loss, not a silent fallback.

| Error Code | Constant | When |
|-----------|----------|------|
| `StorageConnectionError` | `errorcodes.StorageConnectionError` | Network/timeout failure |
| `StorageUploadFailed` | `errorcodes.StorageUploadFailed` | Upload operation failed |
| `StorageDownloadFailed` | `errorcodes.StorageDownloadFailed` | Download or read failure |
| `StorageSignedURLFailed` | `errorcodes.StorageSignedURLFailed` | Signed URL generation failed |
| `StorageDeleteFailed` | `errorcodes.StorageDeleteFailed` | Delete operation failed |
| `StorageFileNotFound` | `errorcodes.StorageFileNotFound` | File does not exist (404/NoSuchKey) |
| `StorageUnauthorized` | `errorcodes.StorageUnauthorized` | Auth failure (401/403) |
| `InitializationError` | `errorcodes.InitializationError` | No-op client hit at runtime |

### Checking error types

```go
if common.IsNotFoundError(err) {
    // File doesn't exist — handle gracefully
}
```

### Graceful handling for missing files

Some operations (like cleanup) should succeed even if the file was never uploaded:

```go
err := s.storageClient.Delete(ctx, bucket, path)
if err != nil {
    if common.IsNotFoundError(err) {
        // File was never uploaded — still mark record as cleaned up
        return nil
    }
    return err
}
```

**Reference:** `workers/jobs/scheduled/common/pending_upload_cleanup.go` handles `StorageFileNotFound` gracefully.

---

## Using Storage in Background Processes

Background processes commonly generate files (PDFs, CSVs) and upload them to storage. The typical flow:

```
1. Parse payload         (10%)
2. Fetch data            (30%)
3. Generate file         (60%)
4. Upload to storage     (80%)
5. Create file record    (90%)
6. Generate download URL (95%)
7. Complete              (100%)
```

**Reference:** `workers/tasks/invoice_pdf.go` implements this exact pattern.

```go
// Upload generated file
err = storageClient.Upload(ctx, bucket, storagePath, pdfBytes, "application/pdf")
if err != nil {
    // Retryable — return error to let Asynq retry
    return err
}

// Generate signed download URL for the output
downloadURL, err := storageClient.GetSignedDownloadURL(ctx, bucket, storagePath, 24*time.Hour)
if err != nil {
    logger.Warnw("Failed to generate download URL", "error", err)
    // Non-critical — complete without URL
}

// Store URL in process output
output := map[string]interface{}{
    "file_path":    storagePath,
    "download_url": downloadURL,
}
```

---

## Local Development with MinIO

### Docker Compose setup

MinIO runs as two Docker services in `docker-compose.yaml`:

- **`industrix-minio`** — the S3-compatible server
  - API: `http://localhost:9000`
  - Web console: `http://localhost:9002`
  - Default credentials: `minioadmin` / `changeme_minio_dev`
- **`industrix-minio-init`** — one-shot bucket provisioner that auto-creates the default bucket

### Accessing MinIO console

1. Open `http://localhost:9002` in your browser
2. Log in with `minioadmin` / `changeme_minio_dev`
3. Navigate to the `documents` bucket to see uploaded files

### Presigned URL endpoints (the two-endpoint problem)

In Docker, containers talk to MinIO via its Docker DNS name (`industrix-minio:9000`). But the browser can't reach that hostname — it needs `localhost:9000`. If presigned URLs pointed to `industrix-minio:9000`, the user's browser would get a DNS resolution failure when trying to upload or download files.

To solve this, the S3 client maintains **two separate presign clients**: one using the internal endpoint (for server-side operations like `Upload` and `Download`) and one using the public endpoint (for browser-facing URLs from `GetSignedURL` and `GetSignedDownloadURL`). The public presign client derives its SSL setting from the URL scheme — the internal endpoint might be HTTP while the public one is HTTPS behind a reverse proxy.

| Purpose | Endpoint | Used by |
|---------|----------|---------|
| Internal (container-to-container) | `http://industrix-minio:9000` | `Upload`, `Download`, `Delete`, `Exists` |
| Public (browser-accessible) | `http://localhost:9000` | `GetSignedURL`, `GetSignedDownloadURL` |

This is configured via `S3_ENDPOINT` (internal) and `S3_PUBLIC_ENDPOINT` (public) environment variables. If `S3_PUBLIC_ENDPOINT` is not set, the client falls back to using the internal endpoint for presigned URLs as well.

### Environment variables (already set in docker-compose.yaml)

```bash
STORAGE_BACKEND=s3
S3_ENDPOINT=http://industrix-minio:9000
S3_PUBLIC_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=changeme_minio_dev
S3_BUCKET=documents
S3_REGION=us-east-1
S3_USE_SSL=false
```

### Starting MinIO

```bash
docker compose up -d industrix-minio
```

The `industrix-minio-init` service will auto-create the bucket.

---

## Testing

### Unit tests with mock client

The storage client has a generated mock at `pkg/common/storage/mocks/Client.go`.

```go
import storagemocks "github.com/industrix-id/backend/pkg/common/storage/mocks"

func TestUploadFile(t *testing.T) {
    mockStorage := storagemocks.NewClient(t)

    // Expect upload call
    mockStorage.On("Upload",
        mock.Anything,          // ctx
        "",                     // bucket (empty = default)
        "invoices/123/doc.pdf", // path
        mock.Anything,          // data
        "application/pdf",      // contentType
    ).Return(nil)

    // ... test your service method ...

    mockStorage.AssertExpectations(t)
}
```

### Testing error scenarios

```go
func TestUploadFile_StorageFails(t *testing.T) {
    mockStorage := storagemocks.NewClient(t)

    mockStorage.On("Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
        Return(common.NewCustomError("upload failed").
            WithErrorCode(errorcodes.StorageUploadFailed))

    // ... verify your service handles the error correctly ...
}
```

### Integration tests (with real MinIO)

Integration tests that need actual storage use the `integration` or `integration_all` build tags and expect MinIO to be running:

```go
//go:build integration || integration_all

func TestS3ClientIntegration(t *testing.T) {
    cfg := &storage.S3Config{
        Endpoint:  "http://localhost:9000",
        AccessKey: "minioadmin",
        SecretKey: "changeme_minio_dev",
        Bucket:    "test-bucket",
        Region:    "us-east-1",
    }
    client, err := storage.NewS3Client(cfg)
    require.NoError(t, err)

    // Test upload
    err = client.Upload(ctx, "", "test/file.txt", []byte("hello"), "text/plain")
    require.NoError(t, err)

    // Test download
    data, err := client.Download(ctx, "", "test/file.txt")
    require.NoError(t, err)
    assert.Equal(t, "hello", string(data))

    // Cleanup
    _ = client.Delete(ctx, "", "test/file.txt")
}
```

---

## Related Guides

- [Storage Backend Configuration](../deployment/storage-configuration.md) — switching between MinIO, Supabase, and AWS S3
- [How to Create a Background Process](../workers/how-to-create-a-background-process.md) — background processes that produce file artifacts
- [How to Write Service Layer](./how-to-write-service-layer.md) — service patterns and dependency injection
