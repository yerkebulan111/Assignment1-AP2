# AP2 Assignment 1,2 — gRPC Migration (Order & Payment Services)



---

##  Repository Links

| Repository | Purpose | Link                                                               |
|---|---|--------------------------------------------------------------------|
| **Protos repo** | Source `.proto` files (contract definitions) | [ap2-protos](https://github.com/yerkebulan111/ap-2_protos)         |
| **Generated repo** | Auto-generated `.pb.go` files via GitHub Actions | [ap2-protos-gen](https://github.com/yerkebulan111/ap-2_protos-gen) |
| **This repo** | Order & Payment service source code | [Assignment1-AP2](https://github.com/yerkebulan111/Assignment1-AP2.git)                                                |


---

## Architecture

The system migrated internal service-to-service communication from REST to **gRPC**, while keeping the external REST API (Gin) unchanged for end users.

```
┌─────────────────────────────────────────────────────────────────────┐
│                         End User / Client                           │
└────────────────────────────┬────────────────────────────────────────┘
                             │  REST (HTTP/JSON)
                             │  POST /orders
                             │  GET  /orders/:id
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        ORDER SERVICE  :8080                         │
│                                                                     │
│   ┌─────────────────┐    ┌──────────────────┐    ┌──────────────┐  │
│   │  Gin REST Layer │───▶│   Order UseCase  │───▶│  Order Repo  │  │
│   │   handler.go    │    │ order_usecase.go │    │ postgres.go  │  │
│   └─────────────────┘    └────────┬─────────┘    └──────────────┘  │
│                                   │ domain.PaymentClient interface  │
│                          ┌────────▼─────────┐                      │
│                          │ PaymentGRPCClient│  (replaces HTTP       │
│                          │ payment_grpc_    │   payment_client.go)  │
│                          │ client.go        │                       │
│                          └────────┬─────────┘                      │
│                                   │                                 │
│          ┌────────────────────────┼────────────────────┐           │
│          │         gRPC Server    │  :50052             │           │
│          │   SubscribeToOrder     │                     │           │
│          │   Updates (streaming)  │                     │           │
│          └────────────────────────┘                     │           │
└───────────────────────────┬──────────────────────────────┘
                            │  gRPC (Protocol Buffers)
                            │  ProcessPayment RPC
                            │  port :50051
                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       PAYMENT SERVICE  :8081                        │
│                                                                     │
│   ┌──────────────────┐   ┌───────────────────┐   ┌─────────────┐  │
│   │ PaymentGRPCServer│──▶│ Payment UseCase   │──▶│ Payment Repo│  │
│   │ + Interceptor    │   │payment_usecase.go │   │ postgres.go │  │
│   │ server.go        │   │                   │   │             │  │
│   └──────────────────┘   └───────────────────┘   └─────────────┘  │
│   (logs method+duration)                                            │
│                                                                     │
│   ┌──────────────────┐                                             │
│   │  Gin REST Layer  │  (kept for direct HTTP access if needed)    │
│   │   handler.go     │  port :8081                                 │
│   └──────────────────┘                                             │
└─────────────────────────────────────────────────────────────────────┘
         │                                       │
         ▼                                       ▼
  ┌─────────────┐                        ┌─────────────┐
  │  orders_db  │                        │ payment_db  │
  │ (PostgreSQL)│                        │ (PostgreSQL)│
  └─────────────┘                        └─────────────┘

── Contract-First Flow ──────────────────────────────────────────────
  [ap2-protos repo]  →  GitHub Actions  →  [ap2-protos-gen repo]
   payment.proto                             payment.pb.go
   order.proto          protoc compiler      payment_grpc.pb.go
                                             order.pb.go
                                             order_grpc.pb.go
                              ↓
                   go get github.com/yerkebulan111/ap2-protos-gen@v1.0.0
                   (imported by both services)
```

---

## How to Run

### Prerequisites

Make sure you have installed:
- Go 1.21+
- PostgreSQL (running locally or via Docker)
- `git`

### 1. Clone the repository

```bash
git clone https://github.com/yerkebulan111/Assignment1-AP2.git
cd Assignment1-AP2
```

### 2. Set up databases

```bash
# Create two databases in PostgreSQL
psql -U postgres -c "CREATE DATABASE orders_db;"
psql -U postgres -c "CREATE DATABASE payment_db;"

# Run migrations for Order Service
psql -U postgres -d orders_db -f order-service/migrations/1_create_orders_table.sql

# Run migrations for Payment Service
psql -U postgres -d payment_db -f payment-service/migrations/1_create_payments_table.sql
```

### 3. Configure environment variables

**`order-service/.env`**
```env
HTTP_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=orders_db
PAYMENT_GRPC_ADDR=localhost:50051
```

**`payment-service/.env`**
```env
SERVER_PORT=8081
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=payment_db
GRPC_PORT=50051
```

### 4. Install generated proto package

```bash
# In both service directories
cd order-service
go get github.com/yerkebulan111/ap-2_protos-gen@v1.0.0
go mod tidy

cd ../payment-service
go get github.com/yerkebulan111/ap-2_protos-gen@v1.0.0
go mod tidy
```

### 5. Start the services

Open **two terminals**:

**Terminal 1 — Payment Service (start first, it's the gRPC server):**
```bash
cd payment-service
go run ./cmd/payment-service/main.go

# Expected output:
# Connected to PostgreSQL successfully
# Payment HTTP listening on :8081
# Payment gRPC listening on :50051
```

**Terminal 2 — Order Service:**
```bash
cd order-service
go run ./cmd/order-service/main.go

# Expected output:
# Connected to PostgreSQL successfully
# Order Service listening on :8080
```

### 6. Test the API

**Create an order (triggers gRPC call to Payment Service):**
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "user-123", "item_name": "Laptop", "amount": 50000}'
```

**Get an order:**
```bash
curl http://localhost:8080/orders/{order-id}
```

**Cancel an order:**
```bash
curl -X PATCH http://localhost:8080/orders/{order-id}/cancel
```

**Test gRPC directly with grpcurl:**
```bash
# Process a payment via gRPC
grpcurl -plaintext -d '{"order_id": "test-123", "amount": 500}' \
  localhost:50051 payment.PaymentService/ProcessPayment

# List payments by amount range
grpcurl -plaintext -d '{"min_amount": 100, "max_amount": 5000}' \
  localhost:50051 payment.PaymentService/ListPayments
```

**Test order status streaming:**
```bash
grpcurl -plaintext -d '{"order_id": "your-order-id"}' \
  localhost:50052 order.OrderService/SubscribeToOrderUpdates
```

---

## Project Structure

```
.
├── order-service/
│   ├── cmd/order-service/
│   │   └── main.go                    # Entry point (uses gRPC client now)
│   ├── internal/
│   │   ├── app/app.go                 # Server wiring
│   │   ├── domain/
│   │   │   ├── order.go               # Order entity 
│   │   │   └── ports.go               # PaymentClient interface 
│   │   ├── repository/postgres.go     # DB layer 
│   │   ├── usecase/order_usecase.go   # Business logic 
│   │   └── transport/
│   │       ├── http/handler.go        # Gin REST handler 
│   │       └── grpc/
│   │           └── payment_client.go  # NEW — gRPC client (replaces HTTP client)
│   ├── proto/order.proto
│   ├── migrations/
│   └── .env
│
└── payment-service/
    ├── cmd/payment-service/
    │   └── main.go                    # Entry point (starts gRPC + HTTP servers)
    ├── internal/
    │   ├── app/app.go                 # HTTP router wiring 
    │   ├── domain/
    │   │   ├── payment.go             # Payment entity 
    │   │   └── repository.go          # Repository interface (+ FindByAmountRange)
    │   ├── repository/postgres.go     # DB layer (+ FindByAmountRange impl)
    │   ├── usecase/payment_usecase.go # Business logic (+ ListByAmountRange)
    │   └── transport/
    │       ├── http/handler.go        # Gin REST handler 
    │       └── grpc/
    │           ├── server.go          # NEW — gRPC server (ProcessPayment, ListPayments)
    │           └── interceptor.go     # NEW — logging interceptor 
    ├── proto/payment.proto
    ├── migrations/
    └── .env
```

---

## Evidence



---

## Contract-First Flow

```
1. Write .proto file in [ap-2_protos] repo
           │
           │  git push
           ▼
2. GitHub Actions triggers on [ap-2_protos-gen] repo
           │
           │  protoc compiles .proto → .pb.go files
           ▼
3. Generated .pb.go pushed to [ap-2_protos-gen] repo
           │
           │  Create release tag v1.0.0
           ▼
4. Services import via:
   go get github.com/yerkebulan111/ap-2_protos-gen@v1.0.0
```

---

## gRPC Services Defined

### PaymentService (`payment/payment.proto`)

| RPC | Type | Request | Response |
|---|---|---|---|
| `ProcessPayment` | Unary | `PaymentRequest` | `PaymentResponse` |
| `ListPayments` | Unary | `ListPaymentsRequest` | `ListPaymentsResponse` |

### OrderService (`order/order.proto`)

| RPC | Type | Request | Response |
|---|---|---|---|
| `SubscribeToOrderUpdates` | Server-side streaming | `OrderRequest` | `stream OrderStatusUpdate` |

---

## gRPC Interceptor

Payment Service includes a **UnaryServerInterceptor** that logs every incoming gRPC call:

```
[gRPC] method=/payment.PaymentService/ProcessPayment duration=2.3ms err=<nil>
[gRPC] method=/payment.PaymentService/ListPayments   duration=1.1ms err=<nil>
```

Implemented in `payment-service/internal/transport/grpc/interceptor.go`.