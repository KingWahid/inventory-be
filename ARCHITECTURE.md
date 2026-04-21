# Inventory Management System — Architecture Document

> **Version:** 1.0.0
> **Last Updated:** 2026-04-18
> **Status:** Draft
> **Language:** English — Indonesian version: [ARCHITECTURE.md](ARCHITECTURE.md)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Tech Stack](#2-tech-stack)
3. [High-Level Architecture](#3-high-level-architecture)
4. [Project Structure](#4-project-structure) — [§4.1 Docs ↔ repo mapping](#41-documentation-pattern--repository-mapping)
5. [Layered Architecture](#5-layered-architecture)
6. [Multi-Tenancy Strategy](#6-multi-tenancy-strategy)
7. [Authentication & Authorization](#7-authentication--authorization)
8. [Database Design](#8-database-design)
9. [API Design](#9-api-design)
10. [Event-Driven Architecture](#10-event-driven-architecture)
11. [Realtime Dashboard](#11-realtime-dashboard)
12. [Job Queue & Automation](#12-job-queue--automation)
13. [Caching Strategy](#13-caching-strategy)
14. [Logging & Audit Trail](#14-logging--audit-trail)
15. [Error Handling](#15-error-handling)
16. [Deployment Architecture](#16-deployment-architecture)
17. [Implementation Roadmap](#17-implementation-roadmap) — [§17.1 Quality gates](#171-quality-gates-and-development-workflow)

---

## 1. Overview

**Repository layout:** Go monorepo — **`infra/`** (config & migrations), **`pkg/`** (shared libraries), **`services/`** (microservices), **`workers/`** (background); **all UI (Next.js)** under **`frontend/`** — see [§4](#4-project-structure).

**Microservice-ready:** production target is **Kong** plus split containers (`inventory-api`, `notification-service`, workers) — see [§3.1](#31-microservice-ready-topology-kong--docker).

A full-stack, multi-tenant inventory management system covering the full item lifecycle: master data (products, categories, warehouses), stock movements (inbound, outbound, inter-warehouse transfers), a realtime dashboard, automated notifications, and a complete audit trail.

### Design Principles

- **Stock Consistency First** — Every stock change MUST go through a movement (no ad-hoc stock updates). Stock balance and movement rows are committed in a single DB transaction.
- **Modular Monolith, Event-Ready** — A single **`services/inventory`** service (**inventory-api**) in the first phase with clear modules (catalog, warehouse, movement, audit). Other services (`notification`, `authentication`, …) have their own folders and can be rolled out gradually using the same event boundaries.
- **Outbox Pattern** — Events are published to Redis Streams after the DB commit (not before), avoiding inconsistency between DB state and the event bus.
- **Tenant Isolation** — All domain data is scoped by `tenant_id`; there is no cross-tenant access.

---

## 2. Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| API Gateway | Kong | Routing, rate limiting, JWT validation, CORS, TLS termination |
| Backend | Go (Echo) | HTTP server, SSE/WebSocket, middleware |
| Dependency Injection | Uber fx | Module wiring, lifecycle management |
| ORM | GORM | Database queries, migrations, transactions |
| Database | PostgreSQL | Primary data store, ACID transactions |
| Cache | Redis | Read-through cache, TTL-based invalidation |
| Event Bus | Redis Streams | Async event processing, consumer groups |
| Realtime Push | Redis Pub/Sub | Server-to-client push via SSE |
| Logger | Zap | Structured logging with correlation IDs |
| Auth | JWT | Stateless authentication, tenant scoping |
| API Spec | OpenAPI 3.x | Contract-first API design |
| Frontend | Next.js (App Router) | Server components, SSE client, dashboard UI |

---

## 3. High-Level Architecture

```mermaid
flowchart TB
  subgraph clients [Client Layer]
    Browser[Browser]
    Mobile[Mobile App future]
  end

  subgraph gateway [API Gateway]
    Kong[Kong]
  end

  subgraph app [Application Layer - services/inventory]
    direction TB
    Echo[Echo HTTP Server]

    subgraph modules [Domain Modules]
      Catalog[Catalog Module]
      Warehouse[Warehouse Module]
      Movement[Movement Module]
      Audit[Audit Module]
      Auth[Auth Module]
    end

    subgraph layers [Layered Architecture]
      Handler[Handler Layer]
      Usecase[Usecase Layer]
      Repository[Repository Layer]
    end
  end

  subgraph data [Data Layer]
    PG[(PostgreSQL)]
    RedisCache[(Redis Cache)]
    RedisStream[(Redis Streams)]
    RedisPubSub[(Redis Pub/Sub)]
  end

  subgraph workers [Worker Layer]
    AlertWorker[Alert Worker]
    ReportWorker[Report Worker]
    SyncWorker[Sync Worker]
    OutboxRelay[Outbox Relay]
  end

  subgraph external [External]
    Email[Email Service]
    S3[Object Storage]
    ERP[ERP / 3rd Party]
  end

  Browser --> Kong
  Mobile --> Kong
  Kong --> Echo
  Echo --> Handler --> Usecase --> Repository --> PG
  Usecase --> RedisCache
  Usecase --> RedisPubSub
  OutboxRelay --> PG
  OutboxRelay --> RedisStream
  AlertWorker --> RedisStream
  ReportWorker --> RedisStream
  SyncWorker --> RedisStream
  AlertWorker --> Email
  ReportWorker --> S3
  SyncWorker --> ERP
  RedisPubSub --> Echo
```

### Data Flow Summary

```
Client Request
  → Kong (JWT validation, rate limit)
    → Echo Handler (bind, validate)
      → Usecase (business rules)
        → Repository (DB transaction: movement + stock + outbox)
      ← Response to client
  
Async (background):
  Outbox Relay → polls outbox table → XADD to Redis Stream
  Workers → XREADGROUP from Redis Stream → process (alert/report/sync)
  
Realtime:
  Usecase → PUBLISH to Redis Pub/Sub channel
  Echo SSE handler → SUBSCRIBE → push to connected clients
```

### 3.1 Microservice-ready topology (Kong & Docker)

This design **splits into multiple containers** without changing core business rules: **stock and movement** stay in a single **`inventory-api`** service (single DB transaction). Other services **only** consume events or run side effects (notifications, reports, sync).

**Principles**

| Principle | Meaning |
|-----------|---------|
| **Single gateway** | Clients (browser/mobile) talk only to **Kong** (TLS, rate limit, routing); workers are not called directly. |
| **Service boundaries** | `inventory-api` = stock domain + auth + HTTP audit; **notification-service** = email/push/webhook from events; **worker-*** = outbox relay + stream consumers (alerts, reports, sync). |
| **Inter-service communication** | Primarily **Redis Streams** (and Pub/Sub for realtime UI); avoid synchronous HTTP chains for stock operations. |
| **Scaling** | Horizontal replicas: stateless `inventory-api` + JWT; workers per **consumer group**; notification-service can scale independently. |
| **Evolution** | The **`services/*`** tree can use **one Dockerfile per service** or a shared image with different `CMD`. |

```mermaid
flowchart TB
  subgraph clients [Clients]
    FE[frontend Next.js]
  end

  subgraph gateway [Edge]
    Kong[Kong API Gateway]
  end

  subgraph core [Stock domain - single bounded context]
    InvAPI[inventory-api]
  end

  subgraph sidecar [Side services - event-driven]
    Notif[notification-service]
    Relay[worker-outbox-relay]
    WAlert[worker-alerts]
    WReport[worker-reports]
    WSync[worker-sync]
  end

  subgraph data [Data]
    PG[(PostgreSQL)]
    Redis[(Redis Streams + Pub/Sub)]
  end

  FE --> Kong
  Kong -->|/api/v1/inventory| InvAPI
  Kong -->|/api/v1/notifications| Notif
  InvAPI --> PG
  InvAPI --> Redis
  Relay --> PG
  Relay --> Redis
  Notif --> Redis
  WAlert --> Redis
  WReport --> Redis
  WReport --> PG
  WSync --> Redis
  WSync --> PG
  Notif --> ExtN[Email Push Webhook]
```

**Example Kong routes (declarative)**

| Path prefix | Upstream | Notes |
|-------------|----------|--------|
| `/api/v1/inventory` | `inventory-api:8080` | Products, movements, stock, audit, stock SSE |
| `/api/v1/notifications` | `notification-service:8081` | Optional: channel prefs, webhook test, health — **not** the stock source of truth |
| `/` | `frontend:3000` | Only if Kong also proxies the UI; production often uses a separate CDN/domain |

Common Kong plugins: **JWT** (or **OIDC**), **rate-limiting**, **CORS**, **request-id**, **prometheus**.

---

## 4. Project Structure

Monorepo: **infrastructure & migrations** in `infra/`, **shared libraries** in `pkg/`, **HTTP services** in `services/`, **background processes** in `workers/`, **UI** in `frontend/`. Matches the common **infra + pkg + services + workers** layout for team/CI splits. Multiple images from one root `go.mod` (or `go.work`) — see [§3.1](#31-microservice-ready-topology-kong--docker) and [§16](#16-deployment-architecture).

**Path convention in this repo:** Go sources and backend compose live under **`backend/`** (module **`github.com/KingWahid/inventory/backend`** — `backend/go.mod`). Trees below use paths relative to **`backend/`**; `frontend/` may live at the monorepo root (`../frontend/` from `backend/`).

### Responsibility split

| Folder | Owns | Does not own |
|--------|------|----------------|
| **`infra/`** | Compose, SQL migrations, seeds, Kong declarative ([`infra/kong/kong.yml`](infra/kong/kong.yml)), Postgres / Redis | Domain business logic |
| **`pkg/`** | `common` (JWT, validation, errorcodes, logger), `database/base`, `database/transaction`, `eventbus`, etc. | HTTP-specific rules for one service |
| **`services/*`** | One folder per service: Echo, OpenAPI, stub codegen, fx wiring; inventory domains under **`domains/<name>/handler|usecase|repository`** | React pages |
| **`workers/`** | Outbox relay, stream consumers, cron (as implemented) | Browser-facing REST |
| **`frontend/`** | Next.js, `fetch`/SSE | Stock rules & DB |

Clients talk to **Kong**; services communicate via **Redis Streams** / internal HTTP. Frontend uses `NEXT_PUBLIC_API_URL` to the gateway.

### 4.1 Documentation pattern ↔ repository mapping

```
backend/                                 # go.mod: github.com/KingWahid/inventory/backend
├── infra/
│   ├── database/
│   │   ├── migrations/
│   │   └── cmd/seed/
│   ├── kong/
│   │   └── kong.yml                     # Declarative Kong (not kong.template.yml)
│   ├── postgres/
│   └── redis/
├── pkg/
│   ├── common/
│   ├── database/
│   │   ├── base/
│   │   ├── transaction/
│   │   └── schemas/
│   └── eventbus/
├── services/
│   ├── authentication/
│   │   ├── cmd/server/main.go
│   │   ├── api/
│   │   ├── config/
│   │   ├── fx/
│   │   ├── openapi/
│   │   ├── repository/
│   │   ├── service/
│   │   └── stub/
│   ├── inventory/
│   │   ├── cmd/server/main.go
│   │   ├── api/
│   │   ├── config/
│   │   ├── fx/
│   │   ├── service/
│   │   ├── openapi/
│   │   ├── stub/
│   │   └── domains/
│   │       ├── catalog/
│   │       ├── warehouse/
│   │       ├── movement/
│   │       ├── audit/
│   │       └── auth/
│   │           ├── module.go
│   │           ├── handler/
│   │           ├── usecase/
│   │           └── repository/
│   │   ├── docker/Dockerfile
│   │   └── air.toml
│   └── notification/
├── workers/
├── Makefile
├── ARCHITECTURE.md
└── docs/
```

**Notes**

- **Stock domain & transactional consistency** remain in **`services/inventory`** (movement + balance + outbox in one DB transaction).
- **Gateway:** route source of truth for deploy is **`infra/kong/kong.yml`** — keep in sync with §9 and compose in §16.
- **Migrations** under **`infra/database/migrations`**; run via Makefile / CI targets.

This supports **per-service CI** and **future repo extraction** without changing public API contracts.

---

## 5. Layered Architecture

Each domain module follows three strict layers:

```mermaid
flowchart TB
  subgraph handlerLayer [Handler Layer]
    H1[Bind request body/params]
    H2[Call usecase]
    H3[Return HTTP response]
  end

  subgraph usecaseLayer [Usecase Layer]
    U1[Validate business rules]
    U2[Orchestrate repositories]
    U3[Manage transactions]
    U4[Publish events]
    U5[Invalidate cache]
  end

  subgraph repoLayer [Repository Layer]
    R1[GORM queries]
    R2[Raw SQL if needed]
    R3[Implement interfaces defined in usecase]
  end

  H1 --> H2 --> H3
  H2 --> U1 --> U2 --> U3
  U3 --> U4
  U3 --> U5
  U2 --> R1
  U2 --> R2
```

### Rules

| Layer | Can Depend On | Cannot Depend On | Responsibilities |
|-------|--------------|------------------|-----------------|
| **Handler** | Usecase interfaces | Repository, GORM, Redis directly | Request binding, response formatting, HTTP status codes |
| **Usecase** | Repository interfaces, domain entities, cache/stream interfaces | Echo, HTTP concepts | Business rules, transaction orchestration, event publishing |
| **Repository** | GORM, domain entities | Usecase, Handler | Data access, query building, implements interfaces |

### Dependency Injection with fx

```go
// services/inventory/domains/catalog/module.go — catalog domain module
var Module = fx.Module("catalog",
    fx.Provide(
        repository.New,
        usecase.New,
        handler.New,
    ),
)

// services/inventory/cmd/server/main.go — combines invfx.Module (config, postgres, redis, api, domain modules)
func main() {
    uberfx.New(
        uberfx.WithLogger(/* fxevent.ZapLogger */),
        invfx.Module, // services/inventory/fx — auth, audit, catalog, movement, warehouse, service, api
    ).Run()
}
```

---

## 6. Multi-Tenancy Strategy

### Approach: Column-Level Isolation (`tenant_id`)

All domain tables have a `tenant_id NOT NULL` column. One database, one schema, one connection pool.

```mermaid
flowchart LR
  JWT["JWT: { tenant_id: uuid }"] --> Middleware[Tenant Middleware]
  Middleware --> Context["ctx.Set('tenant_id', uuid)"]
  Context --> Usecase
  Usecase --> Repo["Repository: WHERE tenant_id = ?"]
  Repo --> DB[(PostgreSQL)]
```

### Implementation

```go
// services/inventory/api/middleware/tenant.go — extract tenant_id from JWT claims
func TenantMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            claims := c.Get("jwt_claims").(*JWTClaims)
            c.Set("tenant_id", claims.TenantID)
            return next(c)
        }
    }
}

// repository — every query scoped
func (r *productRepo) FindAll(ctx context.Context, tenantID uuid.UUID) ([]domain.Product, error) {
    var products []domain.Product
    err := r.db.WithContext(ctx).
        Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
        Find(&products).Error
    return products, err
}
```

### Tenant Table (global, not tenant-scoped)

```sql
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    settings    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## 7. Authentication & Authorization

### JWT Structure

```json
{
  "sub": "user-uuid",
  "tenant_id": "tenant-uuid",
  "role": "admin",
  "permissions": ["product:write", "movement:write", "report:read"],
  "exp": 1719878400,
  "iat": 1719792000
}
```

### Auth Flow

```mermaid
sequenceDiagram
  participant Client
  participant Kong
  participant AuthModule
  participant DB

  Client->>Kong: POST /api/v1/auth/login
  Kong->>AuthModule: Forward request
  AuthModule->>DB: Validate credentials
  DB-->>AuthModule: User + tenant info
  AuthModule-->>Kong: JWT access + refresh token
  Kong-->>Client: Set tokens

  Client->>Kong: GET /api/v1/products (Bearer token)
  Kong->>Kong: Validate JWT signature + expiry
  Kong->>AuthModule: Forward (valid token in header)
  AuthModule-->>Client: 200 OK + data
```

### User & Role Tables

```sql
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    email       VARCHAR(255) NOT NULL,
    password    VARCHAR(255) NOT NULL,  -- bcrypt hash
    full_name   VARCHAR(255) NOT NULL,
    role        VARCHAR(50) NOT NULL DEFAULT 'staff',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    last_login  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_users_tenant_email UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
```

Roles: `super_admin` (cross-tenant), `admin` (full tenant access), `manager` (movement + reports), `staff` (read + create movement).

---

## 8. Database Design

### Entity Relationship Diagram

```mermaid
erDiagram
    tenants ||--o{ users : has
    tenants ||--o{ products : has
    tenants ||--o{ categories : has
    tenants ||--o{ warehouses : has

    categories ||--o{ products : contains
    products ||--o{ stock_balances : tracked_in
    warehouses ||--o{ stock_balances : stores

    movements ||--o{ movement_lines : contains
    products ||--o{ movement_lines : referenced_by
    warehouses ||--o{ movements : source
    warehouses ||--o{ movements : destination
    users ||--o{ movements : created_by

    users ||--o{ audit_logs : performed_by

    tenants {
        uuid id PK
        varchar name
        varchar slug UK
        boolean is_active
        jsonb settings
    }

    users {
        uuid id PK
        uuid tenant_id FK
        varchar email
        varchar password
        varchar full_name
        varchar role
        boolean is_active
    }

    categories {
        uuid id PK
        uuid tenant_id FK
        varchar name
        varchar description
        uuid parent_id FK
        int sort_order
        timestamptz deleted_at
    }

    products {
        uuid id PK
        uuid tenant_id FK
        uuid category_id FK
        varchar sku UK
        varchar name
        text description
        varchar unit
        decimal price
        int reorder_level
        jsonb metadata
        timestamptz deleted_at
    }

    warehouses {
        uuid id PK
        uuid tenant_id FK
        varchar code UK
        varchar name
        varchar address
        boolean is_active
        timestamptz deleted_at
    }

    stock_balances {
        uuid id PK
        uuid tenant_id FK
        uuid warehouse_id FK
        uuid product_id FK
        int quantity
        int reserved_quantity
        timestamptz last_movement_at
    }

    movements {
        uuid id PK
        uuid tenant_id FK
        varchar type
        varchar reference_number UK
        uuid source_warehouse_id FK
        uuid destination_warehouse_id FK
        uuid created_by FK
        varchar status
        text notes
        varchar idempotency_key UK
        timestamptz created_at
    }

    movement_lines {
        uuid id PK
        uuid movement_id FK
        uuid product_id FK
        int quantity
        text notes
    }

    audit_logs {
        uuid id PK
        uuid tenant_id FK
        uuid user_id FK
        varchar action
        varchar entity
        uuid entity_id
        jsonb before_data
        jsonb after_data
        varchar ip_address
        varchar request_id
        timestamptz created_at
    }

    outbox_events {
        bigint id PK
        uuid tenant_id FK
        varchar event_type
        varchar aggregate_type
        uuid aggregate_id
        jsonb payload
        boolean published
        timestamptz created_at
        timestamptz published_at
    }
```

### Full DDL

#### `categories`

```sql
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    parent_id   UUID REFERENCES categories(id),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,

    CONSTRAINT uq_categories_tenant_name UNIQUE (tenant_id, name)
);

CREATE INDEX idx_categories_tenant ON categories(tenant_id);
CREATE INDEX idx_categories_parent ON categories(parent_id);
CREATE INDEX idx_categories_deleted ON categories(deleted_at) WHERE deleted_at IS NULL;
```

#### `products`

```sql
CREATE TABLE products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    category_id     UUID REFERENCES categories(id),
    sku             VARCHAR(100) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    unit            VARCHAR(50) NOT NULL DEFAULT 'pcs',
    price           DECIMAL(15,2) NOT NULL DEFAULT 0,
    reorder_level   INT NOT NULL DEFAULT 0,
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT uq_products_tenant_sku UNIQUE (tenant_id, sku)
);

CREATE INDEX idx_products_tenant ON products(tenant_id);
CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_tenant_name ON products(tenant_id, name);
CREATE INDEX idx_products_deleted ON products(deleted_at) WHERE deleted_at IS NULL;
```

#### `warehouses`

```sql
CREATE TABLE warehouses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    code        VARCHAR(50) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    address     TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,

    CONSTRAINT uq_warehouses_tenant_code UNIQUE (tenant_id, code)
);

CREATE INDEX idx_warehouses_tenant ON warehouses(tenant_id);
```

#### `stock_balances`

```sql
CREATE TABLE stock_balances (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    warehouse_id        UUID NOT NULL REFERENCES warehouses(id),
    product_id          UUID NOT NULL REFERENCES products(id),
    quantity            INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reserved_quantity   INT NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
    last_movement_at    TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_stock_tenant_warehouse_product UNIQUE (tenant_id, warehouse_id, product_id),
    CONSTRAINT chk_reserved_lte_quantity CHECK (reserved_quantity <= quantity)
);

CREATE INDEX idx_stock_tenant ON stock_balances(tenant_id);
CREATE INDEX idx_stock_warehouse ON stock_balances(warehouse_id);
CREATE INDEX idx_stock_product ON stock_balances(product_id);
CREATE INDEX idx_stock_low ON stock_balances(tenant_id, quantity) WHERE quantity > 0;
```

#### `movements`

```sql
CREATE TYPE movement_type AS ENUM ('inbound', 'outbound', 'transfer', 'adjustment');
CREATE TYPE movement_status AS ENUM ('draft', 'confirmed', 'cancelled');

CREATE TABLE movements (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL REFERENCES tenants(id),
    type                     movement_type NOT NULL,
    reference_number         VARCHAR(100) NOT NULL,
    source_warehouse_id      UUID REFERENCES warehouses(id),
    destination_warehouse_id UUID REFERENCES warehouses(id),
    created_by               UUID NOT NULL REFERENCES users(id),
    status                   movement_status NOT NULL DEFAULT 'draft',
    notes                    TEXT,
    idempotency_key          VARCHAR(255),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_movements_tenant_ref UNIQUE (tenant_id, reference_number),
    CONSTRAINT uq_movements_idempotency UNIQUE (tenant_id, idempotency_key),
    CONSTRAINT chk_movement_warehouses CHECK (
        (type = 'inbound'    AND source_warehouse_id IS NULL     AND destination_warehouse_id IS NOT NULL) OR
        (type = 'outbound'   AND source_warehouse_id IS NOT NULL AND destination_warehouse_id IS NULL) OR
        (type = 'transfer'   AND source_warehouse_id IS NOT NULL AND destination_warehouse_id IS NOT NULL
                             AND source_warehouse_id != destination_warehouse_id) OR
        (type = 'adjustment' AND (source_warehouse_id IS NOT NULL OR destination_warehouse_id IS NOT NULL))
    )
);

CREATE INDEX idx_movements_tenant ON movements(tenant_id);
CREATE INDEX idx_movements_type ON movements(tenant_id, type);
CREATE INDEX idx_movements_status ON movements(tenant_id, status);
CREATE INDEX idx_movements_created ON movements(tenant_id, created_at DESC);
CREATE INDEX idx_movements_source ON movements(source_warehouse_id) WHERE source_warehouse_id IS NOT NULL;
CREATE INDEX idx_movements_dest ON movements(destination_warehouse_id) WHERE destination_warehouse_id IS NOT NULL;
```

#### `movement_lines`

```sql
CREATE TABLE movement_lines (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    movement_id     UUID NOT NULL REFERENCES movements(id) ON DELETE CASCADE,
    product_id      UUID NOT NULL REFERENCES products(id),
    quantity        INT NOT NULL CHECK (quantity > 0),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_movement_lines_movement ON movement_lines(movement_id);
CREATE INDEX idx_movement_lines_product ON movement_lines(product_id);
```

#### `audit_logs`

```sql
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    user_id     UUID REFERENCES users(id),
    action      VARCHAR(50) NOT NULL,
    entity      VARCHAR(100) NOT NULL,
    entity_id   UUID NOT NULL,
    before_data JSONB,
    after_data  JSONB,
    ip_address  VARCHAR(45),
    user_agent  TEXT,
    request_id  VARCHAR(100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_entity ON audit_logs(tenant_id, entity, entity_id);
CREATE INDEX idx_audit_user ON audit_logs(tenant_id, user_id);
CREATE INDEX idx_audit_action ON audit_logs(tenant_id, action);
CREATE INDEX idx_audit_created ON audit_logs(tenant_id, created_at DESC);
```

#### `outbox_events`

```sql
CREATE TABLE outbox_events (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    event_type      VARCHAR(100) NOT NULL,
    aggregate_type  VARCHAR(100) NOT NULL,
    aggregate_id    UUID NOT NULL,
    payload         JSONB NOT NULL,
    published       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at    TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox_events(created_at)
    WHERE published = false;
CREATE INDEX idx_outbox_tenant ON outbox_events(tenant_id);
```

### Soft Delete Rules (Business Logic — enforced in Usecase layer)

```
Product soft-delete:
  1. Check stock_balances WHERE product_id = ? AND quantity > 0
     → If any exist: REJECT with "Cannot deactivate product with remaining stock"
  2. Otherwise: SET deleted_at = now()
  3. Product remains referenced by historical movements and audit_logs

Category soft-delete:
  1. Check products WHERE category_id = ? AND deleted_at IS NULL
     → If any active products: REJECT or reassign to "Uncategorized"
  2. Otherwise: SET deleted_at = now()

Warehouse soft-delete:
  1. Check stock_balances WHERE warehouse_id = ? AND quantity > 0
     → If any stock: REJECT
  2. Otherwise: SET is_active = false, deleted_at = now()
```

---

## 9. API Design

### Base URL & Versioning

```
https://api.inventory.example.com/api/v1
```

Kong routes follow **`infra/kong/kong.yml`**: typically **`/api/v1/inventory`** → **inventory-api** (`services/inventory`), **`/api/v1/notifications`** → **notification-service**. Auth routes (`/auth/...`) may be served by **`services/inventory`** initially or **`services/authentication`** after split — keep Kong in sync. JWT is required on protected routes except public login/register.

### Resource Endpoints

#### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Register new tenant + admin user |
| POST | `/auth/login` | Login, returns JWT pair |
| POST | `/auth/refresh` | Refresh access token |
| POST | `/auth/logout` | Invalidate refresh token |
| GET | `/auth/me` | Current user profile |

#### Products

| Method | Path | Description |
|--------|------|-------------|
| GET | `/products` | List products (paginated, filterable) |
| GET | `/products/:id` | Get single product |
| POST | `/products` | Create product |
| PUT | `/products/:id` | Update product |
| DELETE | `/products/:id` | Soft delete product |
| POST | `/products/:id/restore` | Restore soft-deleted product |

Query params: `?search=&category_id=&page=1&per_page=20&sort=name&order=asc`

#### Categories

| Method | Path | Description |
|--------|------|-------------|
| GET | `/categories` | List categories (tree or flat) |
| GET | `/categories/:id` | Get category with children |
| POST | `/categories` | Create category |
| PUT | `/categories/:id` | Update category |
| DELETE | `/categories/:id` | Soft delete category |

#### Warehouses

| Method | Path | Description |
|--------|------|-------------|
| GET | `/warehouses` | List warehouses |
| GET | `/warehouses/:id` | Get warehouse detail |
| POST | `/warehouses` | Create warehouse |
| PUT | `/warehouses/:id` | Update warehouse |
| DELETE | `/warehouses/:id` | Soft delete warehouse |

#### Stock

| Method | Path | Description |
|--------|------|-------------|
| GET | `/stock` | List all stock balances (filterable by warehouse, product) |
| GET | `/stock/summary` | Aggregated stock per product across warehouses |
| GET | `/stock/low` | Products below reorder level |
| GET | `/warehouses/:id/stock` | Stock in specific warehouse |

#### Movements

| Method | Path | Description |
|--------|------|-------------|
| GET | `/movements` | List movements (filterable, paginated) |
| GET | `/movements/:id` | Get movement with lines |
| POST | `/movements/inbound` | Record inbound (restock) |
| POST | `/movements/outbound` | Record outbound (sales) |
| POST | `/movements/transfer` | Transfer between warehouses |
| POST | `/movements/adjustment` | Stock adjustment (correction) |
| POST | `/movements/:id/confirm` | Confirm draft movement |
| POST | `/movements/:id/cancel` | Cancel draft movement |

Headers: `Idempotency-Key: <uuid>` (required for create operations)

#### Audit

| Method | Path | Description |
|--------|------|-------------|
| GET | `/audit-logs` | List audit logs (filterable) |
| GET | `/audit-logs/:entity/:id` | Audit history for specific entity |

#### Dashboard

| Method | Path | Description |
|--------|------|-------------|
| GET | `/dashboard/summary` | Total products, warehouses, movements today, low stock count |
| GET | `/dashboard/movements/chart` | Movement trends (daily/weekly/monthly) |
| GET | `/sse/stock` | SSE stream for realtime stock updates |

### Standard Response Format

```json
// Success (single)
{
  "success": true,
  "data": { ... },
  "meta": {
    "request_id": "req-uuid"
  }
}

// Success (list)
{
  "success": true,
  "data": [ ... ],
  "meta": {
    "request_id": "req-uuid",
    "pagination": {
      "page": 1,
      "per_page": 20,
      "total": 150,
      "total_pages": 8
    }
  }
}

// Error
{
  "success": false,
  "error": {
    "code": "PRODUCT_HAS_STOCK",
    "message": "Cannot delete product with remaining stock",
    "details": {
      "product_id": "uuid",
      "remaining_stock": 42
    }
  },
  "meta": {
    "request_id": "req-uuid"
  }
}
```

---

## 10. Event-Driven Architecture

### Outbox Pattern Flow

```mermaid
sequenceDiagram
  participant Usecase
  participant DB
  participant OutboxRelay
  participant Stream as Redis Streams
  participant Worker

  Usecase->>DB: BEGIN
  Usecase->>DB: INSERT movement + UPDATE stock_balances
  Usecase->>DB: INSERT audit_log
  Usecase->>DB: INSERT outbox_events (published=false)
  Usecase->>DB: COMMIT

  loop Every 500ms
    OutboxRelay->>DB: SELECT * FROM outbox_events WHERE published=false ORDER BY id LIMIT 100
    OutboxRelay->>Stream: XADD inventory.events * type StockChanged payload ...
    OutboxRelay->>DB: UPDATE outbox_events SET published=true, published_at=now()
  end

  Worker->>Stream: XREADGROUP GROUP alerts CONSUMER worker-1
  Worker->>Worker: Process event
  Worker->>Stream: XACK inventory.events alerts message-id
```

### Event Types

| Event | Trigger | Payload |
|-------|---------|---------|
| `StockChanged` | After any confirmed movement | `{ tenant_id, warehouse_id, product_id, old_qty, new_qty, movement_id }` |
| `StockBelowThreshold` | When new qty < reorder_level | `{ tenant_id, warehouse_id, product_id, current_qty, reorder_level }` |
| `MovementCreated` | New movement confirmed | `{ tenant_id, movement_id, type, reference_number, line_count }` |
| `ProductCreated` | New product | `{ tenant_id, product_id, sku, name }` |
| `ProductUpdated` | Product modified | `{ tenant_id, product_id, changed_fields }` |
| `ProductDeleted` | Product soft-deleted | `{ tenant_id, product_id, sku }` |

### Redis Streams Structure

```
Stream: inventory.events
  Consumer Groups:
    - alerts   → AlertWorker(s)
    - reports  → ReportWorker(s)
    - sync     → SyncWorker(s)
    - realtime → SSE broadcaster

Dead Letter: inventory.events.dead
  (messages that failed N retries)
```

---

## 11. Realtime Dashboard

### SSE (Server-Sent Events) Architecture

```mermaid
flowchart LR
  subgraph invSvc [services/inventory]
    Usecase --> PubSub[Redis PUBLISH]
    SSEHub[SSE Hub goroutine]
    SSEHub --> |SUBSCRIBE| PubSub
    SSEEndpoint[GET /sse/stock]
    SSEHub --> SSEEndpoint
  end

  subgraph frontend [Next.js]
    EventSource --> SSEEndpoint
    EventSource --> Dashboard[Dashboard UI]
  end
```

### Channel Design

```
Redis Pub/Sub channels:
  stock:{tenant_id}              → all stock changes for tenant
  stock:{tenant_id}:{warehouse}  → per-warehouse (optional granularity)
  alerts:{tenant_id}             → low stock alerts
```

### Client-Side (Next.js)

```typescript
// hooks/useStockSSE.ts
function useStockSSE(tenantId: string) {
  const [stocks, setStocks] = useState<StockUpdate[]>([]);

  useEffect(() => {
    const es = new EventSource(`/api/v1/sse/stock?token=${shortLivedToken}`);

    es.addEventListener('stock_changed', (e) => {
      const data = JSON.parse(e.data);
      setStocks(prev => mergeStockUpdate(prev, data));
    });

    es.addEventListener('low_stock_alert', (e) => {
      const data = JSON.parse(e.data);
      showNotification(`Low stock: ${data.product_name} (${data.current_qty})`);
    });

    return () => es.close();
  }, [tenantId]);

  return stocks;
}
```

---

## 12. Job Queue & Automation

### Worker Architecture

```mermaid
flowchart TB
  subgraph workerProcess [Worker Process workers/cmd/worker]
    Cron[Cron Scheduler]
    AlertConsumer[Alert Consumer Group]
    ReportConsumer[Report Consumer Group]
    SyncConsumer[Sync Consumer Group]
  end

  Stream[Redis Streams] --> AlertConsumer
  Stream --> ReportConsumer
  Stream --> SyncConsumer

  Cron --> |daily 00:00| ReportJob[Generate Daily Report]
  Cron --> |every 6h| SyncJob[Sync External Systems]

  AlertConsumer --> Email[Send Email/Notification]
  ReportConsumer --> S3[Upload Report to Storage]
  SyncConsumer --> ERP[Push to ERP API]
```

### Job Details

| Job | Type | Schedule/Trigger | Details |
|-----|------|-----------------|---------|
| **Restock Alert** | Event-driven | `StockBelowThreshold` event | Compare qty vs reorder_level; send notification; debounce (don't re-alert within 1h for same SKU) |
| **Daily Report** | Cron | `0 0 * * *` | Query movements of the day per tenant; generate summary CSV/PDF; store in S3 or email |
| **Data Sync** | Event + Cron | `StockChanged` + every 6h full sync | Push delta changes to external ERP/POS; full reconciliation on schedule |

### Retry & Dead Letter Queue

```
Max retries per message: 5
Backoff: exponential (1s, 2s, 4s, 8s, 16s)
After max retries: XADD to inventory.events.dead with original message + error info
DLQ messages: manual review via admin endpoint or periodic check
```

---

## 13. Caching Strategy

### What to Cache

| Data | Redis Key Pattern | TTL | Invalidation |
|------|------------------|-----|-------------|
| Product list (per tenant, page) | `cache:t:{tid}:products:p:{page}` | 5 min | On product create/update/delete |
| Single product | `cache:t:{tid}:product:{id}` | 10 min | On product update/delete |
| Category tree | `cache:t:{tid}:categories` | 15 min | On category change |
| Warehouse list | `cache:t:{tid}:warehouses` | 15 min | On warehouse change |
| Dashboard summary | `cache:t:{tid}:dashboard:summary` | 30 sec | On any movement confirm |
| Stock balance (per warehouse) | NOT cached (read from DB) | — | Realtime accuracy required |

### Cache Invalidation

```go
// usecase layer — after DB commit
func (u *catalogUsecase) UpdateProduct(ctx context.Context, tenantID uuid.UUID, req UpdateProductRequest) error {
    // ... DB update ...

    // invalidate related cache keys
    u.cache.Delete(ctx, fmt.Sprintf("cache:t:%s:product:%s", tenantID, req.ID))
    u.cache.DeletePattern(ctx, fmt.Sprintf("cache:t:%s:products:*", tenantID))

    return nil
}
```

Stock balances are **NOT cached** — always read from DB to guarantee accuracy. The dashboard summary uses a short TTL (30s) as a compromise.

---

## 14. Logging & Audit Trail

### Two Complementary Systems

```
1. Application Logs (Zap) → stdout/stderr → log aggregator (ELK, Loki, etc.)
   - Request/response lifecycle
   - Errors and panics
   - Performance metrics
   - NOT the source of truth for business audit

2. Audit Log Table (PostgreSQL) → queryable via API
   - WHO did WHAT to WHICH entity
   - Before/after snapshots
   - Immutable append-only
   - Source of truth for compliance
```

### Structured Log Fields (Zap)

```go
// Every log entry carries:
logger.Info("movement confirmed",
    zap.String("request_id", requestID),
    zap.String("tenant_id", tenantID.String()),
    zap.String("user_id", userID.String()),
    zap.String("movement_id", movement.ID.String()),
    zap.String("type", string(movement.Type)),
    zap.Int("line_count", len(movement.Lines)),
    zap.Duration("duration", elapsed),
)
```

### Audit Log Entry Example

```json
{
  "id": "audit-uuid",
  "tenant_id": "tenant-uuid",
  "user_id": "user-uuid",
  "action": "movement.confirm",
  "entity": "movement",
  "entity_id": "movement-uuid",
  "before_data": { "status": "draft" },
  "after_data": {
    "status": "confirmed",
    "lines": [
      { "product_id": "p1", "quantity": 100 },
      { "product_id": "p2", "quantity": 50 }
    ]
  },
  "ip_address": "192.168.1.100",
  "request_id": "req-uuid",
  "created_at": "2026-04-16T10:30:00Z"
}
```

---

## 15. Error Handling

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Request body/params invalid |
| `UNAUTHORIZED` | 401 | Missing or invalid JWT |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Entity not found (within tenant scope) |
| `CONFLICT` | 409 | Duplicate SKU, reference number, etc. |
| `IDEMPOTENCY_CONFLICT` | 409 | Idempotency key already processed |
| `PRODUCT_HAS_STOCK` | 422 | Cannot delete product with remaining stock |
| `WAREHOUSE_HAS_STOCK` | 422 | Cannot deactivate warehouse with stock |
| `INSUFFICIENT_STOCK` | 422 | Outbound/transfer qty exceeds available |
| `MOVEMENT_NOT_DRAFT` | 422 | Cannot modify confirmed/cancelled movement |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

### Domain Error Pattern (Go)

```go
// pkg/common/errorcodes or services/inventory/domains/<domain>/errors.go
type AppError struct {
    Code    string         `json:"code"`
    Message string         `json:"message"`
    Status  int            `json:"-"`
    Details map[string]any `json:"details,omitempty"`
}

var (
    ErrProductHasStock = &AppError{
        Code: "PRODUCT_HAS_STOCK", Message: "Cannot delete product with remaining stock", Status: 422,
    }
    ErrInsufficientStock = &AppError{
        Code: "INSUFFICIENT_STOCK", Message: "Outbound quantity exceeds available stock", Status: 422,
    }
)
```

---

## 16. Deployment Architecture

**Microservice-ready** environment: multiple small containers behind **Kong**; scale and restart each service independently.

```mermaid
flowchart TB
  subgraph edge [Edge]
    LB[Load Balancer]
    KongA[Kong]
  end

  subgraph inventory [Inventory domain]
    Inv1[inventory-api]
    Inv2[inventory-api replica]
  end

  subgraph notify [Notifications]
    N1[notification-service]
  end

  subgraph workers [Workers]
    R[worker-outbox-relay]
    A[worker-alerts]
    Rep[worker-reports]
    S[worker-sync]
  end

  subgraph data [Data plane]
    PG[(PostgreSQL)]
    Redis[(Redis)]
  end

  LB --> KongA
  KongA --> Inv1
  KongA --> Inv2
  KongA --> N1
  Inv1 --> PG
  Inv2 --> PG
  Inv1 --> Redis
  Inv2 --> Redis
  N1 --> Redis
  R --> PG
  R --> Redis
  A --> Redis
  Rep --> PG
  Rep --> Redis
  S --> Redis
  S --> PG
```

**Operational notes**

- **inventory-api** — stateless; horizontal scale; single PostgreSQL (add PgBouncer if needed).
- **notification-service** — does not change stock; idempotent delivery (e.g. Redis dedup) recommended.
- **worker-*** — can be one process in dev (`/app/worker --all`); split containers in production for independent **restart** and **resource limits**.
- **Kong** — single entry for `/api/v1/...`; health checks per upstream.

### Docker Compose (development — single worker process)

Example: build from **`services/inventory`**, **`services/notification`**, **`workers`**. One **`worker`** service runs outbox relay + alert/report/sync consumers in **one binary** (`--mode=all` or equivalent in `cmd/worker`) — enough for dev and small setups. `kong.yml` still defines upstreams `inventory-api`, `notification-service`.

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: inventory
      POSTGRES_USER: inventory
      POSTGRES_PASSWORD: secret
    ports: ["5432:5432"]
    volumes: ["pgdata:/var/lib/postgresql/data"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    command: redis-server --appendonly yes

  kong:
    image: kong:3.6
    environment:
      KONG_DATABASE: "off"
      KONG_DECLARATIVE_CONFIG: /kong/kong.yml
      KONG_PROXY_LISTEN: "0.0.0.0:8000"
    ports: ["8000:8000", "8001:8001"]
    volumes: ["./infra/kong:/kong"]
    depends_on: [inventory-api, notification-service]

  inventory-api:
    build: { context: ., dockerfile: services/inventory/docker/Dockerfile }
    command: ["/app/server"]
    environment:
      DB_DSN: "host=postgres user=inventory password=secret dbname=inventory sslmode=disable"
      REDIS_ADDR: "redis:6379"
      JWT_SECRET: "dev-secret-change-in-prod"
    expose: ["8080"]
    depends_on: [postgres, redis]

  notification-service:
    build: { context: ./services/notification, dockerfile: Dockerfile }
    command: ["/app/server"]
    environment:
      REDIS_ADDR: "redis:6379"
    expose: ["8081"]
    depends_on: [redis]

  worker:
    build: { context: ./workers, dockerfile: Dockerfile }
    command: ["/app/worker", "--mode=all"]
    environment:
      DB_DSN: "host=postgres user=inventory password=secret dbname=inventory sslmode=disable"
      REDIS_ADDR: "redis:6379"
    depends_on: [postgres, redis]

  frontend:
    build: { context: ./frontend, dockerfile: Dockerfile }
    ports: ["3000:3000"]
    environment:
      NEXT_PUBLIC_API_URL: "http://kong:8000/api/v1/inventory"

volumes:
  pgdata:
```

**Production / scale-out:** split into multiple containers with different `command` (`--mode=outbox-relay`, `--mode=alerts`, `--mode=reports`, `--mode=sync`) for independent **restart**, **CPU**, and **replicas** — or scale only the consumer that is the bottleneck.

**`infra/kong/kong.yml` (concept)** — upstream `inventory-api` → `inventory-api:8080`, route `paths: ["/api/v1/inventory"]`; upstream `notification-service:8081`, route `paths: ["/api/v1/notifications"]`.

---

## 17. Implementation Roadmap

Granular backend task checklists (setup → production) are owned by the team (issue board / internal docs). This architecture document does not pin a single roadmap file in the repo; stay aligned with product owners.

### 17.1 Quality gates and development workflow

| Step | Content |
|------|---------|
| **Vertical order** | DB schema & migrations → repository / data access → **usecase** (business rules) → HTTP handlers & OpenAPI → Kong config (`infra/kong/kong.yml`). |

### Phase 1 — Foundation (Backend Core)

1. Root **`go.mod`** + skeleton **`services/inventory`**, **`pkg/common`**, **`workers`** (fx, config, Zap)
2. PostgreSQL + GORM + migration runner (`infra/database/migrations`)
3. DB schema: all tables as defined in section 8
4. Auth module: register, login, JWT issue/verify, tenant middleware
5. Catalog module: CRUD products + categories (with soft delete rules)
6. Warehouse module: CRUD warehouses

### Phase 2 — Stock & Movement

7. Movement module: inbound, outbound, transfer, adjustment
8. Stock balance updates (transactional with movement)
9. Idempotency key handling
10. Audit log insertion on every mutation
11. Outbox table insertion within movement transaction

### Phase 3 — Event & Async

12. Redis connection + cache layer
13. Outbox relay (poll → XADD to Redis Streams)
14. Alert worker (StockBelowThreshold → notification)
15. Report worker (cron → daily summary)
16. OpenAPI spec finalization

### Phase 4 — Realtime & Frontend (`frontend/`)

17. SSE endpoint + Redis Pub/Sub subscription (in **`services/inventory`**)
18. Kong configuration (routing, JWT plugin, rate limit)
19. In **`frontend/`** (Next.js): auth pages, product/category/warehouse CRUD
20. In **`frontend/`**: movement forms (inbound, outbound, transfer)
21. In **`frontend/`**: dashboard with SSE realtime updates
22. In **`frontend/`**: audit log viewer

### Phase 5 — Polish & Production

23. Sync worker for external systems
24. **Microservice-ready deployment:** Kong declarative, containers for `inventory-api` / `notification-service` / **`worker`** (dev: single process `--mode=all`; prod: optionally split by `--mode`), per-service health checks
25. Monitoring, metrics (Prometheus), tracing (OpenTelemetry) per service
26. Load testing, index tuning
27. CI/CD pipeline (separate image per `cmd`), Docker production
