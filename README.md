# Trawl

> A distributed, scalable search engine built from the ground up in Go

## What is This?

Trawl is a production-grade search engine that demonstrates how systems like Elasticsearch work under the hood. It processes millions of documents, builds inverted indexes, and returns ranked search results in milliseconds.

## Core Concepts

- **Inverted Index**: Maps words to documents for O(log n) search performance
- **Distributed Architecture**: Sharded across multiple ScyllaDB nodes for horizontal scaling
- **Async Processing**: RabbitMQ-powered job queues for non-blocking document indexing
- **Microservices**: Independent services for auth, search, indexing, and analytics

## Tech Stack

**Backend**: Go (Golang)  
**Databases**: PostgreSQL, ScyllaDB  
**Message Queue**: RabbitMQ  
**Storage**: MinIO (S3-compatible)  
**Gateway**: Nginx

## Architecture

```
Users → Nginx → Microservices → [Auth, Search, Indexing, Analytics]
                       ↓
        [PostgreSQL, ScyllaDB, RabbitMQ, MinIO]
```

## What You'll Find Here

- Full-text search with TF-IDF ranking
- JWT authentication
- Distributed query coordination
- Async document processing
- Real-time analytics

---

Built to understand distributed systems, search algorithms, and microservice patterns.
