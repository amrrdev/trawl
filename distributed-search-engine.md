# SearchFlow - Distributed Search Engine

## 🎯 Project Overview

SearchFlow is a distributed, scalable search engine built from scratch in Go. It provides fast full-text search capabilities across millions of documents with features like ranking, autocomplete, and faceted search - similar to Elasticsearch but simpler and more educational.

---

## 📋 Table of Contents

- [Core Concept](#core-concept)
- [System Architecture](#system-architecture)
- [Key Components](#key-components)
- [Technology Stack](#technology-stack)
- [Features](#features)
- [Data Flow](#data-flow)
- [Database Schema](#database-schema)
- [API Endpoints](#api-endpoints)
- [Algorithms & Techniques](#algorithms--techniques)
- [Scalability & Distribution](#scalability--distribution)
- [Project Boundaries](#project-boundaries)
- [Learning Outcomes](#learning-outcomes)
- [Implementation Phases](#implementation-phases)

---

## 🧠 Core Concept

### The Library Analogy

Imagine a massive library with millions of books. Instead of reading every book to find information (slow!), we create a detailed index:

- **Traditional Search**: Read every document sequentially → O(n) time
- **SearchFlow**: Query pre-built inverted index → O(log n) time

### What is an Inverted Index?

Instead of: `Document → Words it contains`

We store: `Word → All documents containing it`

**Example:**

```
Document 1: "Golang is great for APIs"
Document 2: "Building APIs with Node.js"
Document 3: "Golang microservices architecture"

Inverted Index:
├─ "golang"    → [Doc1, Doc3]
├─ "apis"      → [Doc1, Doc2]
├─ "building"  → [Doc2]
├─ "node.js"   → [Doc2]
└─ "microservices" → [Doc3]
```

When user searches "golang apis", we:

1. Find docs containing "golang": [Doc1, Doc3]
2. Find docs containing "apis": [Doc1, Doc2]
3. Intersection: [Doc1] ← Contains BOTH words
4. Return Doc1 as the best match

---

## 🏗️ System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLIENT LAYER                         │
│  (Web UI, Mobile App, API Consumers)                        │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                    NGINX (Load Balancer)                     │
│  - Reverse proxy                                             │
│  - SSL termination                                           │
│  - Rate limiting                                             │
│  - Request routing                                           │
└────────────────────────┬────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┬──────────────┐
         ↓               ↓               ↓              ↓
┌─────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐
│    AUTH     │  │   INDEXING   │  │    SEARCH    │  │  ANALYTICS  │
│   SERVICE   │  │   SERVICE    │  │   SERVICE    │  │   SERVICE   │
│             │  │              │  │              │  │             │
│ - JWT auth  │  │ - Parse docs │  │ - Query      │  │ - Track     │
│ - Token     │  │ - Tokenize   │  │   processing │  │   searches  │
│   validation│  │ - Build      │  │ - Ranking    │  │ - Generate  │
│ - User mgmt │  │   index      │  │ - Result     │  │   reports   │
└──────┬──────┘  └──────┬───────┘  │   merging    │  └──────┬──────┘
       │                │          └──────┬───────┘         │
       │                │                 │                 │
       │                ↓                 │                 │
       │         ┌─────────────┐          │                 │
       │         │  RabbitMQ   │          │                 │
       │         │             │          │                 │
       │         │ - Indexing  │          │                 │
       │         │   queue     │          │                 │
       │         │ - Async     │          │                 │
       │         │   tasks     │          │                 │
       │         └──────┬──────┘          │                 │
       │                │                 │                 │
       └────────────────┼─────────────────┼─────────────────┘
                        │                 │
         ┌──────────────┼─────────────────┼─────────────┐
         ↓              ↓                 ↓             ↓
┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  ScyllaDB   │  │  ScyllaDB   │  │  ScyllaDB   │  │  PostgreSQL │
│   Shard 1   │  │   Shard 2   │  │   Shard 3   │  │             │
│             │  │             │  │             │  │ - Users     │
│ Words: A-H  │  │ Words: I-P  │  │ Words: Q-Z  │  │ - API keys  │
│             │  │             │  │             │  │ - Auth data │
│ (Replicated)│  │ (Replicated)│  │ (Replicated)│  │             │
└─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘
                        │
                        ↓
              ┌──────────────────┐
              │      MinIO       │
              │ (Object Storage) │
              │                  │
              │ - Original docs  │
              │ - PDFs, JSON,    │
              │   text files     │
              └──────────────────┘
```

---

## 🔧 Key Components

### 1. **Nginx (Load Balancer & Reverse Proxy)**

**Responsibilities:**

- Routes requests to appropriate services
- SSL/TLS termination
- Rate limiting per IP/user
- Request/response compression
- Static file serving (UI)
- Health check endpoints

**Configuration Example:**

```nginx
upstream auth_service {
    server auth:8081;
    server auth:8082;
}

upstream search_service {
    server search:8083;
    server search:8084;
    server search:8085;
}

upstream indexer_service {
    server indexer:8086;
}

server {
    listen 80;

    location /api/v1/auth {
        proxy_pass http://auth_service;
    }

    location /api/v1/search {
        proxy_pass http://search_service;
    }

    location /api/v1/documents {
        proxy_pass http://indexer_service;
    }
}
```

---

### 2. **Auth Service**

**Responsibilities:**

- User registration/login
- JWT token generation and validation
- API key management
- Role-based access control (RBAC)

**Tech Stack:** Go + PostgreSQL

**Endpoints:**

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
GET    /api/v1/auth/validate
DELETE /api/v1/auth/logout
```

**JWT Flow:**

```
1. User logs in → Auth service validates
2. Return JWT token (expires in 1 hour)
3. Client includes token in headers: Authorization: Bearer <token>
4. All services validate token via Auth service or shared secret
```

---

### 3. **Indexing Service**

**Responsibilities:**

- Accept document uploads
- Extract text from various formats
- Publish indexing jobs to RabbitMQ
- Process async indexing tasks (worker mode)

**Process Flow:**

```
Document Upload (API)
    ↓
Validate & Store in MinIO
    ↓
Publish message to RabbitMQ
    ↓
Return immediately to user (202 Accepted)

--- Async Processing ---

Worker consumes from RabbitMQ
    ↓
Extract text
    ↓
Tokenize & build index
    ↓
Store in ScyllaDB
    ↓
Update document status
```

**Input:**

- Document ID: `uuid`
- Document content: `binary/text`
- Metadata: `{ title, author, type, tags }`

**Output:**

- Inverted index entries in ScyllaDB
- Document stored in MinIO
- Job status: `queued/processing/completed/failed`

---

### 4. **RabbitMQ (Message Broker)**

**Why RabbitMQ?**

- ✅ Simpler than Kafka for this use case
- ✅ Built-in retry and dead-letter queues
- ✅ Better for job queues (vs Kafka's event streaming)
- ✅ Lower operational overhead
- ✅ Excellent Go client library

**Queues:**

```
┌──────────────────────────┐
│  indexing_queue          │
│                          │
│  - Document indexing     │
│  - Priority: Normal      │
│  - TTL: 1 hour           │
│  - Prefetch: 10          │
└──────────────────────────┘

┌──────────────────────────┐
│  indexing_queue_dlq      │
│  (Dead Letter Queue)     │
│                          │
│  - Failed jobs after 3   │
│    retries               │
│  - Manual intervention   │
└──────────────────────────┘

┌──────────────────────────┐
│  analytics_queue         │
│                          │
│  - Search tracking       │
│  - Low priority          │
│  - Batch processing      │
└──────────────────────────┘
```

**Message Format:**

```json
{
  "job_id": "uuid",
  "type": "document_indexing",
  "payload": {
    "doc_id": "abc-123",
    "file_path": "documents/abc-123.pdf",
    "metadata": {
      "title": "Golang Tutorial",
      "author": "John Doe"
    }
  },
  "created_at": "2025-01-15T10:30:00Z",
  "retry_count": 0
}
```

---

### 5. **Search Service**

**Responsibilities:**

- Process search queries
- Coordinate distributed queries across shards
- Rank and merge results
- Apply filters and facets

**Query Processing Pipeline:**

```
User Query: "golang tutorials"
    ↓
Validate JWT token
    ↓
Tokenize: ["golang", "tutorials"]
    ↓
Normalize: ["golang", "tutorial"]
    ↓
Query ScyllaDB shards (parallel)
    ↓
Retrieve document IDs
    ↓
Calculate TF-IDF scores
    ↓
Rank results
    ↓
Fetch top documents from MinIO
    ↓
Return results to user
```

**Input:**

- Search query: `string`
- Filters: `{ type, date_range, author }`
- Pagination: `{ page, limit }`
- Ranking preference: `relevance/date/popularity`

**Output:**

```json
{
  "total_results": 1234,
  "query_time_ms": 45,
  "results": [
    {
      "doc_id": "abc-123",
      "title": "Golang Tutorial for Beginners",
      "snippet": "...learn golang fundamentals...",
      "score": 0.89,
      "url": "https://minio/docs/abc-123"
    }
  ]
}
```

---

### 6. **Analytics Service**

**Responsibilities:**

- Track search queries
- Monitor system performance
- Generate reports and insights
- Consume analytics events from RabbitMQ

**Metrics Collected:**

- Search query frequency
- Query response times
- Popular search terms
- Zero-result queries
- Click-through rates
- System health metrics

---

## 🛠️ Technology Stack

| Component            | Technology              | Why?                                                      |
| -------------------- | ----------------------- | --------------------------------------------------------- |
| **Backend Services** | Go (Golang)             | High performance, excellent concurrency, statically typed |
| **Load Balancer**    | Nginx                   | Industry standard, SSL termination, rate limiting         |
| **Authentication**   | PostgreSQL + JWT        | Relational data for users, secure token-based auth        |
| **Message Broker**   | RabbitMQ                | Simple, reliable job queues, built-in retry logic         |
| **Database (Index)** | ScyllaDB                | Ultra-fast reads (<10ms), horizontal scalability          |
| **Object Storage**   | MinIO                   | S3-compatible, perfect for large documents                |
| **Monitoring**       | Prometheus + Grafana    | System metrics and performance monitoring                 |
| **Containerization** | Docker + Docker Compose | Easy local development and deployment                     |

---

## ✨ Features

### Core Features

#### 1. **User Authentication**

- JWT-based authentication
- Secure password hashing (bcrypt)
- Token refresh mechanism
- API key management
- Role-based access control

#### 2. **Full-Text Search**

- Search across document content
- Multi-word queries
- Boolean operators: AND, OR, NOT
- Phrase search: "exact phrase matching"

**Example:**

```
Query: golang AND (tutorial OR guide)
Returns: Documents containing "golang" AND either "tutorial" or "guide"
```

#### 3. **Relevance Ranking (TF-IDF)**

**Term Frequency (TF):**

```
TF = (Number of times term appears in document) / (Total terms in document)

Example:
Document has 100 words, "golang" appears 5 times
TF = 5/100 = 0.05
```

**Inverse Document Frequency (IDF):**

```
IDF = log(Total documents / Documents containing term)

Example:
10,000 total documents, "golang" appears in 100
IDF = log(10,000/100) = 2.0

"the" appears in 9,000 documents
IDF = log(10,000/9,000) = 0.046 ← Less important!
```

**Final Score:**

```
Score = TF × IDF

"golang" in document: 0.05 × 2.0 = 0.10
"the" in document: 0.50 × 0.046 = 0.023

"golang" is more relevant despite appearing less frequently!
```

#### 4. **Asynchronous Indexing**

Documents are indexed asynchronously for better UX:

```
User uploads document
    ↓
API returns immediately: "Document queued for indexing"
    ↓
Background worker processes indexing
    ↓
User can check status: /documents/{id}/status
```

#### 5. **Faceted Search (Filters)**

Allow users to refine results by categories:

```
Search: "machine learning"
Results: 5,432 documents

Filters:
┌─ Type:
│  ☑ Article (3,210)
│  ☐ Video (1,432)
│  ☐ Book (790)
│
┌─ Difficulty:
│  ☐ Beginner (1,234)
│  ☑ Intermediate (2,345)
│  ☐ Advanced (1,853)
│
└─ Year:
   ☐ 2025 (876)
   ☑ 2024 (2,345)
   ☐ 2023 (1,567)
```

#### 6. **Autocomplete (Type-ahead)**

Suggest queries as user types:

```
User types: "mach..."

Suggestions:
1. machine learning (12,345 searches)
2. machine vision (3,456 searches)
3. machiavelli (234 searches)
```

#### 7. **Search Analytics**

Track and visualize:

- Most popular queries
- Query performance
- Zero-result queries
- Peak usage times
- Click-through rates

---

## 🔄 Data Flow

### Authentication Flow

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ POST /auth/login
       ↓
┌──────────────────┐
│     Nginx        │
└──────┬───────────┘
       │
       ↓
┌──────────────────┐
│  Auth Service    │
│                  │
│ 1. Validate      │
│ 2. Check DB      │
│ 3. Generate JWT  │
└──────┬───────────┘
       │
       ↓
┌──────────────────┐
│   PostgreSQL     │
│   (User data)    │
└──────────────────┘
       │
       ↓
┌──────────────┐
│   Client     │
│ (JWT token)  │
└──────────────┘
```

---

### Indexing Flow (Async with RabbitMQ)

```
┌─────────────┐
│   Client    │
│ (Upload Doc)│
└──────┬──────┘
       │ POST /documents (with JWT)
       ↓
┌──────────────────┐
│     Nginx        │
└──────┬───────────┘
       │
       ↓
┌──────────────────────────┐
│   Indexing Service       │
│   (API Mode)             │
│                          │
│  1. Validate JWT         │
│  2. Store doc in MinIO   │
│  3. Publish to RabbitMQ  │
│  4. Return 202 Accepted  │
└──────┬───────────────────┘
       │
       ↓
┌──────────────────┐         ┌──────────────┐
│    RabbitMQ      │         │    MinIO     │
│                  │         │              │
│  indexing_queue  │         │ Original doc │
└──────┬───────────┘         └──────────────┘
       │
       │ (Worker consumes)
       ↓
┌──────────────────────────┐
│   Indexing Service       │
│   (Worker Mode)          │
│                          │
│  1. Extract text         │
│  2. Tokenize             │
│  3. Build inverted index │
│  4. Store in ScyllaDB    │
└──────┬───────────────────┘
       │
       ↓
┌─────────────┐
│  ScyllaDB   │
│             │
│ Inverted    │
│ Index       │
└─────────────┘
```

---

### Search Flow

```
┌─────────────┐
│   Client    │
│ (Search)    │
└──────┬──────┘
       │ GET /search?q=golang (with JWT)
       ↓
┌──────────────────┐
│     Nginx        │
└──────┬───────────┘
       │
       ↓
┌────────────────────────┐
│   Search Service       │
│                        │
│  1. Validate JWT       │
│  2. Parse query        │
│  3. Tokenize           │
│  4. Query coordinator  │
└──────┬─────────────────┘
       │
       ↓
┌────────────────────────────────┐
│   Query Coordinator            │
│                                │
│   Parallel queries to shards   │
└──┬────────┬────────┬───────────┘
   │        │        │
   ↓        ↓        ↓
┌──────┐ ┌──────┐ ┌──────┐
│Shard1│ │Shard2│ │Shard3│
└──┬───┘ └──┬───┘ └──┬───┘
   │        │        │
   └────────┼────────┘
            │
            ↓
┌───────────────────────┐
│  Result Merger        │
│                       │
│  1. Combine results   │
│  2. Re-rank globally  │
│  3. Apply filters     │
└──────┬────────────────┘
       │
       ↓
┌──────────────────┐
│  Document        │
│  Fetcher         │
│  (MinIO)         │
└──────┬───────────┘
       │
       ↓
┌──────────────┐
│   Client     │
│  (Results)   │
└──────────────┘
       │
       │ (Track search asynchronously)
       ↓
┌──────────────────┐
│    RabbitMQ      │
│                  │
│ analytics_queue  │
└──────┬───────────┘
       │
       ↓
┌──────────────────┐
│  Analytics       │
│  Service         │
└──────────────────┘
```

---

## 🗄️ Database Schema

### PostgreSQL (Auth Service)

#### Users Table

```sql
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_users_email ON users(email);
```

#### API Keys Table

```sql
CREATE TABLE api_keys (
    key_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(user_id),
    api_key VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    last_used TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_api_keys_key ON api_keys(api_key);
CREATE INDEX idx_api_keys_user ON api_keys(user_id);
```

---

### ScyllaDB Tables

#### 1. Inverted Index Table

```sql
CREATE TABLE inverted_index (
    word TEXT,
    doc_id UUID,
    term_frequency INT,
    positions LIST<INT>,
    field TEXT,
    PRIMARY KEY (word, doc_id)
) WITH CLUSTERING ORDER BY (doc_id ASC);
```

#### 2. Document Metadata Table

```sql
CREATE TABLE documents (
    doc_id UUID PRIMARY KEY,
    title TEXT,
    author TEXT,
    doc_type TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    file_path TEXT,
    file_size BIGINT,
    tags SET<TEXT>,
    total_words INT,
    language TEXT,
    status TEXT,
    indexed_at TIMESTAMP,
    owner_id UUID
);

CREATE INDEX idx_documents_owner ON documents(owner_id);
CREATE INDEX idx_documents_status ON documents(status);
```

#### 3. Global Statistics Table

```sql
CREATE TABLE global_stats (
    word TEXT PRIMARY KEY,
    document_frequency INT,
    total_frequency BIGINT,
    last_updated TIMESTAMP
);
```

#### 4. Autocomplete Table

```sql
CREATE TABLE autocomplete (
    prefix TEXT,
    suggestion TEXT,
    frequency INT,
    PRIMARY KEY (prefix, frequency, suggestion)
) WITH CLUSTERING ORDER BY (frequency DESC);
```

#### 5. Search Analytics Table

```sql
CREATE TABLE search_analytics (
    date DATE,
    query TEXT,
    search_count COUNTER,
    avg_response_time_ms INT,
    zero_results_count COUNTER,
    user_id UUID,
    PRIMARY KEY ((date), query, user_id)
);
```

---

### MinIO Structure

```
searchflow-bucket/
├── documents/
│   ├── user-uuid-1/
│   │   ├── 123e4567-e89b-12d3-a456-426614174000.pdf
│   │   └── 789e4567-e89b-12d3-a456-426614174001.json
│   ├── user-uuid-2/
│   │   └── 456e4567-e89b-12d3-a456-426614174002.txt
│
└── thumbnails/
    └── 123e4567.jpg
```

---

## 🌐 API Endpoints

### Authentication Endpoints

#### Register User

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "full_name": "John Doe"
}

Response 201:
{
  "user_id": "123e4567-e89b-12d3...",
  "email": "user@example.com",
  "full_name": "John Doe"
}
```

#### Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!"
}

Response 200:
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### Validate Token

```http
GET /api/v1/auth/validate
Authorization: Bearer <token>

Response 200:
{
  "valid": true,
  "user_id": "123e4567-e89b-12d3...",
  "email": "user@example.com",
  "role": "user"
}
```

---

### Document Management Endpoints

#### Upload Document

```http
POST /api/v1/documents
Authorization: Bearer <token>
Content-Type: multipart/form-data

{
  "file": <binary>,
  "title": "Golang Tutorial",
  "author": "John Doe",
  "type": "article",
  "tags": ["golang", "programming", "tutorial"]
}

Response 202:
{
  "doc_id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "queued",
  "message": "Document queued for indexing",
  "status_url": "/api/v1/documents/123e4567.../status"
}
```

#### Check Indexing Status

```http
GET /api/v1/documents/{doc_id}/status
Authorization: Bearer <token>

Response 200:
{
  "doc_id": "123e4567...",
  "status": "completed",
  "progress": 100,
  "message": "Document successfully indexed",
  "indexed_at": "2025-01-15T10:35:00Z"
}

Possible statuses: queued, processing, completed, failed
```

#### Get Document

```http
GET /api/v1/documents/{doc_id}
Authorization: Bearer <token>

Response 200:
{
  "doc_id": "123e4567...",
  "title": "Golang Tutorial",
  "author": "John Doe",
  "download_url": "https://minio.local/documents/123e4567...",
  "metadata": {
    "type": "article",
    "created_at": "2025-01-15T10:30:00Z",
    "file_size": 2048576,
    "word_count": 5432,
    "status": "completed"
  }
}
```

#### List User Documents

```http
GET /api/v1/documents?page=1&limit=20
Authorization: Bearer <token>

Response 200:
{
  "total": 156,
  "page": 1,
  "limit": 20,
  "documents": [
    {
      "doc_id": "123e4567...",
      "title": "Golang Tutorial",
      "status": "completed",
      "created_at": "2025-01-15T10:30:00Z"
    }
  ]
}
```

#### Delete Document

```http
DELETE /api/v1/documents/{doc_id}
Authorization: Bearer <token>

Response 204: No Content
```

---

### Search Endpoints

#### Basic Search

```http
GET /api/v1/search?q=golang+tutorials&page=1&limit=10
Authorization: Bearer <token>

Response 200:
{
  "query": "golang tutorials",
  "total_results": 1234,
  "query_time_ms": 45,
  "page": 1,
  "limit": 10,
  "results": [
    {
      "doc_id": "123e4567...",
      "title": "Golang Tutorial for Beginners",
      "snippet": "Learn **golang** fundamentals with practical **tutorials**...",
      "score": 0.89,
      "url": "/api/v1/documents/123e4567...",
      "metadata": {
        "author": "John Doe",
        "type": "article",
        "date": "2025-01-15"
      }
    }
  ]
}
```

#### Advanced Search with Filters

```http
GET /api/v1/search?q=golang&type=article&from=2024-01-01&to=2025-01-01
Authorization: Bearer <token>

Response 200:
{
  "query": "golang",
  "filters_applied": {
    "type": "article",
    "date_range": "2024-01-01 to 2025-01-01"
  },
  "total_results": 234,
  "results": [...]
}
```

#### Faceted Search

```http
GET /api/v1/search?q=machine+learning&facets=type,year,author
Authorization: Bearer <token>

Response 200:
{
  "results": [...],
  "facets": {
    "type": {
      "article": 3210,
      "video": 1432,
      "book": 790
    },
    "year": {
      "2025": 876,
      "2024": 2345,
      "2023": 1567
    }
  }
}
```

---

### Autocomplete Endpoint

```http
GET /api/v1/autocomplete?q=gol&limit=5
Authorization: Bearer <token>

Response 200:
{
  "prefix": "gol",
  "suggestions": [
    { "text": "golang", "frequency": 10000 },
    { "text": "golang tutorial", "frequency": 2500 },
    { "text": "gold price", "frequency": 5000 }
  ]
}
```

---

### Analytics Endpoints

#### Search Trends

```http
GET /api/v1/analytics/trends?period=7d
Authorization: Bearer <token>

Response 200:
{
  "period": "last_7_days",
  "top_queries": [
    { "query": "golang tutorials", "count": 8765 },
    { "query": "react hooks", "count": 6543 }
  ]
}
```

#### System Health

```http
GET /api/v1/analytics/health
Authorization: Bearer <token>

Response 200:
{
  "status": "healthy",
  "services": {
    "auth": "healthy",
    "search": "healthy",
    "indexer": "healthy",
    "rabbitmq": "healthy",
    "scylladb": "healthy"
  },
  "shards": [
  { "id": "shard-1", "status": "healthy", "doc_count": 3456789 }
    ]
  }
```

---

## 🧮 Algorithms & Techniques

### 1. Tokenization

```go
Input:  "Golang is GREAT for building APIs!"
Steps:
  1. Lowercase: "golang is great for building apis!"
  2. Remove punctuation: "golang is great for building apis"
  3. Split: ["golang", "is", "great", "for", "building", "apis"]
  4. Remove stopwords: ["golang", "great", "building", "apis"]
Output: ["golang", "great", "building", "apis"]
```

### 2. Stemming

```
running   → run
tutorials → tutorial
cats      → cat
```

### 3. TF-IDF Calculation

```go
func CalculateTFIDF(term, docID string, totalDocs int) float64 {
    tf := float64(termFreqInDoc) / float64(totalTermsInDoc)
    idf := math.Log(float64(totalDocs) / float64(docsContainingTerm))
    return tf * idf
}
```

### 4. Consistent Hashing (Shard Selection)

```go
func GetShard(term string, numShards int) int {
    hash := crc32.ChecksumIEEE([]byte(term))
    return int(hash % uint32(numShards))
}
```

---

## 🌍 Scalability & Distribution

### Sharding Strategy

```
Shard 1: Terms A-H (333K terms)
Shard 2: Terms I-P (333K terms)
Shard 3: Terms Q-Z (334K terms)

Each shard has 3 replicas (1 primary + 2 replicas)
```

### Replication

```
Shard 1 Primary → Replica 1 → Replica 2
(Replication Factor = 3)

Read: Any replica (load balanced)
Write: Primary (async to replicas)
```

### Handling Failures

```
Primary fails → Replica promoted → New replica spawned
```

---

## 🚧 Project Boundaries

### In Scope ✅

- JWT authentication
- Async document indexing (RabbitMQ)
- Full-text search with TF-IDF
- Sharding + replication
- Autocomplete
- Faceted search
- Search analytics
- Nginx load balancing
- Docker deployment

### Out of Scope ❌

- Machine learning ranking
- Multi-language support
- Image/video search
- Auto-scaling
- Multi-tenancy
- GDPR compliance features

---

## 📚 Learning Outcomes

- ✅ Microservices with Go
- ✅ JWT authentication
- ✅ Message queues (RabbitMQ)
- ✅ Nginx reverse proxy
- ✅ Distributed systems
- ✅ Search algorithms
- ✅ ScyllaDB/Cassandra
- ✅ Async processing
- ✅ Load balancing

---

## 🗓️ Implementation Phases

### Phase 1: Foundation (Week 1-2)

- [ ] Project setup + Go structure
- [ ] Basic tokenizer
- [ ] In-memory inverted index
- [ ] Simple search + TF-IDF

### Phase 2: Authentication (Week 3)

- [ ] Auth service with JWT
- [ ] PostgreSQL integration
- [ ] User registration/login
- [ ] Token validation

### Phase 3: Nginx Setup (Week 4)

- [ ] Configure Nginx
- [ ] Service routing
- [ ] SSL setup
- [ ] Rate limiting

### Phase 4: Async Indexing (Week 5)

- [ ] RabbitMQ setup
- [ ] Producer (API)
- [ ] Consumer (Worker)
- [ ] Retry logic + DLQ

### Phase 5: Persistence (Week 6)

- [ ] ScyllaDB setup
- [ ] Schema design
- [ ] Migrate index to ScyllaDB
- [ ] MinIO integration

### Phase 6: Distribution (Week 7-8)

- [ ] Multi-node ScyllaDB
- [ ] Sharding implementation
- [ ] Query coordinator
- [ ] Result merging

### Phase 7: Replication (Week 9)

- [ ] Configure replication
- [ ] Failover logic
- [ ] Health checks

### Phase 8: Features (Week 10-11)

- [ ] Autocomplete
- [ ] Faceted search
- [ ] Analytics service
- [ ] Search tracking

### Phase 9: Monitoring (Week 12)

- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Logging
- [ ] Alerting

### Phase 10: Polish (Week 13)

- [ ] Documentation
- [ ] Demo video
- [ ] Performance tuning
- [ ] GitHub showcase

---

## 🎯 Success Metrics

| Metric              | Target    |
| ------------------- | --------- |
| Query Latency (p50) | <50ms     |
| Query Latency (p99) | <200ms    |
| Throughput          | 1,000 QPS |
| Index Size          | 1M+ docs  |
| Availability        | 99.9%     |

---

## 🚀 Getting Started

```bash
# Clone repository
git clone https://github.com/amrrdev/searchflow.git
cd searchflow

# Start all services
docker-compose up -d

# Create first user
curl -X POST http://localhost/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Pass123!"}'

# Login
curl -X POST http://localhost/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Pass123!"}'

# Upload document
curl -X POST http://localhost/api/v1/documents \
  -H "Authorization: Bearer <token>" \
  -F "file=@example.pdf" \
  -F "title=Example Document"

# Search
curl "http://localhost/api/v1/search?q=golang" \
  -H "Authorization: Bearer <token>"
```

---

## 📄 License

MIT License

---

## 👤 Author

**Amr Ashraf Mubarak**

- GitHub: [@amrrdev](https://github.com/amrrdev)
- LinkedIn: [amramubarak](https://linkedin.com/in/amramubarak)
- Email: amrrdev@gmail.com

---

**Built with ❤️ to master distributed systems and search technology**
