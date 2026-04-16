# infra
Infrastruktur proyek (CI, database, gateway, runtime config).

PostgreSQL bootstrap script ada di `infra/postgres/init.sql` dan dijalankan otomatis hanya saat inisialisasi pertama (volume data masih kosong) melalui `/docker-entrypoint-initdb.d`.

Migration workflow (goose via Makefile):
- Pastikan `DB_DSN` terisi.
- Buat file migration baru: `make migration-create NAME=create_tenants_users`
- Jalankan migration: `make up`
- Rollback migration: `make down`
- Lihat status migration: `make migration-status`
