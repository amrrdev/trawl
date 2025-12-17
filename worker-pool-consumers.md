# Auto-Scaling Workers - Deep Dive

## Why Auto-Scaling?

### The Problem with Fixed Workers

**Scenario 1: Low Load**

```
50 workers, 5 messages in queue
❌ 45 workers sitting idle (wasted memory: ~90MB)
❌ 45 goroutines doing nothing
```

**Scenario 2: High Load**

```
50 workers, 10,000 messages in queue
❌ Queue growing faster than processing
❌ Need more workers but stuck at 50
```

### The Solution: Dynamic Scaling

```
Low load:  2 workers  → Uses 4MB RAM
High load: 50 workers → Uses 100MB RAM, then scales back down
```

**Benefits:**

- 💰 **Save resources** when idle
- 🚀 **Handle spikes** automatically
- ⚡ **Respond to demand** in real-time

## How Auto-Scaling Works

### Architecture

```
RabbitMQ → Message Dispatcher → Internal Queue (taskChan)
                                        ↓
                              ┌─────────┴─────────┐
                              │   Worker Pool     │
                              │  (Dynamic Size)   │
                              └─────────┬─────────┘
                                        │
                              ┌─────────┴─────────┐
                              │   Auto-Scaler     │
                              │  (Monitors Load)  │
                              └───────────────────┘
```

### Key Components

1. **Message Dispatcher**

   - Receives from RabbitMQ
   - Puts messages in internal buffered channel
   - Non-blocking

2. **Worker Pool**

   - Dynamic number of goroutines (min to max)
   - Workers pull from internal channel
   - Auto-terminate when idle too long

3. **Auto-Scaler**
   - Runs every 5 seconds
   - Monitors queue length and worker status
   - Spawns or terminates workers

## Scaling Logic

### Scale-Up Triggers

```go
if queueLen > 10 && activeWorkers < maxWorkers {
    // Queue is building up, need more workers!
    workersNeeded = queueLen / 5  // 1 worker per 5 messages
    spawn(workersNeeded)
}
```

**Example:**

```
Queue: 50 messages, Workers: 5
→ Spawn 10 workers (50 ÷ 5 = 10)
→ New total: 15 workers
```

### Scale-Down Triggers

```go
if idleWorkers > minWorkers && activeWorkers > minWorkers {
    // Too many workers sitting idle
    // Workers auto-terminate after 30s idle
}
```

**Example:**

```
Active: 20 workers, Idle: 15 workers, Min: 2
→ 13 workers will self-terminate after 30s idle
→ New total: ~7 workers (2 min + 5 busy)
```

## Worker Lifecycle

```
   Worker Spawned
        ↓
   [IDLE] ← ─ ─ ─ ─ ─ ─ ─ ┐
        ↓                  │
   Wait for message        │
        ↓                  │
   Message arrives         │
        ↓                  │
   [BUSY]                  │
        ↓                  │
   Process PDF             │
        ↓                  │
   Complete                │
        ↓                  │
   Return to [IDLE] ─ ─ ─ ┘
        ↓
   30s timeout?
        ↓
   activeWorkers > minWorkers?
        ↓
   YES → Terminate
   NO  → Stay alive
```

## Real-World Behavior

### Morning Rush (0 → 1000 messages/min)

```
00:00 - Start: 2 workers (min)
      ✅ Processed: 0

00:05 - Queue: 100 messages
      📈 SCALE UP: Spawn 20 workers
      Active: 22 workers
      ✅ Processed: 50

00:10 - Queue: 200 messages (still incoming fast)
      📈 SCALE UP: Spawn 20 workers
      Active: 42 workers
      ✅ Processed: 450

00:15 - Queue: 50 messages (catching up)
      Active: 42 workers
      ✅ Processed: 1200

00:20 - Queue: 0 messages
      Active: 42 workers (all idle)
      ✅ Processed: 1500

00:25 - Workers timing out
      📉 SCALE DOWN: 30 workers terminated
      Active: 12 workers
      ✅ Processed: 1500

00:30 - More workers timing out
      📉 SCALE DOWN: 10 workers terminated
      Active: 2 workers (back to min)
      ✅ Processed: 1500
```

### Configuration Parameters

```go
type AutoScalingConsumer struct {
    minWorkers         int           // Always keep this many alive
    maxWorkers         int           // Never exceed this
    scaleUpThreshold   int           // Messages to trigger scale-up
    scaleDownIdle      time.Duration // Idle timeout before worker exits
    checkInterval      time.Duration // How often to evaluate scaling
}
```

**Tuning Guide:**

| Scenario       | minWorkers | maxWorkers | scaleUpThreshold | scaleDownIdle |
| -------------- | ---------- | ---------- | ---------------- | ------------- |
| Low traffic    | 1          | 10         | 5                | 30s           |
| Medium traffic | 5          | 50         | 10               | 60s           |
| High traffic   | 10         | 200        | 20               | 120s          |
| Bursty traffic | 2          | 100        | 5                | 30s           |
| Steady load    | 20         | 30         | 15               | 300s          |

## Running the Auto-Scaler

### Basic Usage

```bash
# Default: min=2, max=50
go run consumer/main.go

# Custom configuration
MIN_WORKERS=5 MAX_WORKERS=100 go run consumer/main.go
```

### Testing Scaling Behavior

```bash
# Terminal 1: Start consumer
go run consumer/main.go

# Terminal 2: Send burst of messages
for i in {1..100}; do
    go run producer/main.go &
done

# Watch the logs:
# 📈 SCALE UP: Queue has 80 messages, spawning 16 workers
# 👷 Worker 3 started
# 👷 Worker 4 started
# ...
# 📉 SCALE DOWN: 10 workers idle, removing 8 workers
# 👷 Worker 15 stopped (idle timeout)
```

### Expected Output

```
🚀 Auto-scaling consumer started
📊 Min workers: 2, Max workers: 50
👷 Worker 1 started
👷 Worker 2 started

📊 ==================== Stats ====================
   👷 Active Workers: 2 (idle: 2, busy: 0)
   📦 Internal Queue: 0 messages
   ✅ Processed: 0
   ❌ Failed: 0
📊 ===============================================

... messages arrive ...

📈 SCALE UP: Queue has 45 messages, spawning 9 workers (active: 2 -> 11)
👷 Worker 3 started
👷 Worker 4 started
...
✅ Worker 3: Completed pdf-001 in 4.2s
✅ Worker 5: Completed pdf-002 in 3.8s

📊 ==================== Stats ====================
   👷 Active Workers: 11 (idle: 3, busy: 8)
   📦 Internal Queue: 12 messages
   ✅ Processed: 33
   ❌ Failed: 2
📊 ===============================================

... queue cleared ...

👷 Worker 10 stopped (idle timeout, active: 7)
👷 Worker 9 stopped (idle timeout, active: 6)

📉 SCALE DOWN: 4 workers idle, removing 3 workers (active: 5 -> 2)

📊 ==================== Stats ====================
   👷 Active Workers: 2 (idle: 2, busy: 0)
   📦 Internal Queue: 0 messages
   ✅ Processed: 100
   ❌ Failed: 3
📊 ===============================================
```

## Advanced: Metrics-Based Scaling

For production, use actual RabbitMQ queue metrics:

```go
func (c *AutoScalingConsumer) getQueueDepth() (int, error) {
    // Query RabbitMQ Management API
    resp, err := http.Get("http://localhost:15672/api/queues/%2F/pdf_processing")
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    var queue struct {
        Messages int `json:"messages"`
    }

    json.NewDecoder(resp.Body).Decode(&queue)
    return queue.Messages, nil
}

func (c *AutoScalingConsumer) evaluateScaling() {
    queueDepth, _ := c.getQueueDepth()

    // Scale based on actual RabbitMQ queue, not internal buffer
    if queueDepth > c.scaleUpThreshold {
        c.scaleUp()
    }
}
```

## Comparison: Fixed vs Auto-Scaling

### Memory Usage Over 24 Hours

```
Fixed (50 workers):
RAM: ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓ (100MB constant)

Auto-scaling (2-50 workers):
RAM: ▓▓____▓▓▓▓▓▓▓▓____▓▓ (avg 40MB, peaks 100MB)
     └─idle─┘└─busy─┘└─idle─┘
```

**Savings: 60MB average = 60% reduction!**

### Response Time

```
Scenario: 1000 messages arrive suddenly

Fixed (10 workers):
  - Immediate processing: 10 messages/second
  - Time to clear: 100 seconds

Auto-scaling (2-50 workers):
  - 0-5s: 2 workers = 2 msg/sec (10 processed)
  - 5-10s: Scales to 20 workers = 20 msg/sec (100 processed)
  - 10-15s: Scales to 50 workers = 50 msg/sec (250 processed)
  - 15-27s: 50 workers clear remaining 640
  - Total time: 27 seconds

Fixed would take 100s, Auto-scaling takes 27s!
```

## Production Best Practices

1. **Set Reasonable Limits**

   ```go
   minWorkers: 5  // Don't go too low (startup cost)
   maxWorkers: 200 // Don't exceed your RAM/CPU
   ```

2. **Monitor Metrics**

   ```go
   - activeWorkers (current count)
   - queueDepth (messages waiting)
   - processingRate (msgs/second)
   - scaleUpEvents (how often scaling up)
   - scaleDownEvents (how often scaling down)
   ```

3. **Tune for Your Workload**

   - **Predictable load**: Narrow range (10-20 workers)
   - **Bursty traffic**: Wide range (2-100 workers)
   - **24/7 steady**: Higher minimum (20-50 workers)

4. **Add Cooldown Periods**

   ```go
   lastScaleUp := time.Now()

   if time.Since(lastScaleUp) < 30*time.Second {
       return // Don't scale too frequently
   }
   ```

5. **Graceful Degradation**
   ```go
   if activeWorkers == maxWorkers && queueDepth > 1000 {
       log.Warn("At max capacity, consider adding more consumer instances")
   }
   ```

## Summary

**Auto-scaling gives you:**

- ✅ **Lower costs** when idle (2 workers vs 50)
- ✅ **Better performance** during spikes (scales to 50+ automatically)
- ✅ **Automatic adaptation** to load patterns
- ✅ **Resource efficiency** (no waste)

**Trade-offs:**

- ⚠️ More complex code
- ⚠️ Scale-up delay (5-10 seconds)
- ⚠️ Requires tuning for your workload

**Use fixed workers when:**

- Load is predictable and constant
- You need instant response (no scale-up delay)
- Simple > complex for your use case

**Use auto-scaling when:**

- Load varies throughout the day
- You want to optimize costs
- Handling unpredictable traffic spikes
- Processing millions of messages efficiently
