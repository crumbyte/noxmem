# NoxMem

A terminal UI for live Go heap profiling. Connect it to your application's pprof endpoint and see the memory runtime state
that is otherwise scattered across profiling data — heap and stack usage, allocation samples with full traces, GC activity,
and memory pressure. Explore how your application uses memory as it happens. Supports: **Windows**, **macOS**, and **Linux**.

![full-preview!](/img/preview.png "full preview")

## Usage

Your target application must have pprof enabled:

```go
import _ "net/http/pprof"

go func() {
    log.Fatal(http.ListenAndServe("localhost:6060", nil))
}()
```

Then run:

```bash
noxmem --target http://localhost:6060
```

## How to read stats

Values in the `Heap` section are displayed with two percentages in parentheses: `5.71 MB (0.00% | ↓15.66%)`. The first value shows the change from the previous snapshot. The second percentage value shows the change since the `noxmem` session started.
A value that spikes and recovers each refresh will show a large first percentage but a small second one. A slow memory leak will show a small first percentage but a steadily growing second one over time.

* **Heap** - shows OS-obtained memory, currently allocated bytes, in-use spans, idle, and released memory. All values include delta from the previous snapshot and from session start.
* **Garbage Collector** - GC cycle count, forced GC count, last GC timestamp, next GC goal, max observed pause, and CPU fraction spent in GC. The pause graph shows the duration and timing of recent GC stop-the-world events.
* **Allocation Samples** - top allocation sites ranked by inuse memory or total allocated bytes. Each row represents a unique call stack, aggregated across samples. Columns include inuse bytes, total allocated bytes, allocation rate per refresh interval, sample count, object count, and the leaf function with its source location.
* **Sample Trace** - the full call stack for the selected allocation site, from the allocation point up to the goroutine root.

### Help (Memory terms)

| Stat                   | Explanation                                                                                                                                                                                                                   |
|------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **OS Memory Obtained** | The total memory the Go runtime has requested from the operating system for heap use. This is the ceiling. The runtime doesn't always use all of it immediately.                                                              |
| **Allocated**          | A number of bytes occupied by live objects. It drops after GC runs and dead objects are collected.                                                                                                                            |
| **Memory In-use**      | A number of bytes in spans that are currently active. A span is a chunk of memory the runtime manages internally. A span can be partially empty, but the span itself not yet returned to the idle pool. Always `≥ Allocated`. |
| **Idle**               | Spans that are no longer in use and are sitting in a pool waiting to be reused or returned to the OS.                                                                                                                         |
| **Released**           | The portion of idle memory that has already been returned to the OS.                                                                                                                                                          |
| **Fragmentation**      | The gap between `Allocated` and `In-use`. It shows the amount of memory the runtime holds in active spans but no live object is using.                                                                                        |
| **Next GC Goal **      | Shows the heap size required for triggering the next GC cycle.                                                                                                                                                                |
| **GC Pause**           | A stop-the-world event where the entire program freezes briefly while the GC does bookkeeping. Frequent long pauses are a sign of memory pressure.                                                                            |
| **Allocation Rate**    | How fast a particular code site is allocating memory per refresh interval. A site with high inuse but zero rate is stable retained memory. A site with high rate is actively driving GC pressure right now.                   |

The relationship between these values is always:

* `Allocated` ≤ `In-use` ≤ `OS Memory Obtained`
* `Released` ≤ `Idle` ≤ `OS Memory Obtained`

## Keybindings

| Key          | Action                              |
|--------------|-------------------------------------|
| `↑ / k`      | Move up                             |
| `↓ / j`      | Move down                           |
| `g / Home`   | Go to first row                     |
| `G / End`    | Go to last row                      |
| `Enter`      | Select sample, show trace           |
| `Esc`        | Deselect sample                     |
| `1 / 2 / 3`  | Sort by inuse / allocated / objects |
| `e`          | Open file in editor                 |
| `ctrl+f`     | Toggle sample filter                |
| `q / ctrl+c` | Quit                                |

## Flags

NoxDir accepts flags on a startup. Here's a list of currently available
CLI flags:

```
Usage:
  noxmem [flags]

Flags:
  -h, --help            help for noxmem
  -r, --rate int        
                        Rate specifies the refresh rate in seconds at which noxmem will fetch the target
                        application memory stats. For example, setting this value to 5 will result in 
                        noxmem fetching the pprof server stats every 5 seconds. Min allowed value is 1 second.
                        
                        Example: --rate=5 (default 2)
  -t, --target string   
                        Target is a mandatory parameter which specifies the application's pprof endpoint.
                        The pprof server must be up and running for noxmem to fetch heap/stack stats.
                        
                        Example: --target="http://localhost:6060"
```

## Note

Heap profiles are sampled, not exact. The Go runtime records one sample per `512KB` allocated by default.

* Allocation counts and byte values are approximations scaled from sample counts.
* Small short-lived allocations may not appear at all.
* The inuse and allocated columns will not match `runtime.MemStats` exactly, since they are directionally accurate, not
  precise measurements.
