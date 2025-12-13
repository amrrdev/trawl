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
│                      API GATEWAY (Go)                        │
│  - Request routing                                           │
│  - Authentication                                            │
│  - Rate limiting                                             │
└────────────────────────┬────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         ↓               ↓               ↓
┌─────────────┐  ┌──────────────┐  ┌──────────────┐
│  INDEXING   │  │    SEARCH    │  │  ANALYTICS   │
│  SERVICE    │  │   SERVICE    │  │   SERVICE    │
│             │  │              │  │              │
│ - Parse     │  │ - Query      │  │ - Track      │
│   documents │  │   processing │  │   searches   │
│ - Tokenize  │  │ - Ranking    │  │ - Generate   │
│ - Build     │  │ - Result     │  │   reports    │
│   index     │  │   merging    │  │              │
└──────┬──────┘  └──────┬───────┘  └──────┬───────┘
       │                │                 │
       └────────────────┼─────────────────┘
                        │
         ┌──────────────┼──────────────┐
         ↓              ↓              ↓
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  ScyllaDB   │  │   ScyllaDB  │  │  ScyllaDB   │
│   Shard 1   │  │   Shard 2   │  │   Shard 3   │
│             │  │             │  │             │
│ Words: A-H  │  │ Words: I-P  │  │ Words: Q-Z  │
│             │  │             │  │             │
│ (Replicated)│  │ (Replicated)│  │ (Replicated)│
└─────────────┘  └─────────────┘  └─────────────┘
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

### 1. **Indexing Service**

**Responsibility:** Transform documents into searchable inverted indexes

**Process Flow:**
```
Document Upload
    ↓
Extract Text (PDF/JSON/TXT parser)
    ↓
Tokenization (split into words)
    ↓
Normalization (lowercase, remove punctuation)
    ↓
Stop Words Removal (remove "the", "is", "a")
    ↓
Stemming (running → run, cats → cat)
    ↓
Build Inverted Index
    ↓
Store in ScyllaDB
    ↓
Upload Original Document to MinIO
```

**Input:**
- Document ID: `uuid`
- Document content: `binary/text`
- Metadata: `{ title, author, type, tags }`

**Output:**
- Inverted index entries in ScyllaDB
- Document stored in MinIO
- Indexing status: `success/failure`

---

### 2. **Search Service**

**Responsibility:** Process queries and return ranked results

**Query Processing Pipeline:**
```
User Query: "golang tutorials"
    ↓
Tokenize: ["golang", "tutorials"]
    ↓
Normalize: ["golang", "tutorial"] (stemming)
    ↓
Query ScyllaDB Shards (parallel)
    ↓
Retrieve Document IDs
    ↓
Calculate TF-IDF Scores
    ↓
Rank Results
    ↓
Fetch Top Documents from MinIO
    ↓
Return Results to User
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
      "url": "https://minio/docs/abc-123",
      "metadata": {
        "author": "John Doe",
        "type": "article",
        "date": "2025-01-15"
      }
    }
  ],
  "facets": {
    "type": { "article": 890, "video": 234, "book": 110 },
    "year": { "2025": 123, "2024": 345, "2023": 456 }
  }
}
```

---

### 3. **Query Coordinator**

**Responsibility:** Distribute queries across multiple shards and merge results

**Distributed Query Execution:**
```go
// Pseudo-code
func DistributedSearch(query string) SearchResults {
    tokens := tokenize(query)
    
    // Determine which shards to query
    shards := getRelevantShards(tokens)
    
    // Query all shards in parallel
    resultChan := make(chan ShardResult, len(shards))
    for _, shard := range shards {
        go func(s Shard) {
            resultChan <- s.Search(tokens)
        }(shard)
    }
    
    // Collect results from all shards
    var allResults []Document
    for i := 0; i < len(shards); i++ {
        result := <-resultChan
        allResults = append(allResults, result.Docs...)
    }
    
    // Merge and re-rank globally
    return mergeAndRank(allResults)
}
```

---

### 4. **Analytics Service**

**Responsibility:** Track search patterns and system performance

**Metrics Collected:**
- Search query frequency
- Query response times
- Popular search terms
- Zero-result queries (to improve index)
- User click-through rates
- System health metrics

**Output:**
- Dashboard showing search trends
- Slow query reports
- Search improvement suggestions

---

## 🛠️ Technology Stack

| Component | Technology | Why? |
|-----------|-----------|------|
| **Backend** | Go (Golang) | High performance, excellent concurrency (goroutines), statically typed |
| **Database (Index)** | ScyllaDB | Ultra-fast reads (<10ms), horizontal scalability, Cassandra-compatible |
| **Object Storage** | MinIO | S3-compatible, perfect for storing large documents |
| **Message Queue** | (Optional) Kafka/RabbitMQ | For async indexing of large document batches |
| **Cache** | (Optional) Redis | For caching hot queries and autocomplete |
| **Monitoring** | Prometheus + Grafana | System metrics and performance monitoring |

---

## ✨ Features

### Core Features

#### 1. **Full-Text Search**
- Search across document content
- Support for multi-word queries
- Boolean operators: AND, OR, NOT
- Phrase search: "exact phrase matching"

**Example:**
```
Query: golang AND (tutorial OR guide)
Returns: Documents containing "golang" AND either "tutorial" or "guide"
```

#### 2. **Relevance Ranking (TF-IDF)**

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

#### 3. **Faceted Search (Filters)**

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

#### 4. **Autocomplete (Type-ahead)**

Suggest queries as user types:

```
User types: "mach..."

Suggestions:
1. machine learning (12,345 searches)
2. machine vision (3,456 searches)
3. machiavelli (234 searches)
```

**Implementation:** Prefix tree (Trie) data structure

```
        m
        |
        a
        |
        c
        |
        h ──┬── i → machine (freq: 15,801)
            └── i → machiavelli (freq: 234)
```

#### 5. **Search Analytics**

Track and visualize:
- Most popular queries
- Query performance (response times)
- Zero-result queries (needs index improvement)
- Peak usage times
- Click-through rates

---

### Advanced Features (Optional Enhancements)

#### 6. **Fuzzy Search (Typo Tolerance)**

Handle typos using edit distance:

```
User searches: "golng" (typo)
System suggests: "golang" (edit distance: 1)

Algorithm: Levenshtein distance
- golng → golang: 1 character addition
```

#### 7. **Synonym Expansion**

```
User searches: "car"
System also searches: ["automobile", "vehicle"]

Expands results without user effort
```

#### 8. **Highlighting**

Show matched terms in results:

```
Result snippet:
"Learn **Golang** fundamentals and build scalable **APIs** 
with this comprehensive **tutorial**."
```

---

## 🔄 Data Flow

### Indexing Flow

```
┌─────────────┐
│   Client    │
│ (Upload Doc)│
└──────┬──────┘
       │
       ↓
┌──────────────────┐
│  API Gateway     │
│  POST /documents │
└──────┬───────────┘
       │
       ↓
┌──────────────────────────┐
│   Indexing Service       │
│                          │
│  1. Parse document       │
│  2. Extract text         │
│  3. Tokenize             │
│  4. Build inverted index │
└──────┬─────────┬─────────┘
       │         │
       │         └─────────────┐
       ↓                       ↓
┌─────────────┐        ┌──────────────┐
│  ScyllaDB   │        │    MinIO     │
│             │        │              │
│ Store index │        │ Store        │
│ entries     │        │ original doc │
└─────────────┘        └──────────────┘
```

### Search Flow

```
┌─────────────┐
│   Client    │
│ (Search)    │
└──────┬──────┘
       │
       ↓
┌──────────────────┐
│  API Gateway     │
│  GET /search?q=  │
└──────┬───────────┘
       │
       ↓
┌────────────────────────┐
│   Search Service       │
│                        │
│  1. Parse query        │
│  2. Tokenize           │
│  3. Query coordinator  │
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
│                  │
│  Get full docs   │
│  from MinIO      │
└──────┬───────────┘
       │
       ↓
┌──────────────┐
│   Client     │
│  (Results)   │
└──────────────┘
```

---

## 🗄️ Database Schema

### ScyllaDB Tables

#### 1. **Inverted Index Table**

```sql
CREATE TABLE inverted_index (
    word TEXT,                    -- Indexed term (normalized)
    doc_id UUID,                  -- Document identifier
    term_frequency INT,           -- How many times term appears in doc
    positions LIST<INT>,          -- Positions where term appears
    field TEXT,                   -- Which field (title, body, tags)
    PRIMARY KEY (word, doc_id)
) WITH CLUSTERING ORDER BY (doc_id ASC);

-- Example data:
┌─────────┬──────────────────────┬──────────┬────────────┬────────┐
│ word    │ doc_id               │ term_freq│ positions  │ field  │
├─────────┼──────────────────────┼──────────┼────────────┼────────┤
│ golang  │ 123e4567-e89b-12d3...│ 5        │ [1,15,23...│ body   │
│ golang  │ 789e4567-e89b-12d3...│ 2        │ [5,18]     │ title  │
│ api     │ 123e4567-e89b-12d3...│ 3        │ [10,25,40] │ body   │
└─────────┴──────────────────────┴──────────┴────────────┴────────┘
```

#### 2. **Document Metadata Table**

```sql
CREATE TABLE documents (
    doc_id UUID PRIMARY KEY,
    title TEXT,
    author TEXT,
    doc_type TEXT,              -- article, video, book, etc.
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    file_path TEXT,             -- MinIO object path
    file_size BIGINT,
    tags SET<TEXT>,
    total_words INT,
    language TEXT
);

-- Example data:
┌──────────────────────┬─────────────────────┬──────────┬──────────┐
│ doc_id               │ title               │ author   │ doc_type │
├──────────────────────┼─────────────────────┼──────────┼──────────┤
│ 123e4567-e89b-12d3...│ Golang Tutorial     │ John Doe │ article  │
│ 789e4567-e89b-12d3...│ API Design Patterns │ Jane Doe │ book     │
└──────────────────────┴─────────────────────┴──────────┴──────────┘
```

#### 3. **Global Statistics Table**

```sql
CREATE TABLE global_stats (
    word TEXT PRIMARY KEY,
    document_frequency INT,     -- Number of docs containing this word
    total_frequency BIGINT,     -- Total occurrences across all docs
    last_updated TIMESTAMP
);

-- Used for IDF calculation
┌─────────┬────────────┬─────────────┐
│ word    │ doc_freq   │ total_freq  │
├─────────┼────────────┼─────────────┤
│ golang  │ 1,234      │ 15,678      │
│ the     │ 98,765     │ 1,234,567   │
│ api     │ 5,678      │ 45,890      │
└─────────┴────────────┴─────────────┘
```

#### 4. **Autocomplete Table**

```sql
CREATE TABLE autocomplete (
    prefix TEXT,
    suggestion TEXT,
    frequency INT,              -- How often this query was searched
    PRIMARY KEY (prefix, frequency, suggestion)
) WITH CLUSTERING ORDER BY (frequency DESC);

-- Example: prefix "gol"
┌────────┬────────────────┬───────────┐
│ prefix │ suggestion     │ frequency │
├────────┼────────────────┼───────────┤
│ gol    │ golang         │ 10,000    │
│ gol    │ gold price     │ 5,000     │
│ gol    │ golf courses   │ 3,000     │
└────────┴────────────────┴───────────┘
```

#### 5. **Search Analytics Table**

```sql
CREATE TABLE search_analytics (
    date DATE,
    query TEXT,
    search_count COUNTER,
    avg_response_time_ms INT,
    zero_results_count COUNTER,
    PRIMARY KEY (date, query)
);

┌────────────┬──────────────────┬──────────┬──────────────┐
│ date       │ query            │ count    │ avg_time_ms  │
├────────────┼──────────────────┼──────────┼──────────────┤
│ 2025-01-15 │ golang tutorials │ 1,234    │ 45           │
│ 2025-01-15 │ react hooks      │ 890      │ 38           │
└────────────┴──────────────────┴──────────┴──────────────┘
```

---

### MinIO Structure

```
searchflow-bucket/
├── documents/
│   ├── 123e4567-e89b-12d3-a456-426614174000.pdf
│   ├── 789e4567-e89b-12d3-a456-426614174001.json
│   └── 456e4567-e89b-12d3-a456-426614174002.txt
│
├── thumbnails/          (optional: for preview images)
│   ├── 123e4567.jpg
│   └── 789e4567.jpg
│
└── metadata/            (optional: cached metadata)
    └── index_stats.json
```

---

## 🌐 API Endpoints

### 1. **Document Management**

#### Upload Document
```http
POST /api/v1/documents
Content-Type: multipart/form-data

{
  "file": <binary>,
  "title": "Golang Tutorial",
  "author": "John Doe",
  "type": "article",
  "tags": ["golang", "programming", "tutorial"]
}

Response 201:
{
  "doc_id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "indexing",
  "message": "Document uploaded and indexing started"
}
```

#### Get Document
```http
GET /api/v1/documents/{doc_id}

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
    "word_count": 5432
  }
}
```

#### Delete Document
```http
DELETE /api/v1/documents/{doc_id}

Response 204: No Content
```

---

### 2. **Search**

#### Basic Search
```http
GET /api/v1/search?q=golang+tutorials&page=1&limit=10

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
        "date": "2025-01-15",
        "tags": ["golang", "programming"]
      }
    }
  ]
}
```

#### Advanced Search with Filters
```http
GET /api/v1/search?q=golang&type=article&author=John+Doe&from=2024-01-01&to=2025-01-01

Response 200:
{
  "query": "golang",
  "filters_applied": {
    "type": "article",
    "author": "John Doe",
    "date_range": "2024-01-01 to 2025-01-01"
  },
  "total_results": 234,
  "results": [...]
}
```

#### Faceted Search
```http
GET /api/v1/search?q=machine+learning&facets=type,year,author

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
    },
    "author": {
      "John Doe": 234,
      "Jane Smith": 456,
      "Bob Johnson": 189
    }
  }
}
```

---

### 3. **Autocomplete**

```http
GET /api/v1/autocomplete?q=gol&limit=5

Response 200:
{
  "prefix": "gol",
  "suggestions": [
    { "text": "golang", "frequency": 10000 },
    { "text": "gold price", "frequency": 5000 },
    { "text": "golf courses", "frequency": 3000 },
    { "text": "golang tutorial", "frequency": 2500 },
    { "text": "golden retriever", "frequency": 1200 }
  ]
}
```

---

### 4. **Analytics**

#### Search Trends
```http
GET /api/v1/analytics/trends?period=7d

Response 200:
{
  "period": "last_7_days",
  "top_queries": [
    { "query": "golang tutorials", "count": 8765 },
    { "query": "react hooks", "count": 6543 },
    { "query": "docker compose", "count": 5432 }
  ],
  "query_volume": {
    "2025-01-09": 15678,
    "2025-01-10": 16234,
    "2025-01-11": 14567
  }
}
```

#### System Health
```http
GET /api/v1/analytics/health

Response 200:
{
  "status": "healthy",
  "shards": [
    { "id": "shard-1", "status": "healthy", "doc_count": 3456789 },
    { "id": "shard-2", "status": "healthy", "doc_count": 3234567 },
    { "id": "shard-3", "status": "healthy", "doc_count": 3567890 }
  ],
  "avg_query_time_ms": 42,
  "p95_query_time_ms": 78,
  "p99_query_time_ms": 145
}
```

---

## 🧮 Algorithms & Techniques

### 1. **Tokenization**

Break text into words (tokens):

```go
Input:  "Golang is GREAT for building APIs!"
Steps:
  1. Lowercase: "golang is great for building apis!"
  2. Remove punctuation: "golang is great for building apis"
  3. Split by whitespace: ["golang", "is", "great", "for", "building", "apis"]
  4. Remove stopwords: ["golang", "great", "building", "apis"]
Output: ["golang", "great", "building", "apis"]
```

### 2. **Stemming/Lemmatization**

Reduce words to their root form:

```
running   → run
runs      → run
ran       → run
better    → good
cats      → cat
tutorials → tutorial
```

**Algorithm:** Porter Stemmer or Snowball Stemmer

### 3. **TF-IDF Calculation**

```go
// Pseudo-code
func CalculateTFIDF(term string, docID string, totalDocs int) float64 {
    // Term Frequency (TF)
    termFreqInDoc := getTermFrequency(term, docID)
    totalTermsInDoc := getTotalTerms(docID)
    tf := float64(termFreqInDoc) / float64(totalTermsInDoc)
    
    // Inverse Document Frequency (IDF)
    docsContainingTerm := getDocumentFrequency(term)
    idf := math.Log(float64(totalDocs) / float64(docsContainingTerm))
    
    // TF-IDF Score
    return tf * idf
}

// Example:
// Document: 100 words, "golang" appears 5 times
// Total docs: 10,000, "golang" in 100 docs
// TF = 5/100 = 0.05
// IDF = log(10,000/100) = 2.0
// TF-IDF = 0.05 * 2.0 = 0.10
```

### 4. **BM25 Ranking (Advanced Alternative to TF-IDF)**

BM25 improves upon TF-IDF with document length normalization:

```go
func CalculateBM25(term string, docID string, avgDocLength float64) float64 {
    k1 := 1.2  // Term frequency saturation parameter
    b := 0.75  // Length normalization parameter
    
    termFreq := getTermFrequency(term, docID)
    docLength := getDocumentLength(docID)
    idf := calculateIDF(term)
    
    numerator := termFreq * (k1 + 1)
    denominator := termFreq + k1 * (1 - b + b * (docLength / avgDocLength))
    
    return idf * (numerator / denominator)
}
```

### 5. **Consistent Hashing (Shard Selection)**

Determine which shard stores a term:

```go
func GetShard(term string, numShards int) int {
    hash := crc32.ChecksumIEEE([]byte(term))
    return int(hash % uint32(numShards))
}

// Example:
// "golang" → hash → 123456789 → 123456789 % 3 = Shard 0
// "react"  → hash → 987654321 → 987654321 % 3 = Shard 0
// "python" → hash → 456789123 → 456789123 % 3 = Shard 0
```

### 6. **Query Optimization**

```go
// Optimize query by starting with rarest terms
func OptimizeQuery(terms []string) []string {
    // Sort terms by document frequency (ascending)
    // Process rarest terms first to reduce result set quickly
    
    sort.Slice(terms, func(i, j int) bool {
        return getDocumentFrequency(terms[i]) < getDocumentFrequency(terms[j])
    })
    
    return terms
}

// Example:
// Query: "the golang tutorial"
// Doc frequencies: "the" (9000), "golang" (100), "tutorial" (500)
// Optimized order: ["golang", "tutorial", "the"]
// Process "golang" first (smallest result set)
```

---

## 🌍 Scalability & Distribution

### Sharding Strategy

**Horizontal ioning by Term:**

```
Total vocabulary: ~1 million unique terms

Shard 1 (Terms A-H):
├─ "api", "algorithm", "backend", "golang"
├─ Handles ~333,000 terms
└─ 3 replicas for fault tolerance

Shard 2 (Terms I-P):
├─ "index", "java", "kubernetes", "node"
├─ Handles ~333,000 terms
└─ 3 replicas for fault tolerance

Shard 3 (Terms Q-Z):
├─ "react", "search", "tutorial", "yaml"
├─ Handles ~334,000 terms
└─ 3 replicas for fault tolerance
```

**Why this works:**
- Evenly distributes load
- Queries can be parallelized across shards
- Each shard is independently scalable

### Replication Strategy

```
┌─────────────────────────────────────┐
│         Shard 1 (Primary)           │
│         Server: shard1-primary      │
│         Data: Terms A-H             │
└──────────────┬──────────────────────┘
               │
       ┌───────┴────────┐
       ↓                ↓
┌──────────────┐  ┌──────────────┐
│ Shard 1      │  │ Shard 1      │
│ (Replica 1)  │  │ (Replica 2)  │
│ Server:      │  │ Server:      │
│ shard1-rep1  │  │ shard1-rep2  │
└──────────────┘  └──────────────┘
```

**Replication Factor:** 3 (1 primary + 2 replicas)

**Read Strategy:** 
- Reads can go to any replica (load balancing)
- Consistent hashing determines shard
- Round-robin across replicas

**Write Strategy:**
- Writes go to primary
- Async replication to replicas
- Eventual consistency model

### Handling Node Failures

```
Scenario: Shard 1 Primary fails

Before:
[Primary] ─── [Replica 1] ─── [Replica 2]
   ❌            ✓               ✓

After (automatic failover):
[Replica 1]* ─── [Replica 2] ─── [New Replica]
  (promoted)         ✓              (spawning)
  
*Replica 1 becomes new primary
*System spawns new replica to maintain RF=3
```

### Query Distribution

```go
// Simplified query coordinator
func DistributedSearch(query string) Results {
    terms := tokenize(query)
    
    // Group terms by shard
    shardQueries := make(map[int][]string)
    for _, term := range terms {
        shardID := getShardForTerm(term)
        shardQueries[shardID] = append(shardQueries[shardID], term)
    }
    
    // Execute parallel queries
    resultChan := make(chan ShardResult, len(shardQueries))
    for shardID, terms := range shardQueries {
        go func(id int, t []string) {
            replica := selectHealthyReplica(id) // Load balance
            result := replica.Query(t)
            resultChan <- result
        }(shardID, terms)
    }
    
    // Collect and merge
    var allResults []Document
    for i := 0; i < len(shardQueries); i++ {
        result := <-resultChan
        allResults = append(allResults, result.Docs...)
    }
    
    // Global ranking
    return rankAndFilter(allResults, query)
}
```

---

## 🚧 Project Boundaries

### What's In Scope

✅ **Core Search Functionality:**
- Full-text search with inverted indexes
- TF-IDF or BM25 ranking
- Basic boolean queries (AND, OR, NOT)
- Phrase search
- Result pagination

✅ **Document Management:**
- Upload/download documents
- Support for: PDF, TXT, JSON, Markdown
- Basic metadata extraction
- Document deletion

✅ **Distribution:**
- Sharding by term
- Replication (3x)
- Basic load balancing
- Fault tolerance (replica promotion)

✅ **Essential Features:**
- Autocomplete (prefix search)
- Basic faceted search (type, date, author)
- Search analytics (query tracking)
- Simple relevance tuning

✅ **DevOps:**
- Docker containerization
- Basic CI/CD pipeline
- Health monitoring endpoints
- Prometheus metrics export

---

### What's Out of Scope (Future Enhancements)

❌ **Advanced NLP:**
- Machine learning-based ranking
- Entity recognition
- Sentiment analysis
- Multi-language support beyond English

❌ **Complex Features:**
- Geo-spatial search
- Image/video search
- Voice search
- Real-time collaborative features

❌ **Enterprise Features:**
- Multi-tenancy with isolation
- Fine-grained access control (RBAC)
- Audit logging
- Compliance certifications

❌ **Advanced Optimization:**
- Query result caching (Redis)
- Hot/cold data tiering
- Automatic index optimization
- Machine learning for autocomplete

❌ **Production Hardening:**
- Security audit & penetration testing
- GDPR compliance features
- Disaster recovery automation
- 24/7 on-call monitoring

---

### Simplified Assumptions

1. **English-only:** No multi-language support initially
2. **Text documents:** No image/video content analysis
3. **Single tenant:** No user authentication/authorization
4. **Eventual consistency:** Acceptable for analytics
5. **Manual scaling:** No auto-scaling (manual shard addition)
6. **Basic security:** No encryption at rest initially

---

## 📚 Learning Outcomes

After completing this project, you will master:

### Technical Skills

**Backend Development:**
- ✅ Building high-performance Go services
- ✅ Designing RESTful APIs
- ✅ Concurrent programming with goroutines
- ✅ Error handling and logging best practices

**Distributed Systems:**
- ✅ Sharding strategies and consistent hashing
- ✅ Replication and fault tolerance
- ✅ Distributed query execution
- ✅ CAP theorem in practice (Availability over Consistency)

**Databases:**
- ✅ Wide-column stores (ScyllaDB/Cassandra)
- ✅ Data modeling for write-heavy workloads
- ✅ Query optimization
- ✅ Partition key design

**Search Technology:**
- ✅ Inverted index data structures
- ✅ Information retrieval algorithms (TF-IDF, BM25)
- ✅ Text processing pipelines
- ✅ Relevance ranking

**DevOps:**
- ✅ Docker multi-container applications
- ✅ Monitoring and observability (Prometheus/Grafana)
- ✅ CI/CD pipelines
- ✅ Load testing and performance tuning

### Soft Skills

- ✅ System design and architecture
- ✅ Trade-off analysis (consistency vs. availability)
- ✅ Technical documentation writing
- ✅ Performance optimization mindset

---

## 🗓️ Implementation Phases

### Phase 1: Foundation (Week 1-2)
**Goal:** Basic single-node search engine

- [ ] Set up Go project structure
- [ ] Implement basic tokenizer
- [ ] Build in-memory inverted index
- [ ] Create simple search function
- [ ] Add TF-IDF ranking
- [ ] Write unit tests

**Deliverable:** Search single documents in memory

---

### Phase 2: Persistence (Week 3)
**Goal:** Store indexes in ScyllaDB

- [ ] Set up ScyllaDB with Docker
- [ ] Design database schema
- [ ] Implement ScyllaDB client in Go
- [ ] Migrate inverted index to ScyllaDB
- [ ] Add document metadata storage
- [ ] Implement document upload/download

**Deliverable:** Persistent search with ScyllaDB

---

### Phase 3: Object Storage (Week 4)
**Goal:** Store documents in MinIO

- [ ] Set up MinIO with Docker
- [ ] Integrate MinIO SDK in Go
- [ ] Implement document upload to MinIO
- [ ] Extract text from PDFs
- [ ] Handle JSON and TXT files
- [ ] Link MinIO paths with ScyllaDB metadata

**Deliverable:** Full document management system

---

### Phase 4: Distribution (Week 5-6)
**Goal:** Multi-node distributed search

- [ ] Implement consistent hashing for sharding
- [ ] Set up multiple ScyllaDB nodes
- [ ] Build query coordinator
- [ ] Implement parallel query execution
- [ ] Add result merging and re-ranking
- [ ] Handle shard failures

**Deliverable:** Distributed search across multiple nodes

---

### Phase 5: Replication (Week 7)
**Goal:** Fault tolerance

- [ ] Configure ScyllaDB replication factor
- [ ] Implement replica selection logic
- [ ] Add health checks for nodes
- [ ] Build automatic failover mechanism
- [ ] Test node failure scenarios

**Deliverable:** Fault-tolerant search system

---

### Phase 6: Features (Week 8-9)
**Goal:** Enhanced search capabilities

- [ ] Implement autocomplete with prefix tree
- [ ] Add faceted search (filters)
- [ ] Build search analytics tracking
- [ ] Create phrase search support
- [ ] Add boolean operators (AND, OR, NOT)
- [ ] Implement result highlighting

**Deliverable:** Feature-rich search engine

---

### Phase 7: API & Documentation (Week 10)
**Goal:** Production-ready API

- [ ] Design RESTful API endpoints
- [ ] Add request validation
- [ ] Implement pagination
- [ ] Write API documentation (Swagger/OpenAPI)
- [ ] Add rate limiting
- [ ] Create example clients

**Deliverable:** Complete REST API with docs

---

### Phase 8: Monitoring & DevOps (Week 11)
**Goal:** Observability

- [ ] Add Prometheus metrics
- [ ] Set up Grafana dashboards
- [ ] Implement structured logging
- [ ] Create health check endpoints
- [ ] Build Docker Compose for full stack
- [ ] Write deployment guide

**Deliverable:** Production-ready deployment setup

---

### Phase 9: Optimization (Week 12)
**Goal:** Performance tuning

- [ ] Benchmark query performance
- [ ] Optimize ScyllaDB queries
- [ ] Add query result caching (optional Redis)
- [ ] Tune Go performance (profiling)
- [ ] Load test with k6 or Gatling
- [ ] Document performance characteristics

**Deliverable:** Optimized, benchmarked system

---

### Phase 10: Documentation & Polish (Week 13)
**Goal:** Portfolio-ready project

- [ ] Write comprehensive README
- [ ] Create architecture diagrams
- [ ] Record demo video
- [ ] Write blog post explaining key concepts
- [ ] Clean up code and add comments
- [ ] Prepare for GitHub showcase

**Deliverable:** Complete portfolio project

---

## 🎯 Success Metrics

Your search engine should achieve:

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Query Latency (p50)** | <50ms | 50% of queries under 50ms |
| **Query Latency (p99)** | <200ms | 99% of queries under 200ms |
| **Throughput** | 1,000 QPS | Queries per second per node |
| **Index Size** | 1M+ docs | Successfully index 1 million documents |
| **Availability** | 99.9% | Uptime with single node failure |
| **Relevance** | User testing | Top 3 results relevant for test queries |

---

## 📖 Recommended Resources

### Books
- **"Introduction to Information Retrieval"** by Manning, Raghavan, Schütze
- **"Designing Data-Intensive Applications"** by Martin Kleppmann
- **"Cassandra: The Definitive Guide"** by Jeff Carpenter

### Online Resources
- [Go by Example](https://gobyexample.com/)
- [ScyllaDB University](https://university.scylladb.com/)
- [Inverted Index Tutorial](https://nlp.stanford.edu/IR-book/html/htmledition/a-first-take-at-building-an-inverted-index-1.html)

### Tools
- [Postman](https://www.postman.com/) - API testing
- [k6](https://k6.io/) - Load testing
- [Prometheus](https://prometheus.io/) - Monitoring
- [Grafana](https://grafana.com/) - Visualization

---

## 🚀 Getting Started

```bash
# Clone the repository
git clone https://github.com/yourusername/searchflow.git
cd searchflow

# Start infrastructure with Docker Compose
docker-compose up -d scylladb minio

# Install Go dependencies
go mod download

# Run the indexing service
go run cmd/indexer/main.go

# Run the search service
go run cmd/search/main.go

# Run tests
go test ./...

# Upload a document
curl -X POST http://localhost:8080/api/v1/documents \
  -F "file=@example.pdf" \
  -F "title=Example Document"

# Search
curl "http://localhost:8080/api/v1/search?q=golang+tutorial"
```

---

## 🤝 Contributing

This is a learning project, but contributions are welcome!

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Write tests
5. Submit a pull request

---

## 📄 License

MIT License - Feel free to use this for learning and portfolio purposes.

---

## 👤 Author

**Amr Ashraf Mubarak**
- GitHub: [@amrrdev](https://github.com/amrrdev)
- LinkedIn: [amramubarak](https://linkedin.com/in/amramubarak)
- Email: amrrdev@gmail.com

---

**Built with ❤️ to learn distributed systems and search technology**
