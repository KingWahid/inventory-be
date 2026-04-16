# infra
Infrastruktur proyek (CI, database, gateway, runtime config).

PostgreSQL bootstrap script ada di `infra/postgres/init.sql` dan dijalankan otomatis hanya saat inisialisasi pertama (volume data masih kosong) melalui `/docker-entrypoint-initdb.d`.
