# Trawl

Distributed search engine built in Go. Index millions of documents, search them in milliseconds.

## Overview

Trawl implements full-text search using inverted indexes and distributed storage. Documents get tokenized and indexed asynchronously. Queries run in parallel across sharded nodes and return ranked results using TF-IDF scoring.

Built with Go, ScyllaDB, RabbitMQ, PostgreSQL, and MinIO.

## Architecture

Three core services:

auth - Authentication and user management with JWT
search - Query processing, ranking, and result aggregation
indexing - Document ingestion and index building

Services communicate through Nginx. PostgreSQL handles user data. ScyllaDB stores the inverted index across multiple shards. MinIO manages document storage. RabbitMQ coordinates async indexing jobs.

When you upload a document, it gets stored in MinIO and queued for processing. Workers extract text, tokenize it, and write index entries to ScyllaDB shards. Searches query all shards in parallel, calculate relevance scores, and merge results.

## Core Concepts

### Inverted Index

Standard structure maps documents to their words. An inverted index reverses this - it maps words to documents containing them.

```
go          → [doc1, doc2]
distributed → [doc2]
systems     → [doc2, doc5]
```

When someone searches "distributed systems", we look up both words, find their document sets, intersect them, and return matches. No sequential scanning required.

### Sharding

Index data distributes across ScyllaDB nodes using consistent hashing on the term. Each shard handles a subset of the vocabulary. Queries hit all shards concurrently. The search service merges and ranks results globally.

Replication handles failures. Each shard maintains replicas on separate nodes. If a primary fails, a replica takes over.

### TF-IDF Scoring

Relevance combines two metrics:

Term frequency measures how often a word appears in a document relative to document length.

Inverse document frequency measures how rare a word is across the entire corpus. Common words like "the" score lower than distinctive terms like "kubernetes".

Final score multiplies these values. Documents with distinctive terms matching the query rank highest.

### Async Processing

Indexing happens in the background. Upload returns immediately with a job ID. Workers consume from RabbitMQ, extract text, tokenize, and build index entries. Failed jobs retry three times before moving to a dead letter queue.

## Technology Stack

Go for service implementation. Concurrency primitives handle parallel shard queries efficiently.

ScyllaDB for the inverted index. Low-latency reads, horizontal scalability, and automatic sharding.

RabbitMQ for job coordination. Reliable delivery, retry logic, and dead letter queues.

PostgreSQL for user accounts and authentication data. Relational model fits this use case.

MinIO for document storage. S3-compatible object storage.

Nginx for load balancing and reverse proxy. Handles SSL termination and rate limiting.

## Running Locally

Start infrastructure:

```bash
docker-compose up -d
```

Initialize database:

```bash
cd services/auth && go run cmd/migrate/main.go up
```

Launch services:

```bash
cd services/auth && go run cmd/api/main.go      # Terminal 1
cd services/search && go run cmd/api/main.go    # Terminal 2
cd services/indexing && go run cmd/api/main.go  # Terminal 3
```

Create an account:

```bash
curl -X POST http://localhost/api/v1/auth/register \
  -d '{"email":"user@example.com","password":"secure_pass"}'
```

Authenticate:

```bash
curl -X POST http://localhost/api/v1/auth/login \
  -d '{"email":"user@example.com","password":"secure_pass"}'
```

Index a document:

```bash
curl -X POST http://localhost/api/v1/documents \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@document.pdf"
```

Run a search:

```bash
curl -X POST http://localhost/api/v1/search \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "distributed systems"}'
```

## Performance

Target metrics:

- Query latency p50: <50ms
- Query latency p99: <200ms
- Indexing throughput: 1000+ documents/second
- Concurrent queries: 10,000+ QPS
- Index capacity: 1M+ documents per shard
