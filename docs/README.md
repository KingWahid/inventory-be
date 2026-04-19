# Backend Docs Index

Dokumen di folder ini adalah panduan implementasi. Untuk urutan kerja proyek inventory, checklist utama tetap di `BACKEND_TASKS.md`. Untuk tugas UI (Next.js), lihat [`frontend/FRONTEND_TASKS.md`](../../frontend/FRONTEND_TASKS.md).

## Catatan pola repo ini

- Repo ini memakai **monorepo backend** (`backend/go.mod`) + pola `services/<service>/cmd/server` + uber fx.
- Beberapa contoh docs memakai pola template lain (mis. `go.mod` per service, namespace `industrix-id`).
- Jika ada benturan, ikuti **Pol layanan HTTP** di `BACKEND_TASKS.md`, lalu gunakan docs ini untuk detail implementasi layer (handler, service, repository, event, worker).

## Service Guides

- `service/how-to-create-a-service.md`
- `service/how-to-use-configuration.md`
- `service/how-to-structure-openapi.md`
- `service/how-to-write-handlers.md`
- `service/how-to-write-service-layer.md`
- `service/how-to-write-repositories.md`
- `service/how-to-use-transactions.md`
- `service/how-to-use-events.md`
- `service/how-to-implement-caching.md`
- `service/how-to-implement-sorting.md`
- `service/how-to-use-storage.md`
- `service/how-to-use-audit-logging.md`
- `service/how-to-understand-architecture.md`

## Repository Guides

- `repository/how-to-create-a-repository.md`
- `repository/how-to-understand-repositories.md`
- `repository/how-to-implement-queries.md`
- `repository/how-to-handle-errors.md`
- `repository/how-to-handle-transactions.md`
- `repository/how-to-invalidate-caches.md`
- `repository/how-to-use-utilities.md`

## Workers Guides

- `workers/how-to-create-a-background-process.md`
- `workers/how-to-create-a-scheduled-job.md`
