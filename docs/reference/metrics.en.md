This chapter comprehensively introduces all the metrics exported by EgressGateway,
and explains the meaning of each metric based on the metric type and description.

## Controller metrics

| Name                                            | Type      | Description                                                                                               |
|-------------------------------------------------|-----------|-----------------------------------------------------------------------------------------------------------|
| `certwatcher_read_certificate_errors_total`     | counter   | Total number of certificate read errors.                                                                  |
| `certwatcher_read_certificate_total`            | counter   | Total number of certificate reads.                                                                        |
| `controller_runtime_active_workers`             | gauge     | Number of currently used workers per controller.                                                          |
| `controller_runtime_max_concurrent_reconciles`  | gauge     | Maximum number of concurrent reconciles per controller.                                                   |
| `controller_runtime_reconcile_errors_total`     | counter   | Total number of reconciliation errors per controller.                                                     |
| `controller_runtime_reconcile_time_seconds`     | histogram | Length of time per reconciliation per controller.                                                         |
| `controller_runtime_reconcile_total`            | counter   | Total number of reconciliations per controller.                                                           |
| `controller_runtime_webhook_latency_seconds`    | histogram | Histogram of the latency of processing admission requests.                                                |
| `controller_runtime_webhook_requests_in_flight` | gauge     | Current number of admission requests being served.                                                        |
| `controller_runtime_webhook_requests_total`     | counter   | Total number of admission requests by HTTP status code.                                                   |
| `egress_ip_allocate_next_restore_calls`         | counter   | Total number of number of IP allocate next calls for restore operations.                                  |
| `egress_ip_allocate_release_calls`              | counter   | Total number of number of IP release calls.                                                               |
| `egress_mark_allocate_next_calls`               | counter   | Total number of mark allocate next count calls.                                                           |
| `egress_mark_release_calls`                     | counter   | Total number of mark release calls.                                                                       |
| `go_gc_duration_seconds`                        | summary   | A summary of the pause duration of garbage collection cycles.                                             |
| `go_goroutines`                                 | gauge     | Number of goroutines that currently exist.                                                                |
| `go_info`                                       | gauge     | Information about the Go environment.                                                                     |
| `go_memstats_alloc_bytes`                       | gauge     | Number of bytes allocated and still in use.                                                               |
| `go_memstats_alloc_bytes_total`                 | counter   | Total number of bytes allocated, even if freed.                                                           |
| `go_memstats_buck_hash_sys_bytes`               | gauge     | Number of bytes used by the profiling bucket hash table.                                                  |
| `go_memstats_frees_total`                       | counter   | Total number of frees.                                                                                    |
| `go_memstats_gc_sys_bytes`                      | gauge     | Number of bytes used for garbage collection system metadata.                                              |
| `go_memstats_heap_alloc_bytes`                  | gauge     | Number of heap bytes allocated and still in use.                                                          |
| `go_memstats_heap_idle_bytes`                   | gauge     | Number of heap bytes waiting to be used.                                                                  |
| `go_memstats_heap_inuse_bytes`                  | gauge     | Number of heap bytes that are in use.                                                                     |
| `go_memstats_heap_objects`                      | gauge     | Number of allocated objects.                                                                              |
| `go_memstats_heap_released_bytes`               | gauge     | Number of heap bytes released to OS.                                                                      |
| `go_memstats_heap_sys_bytes`                    | gauge     | Number of heap bytes obtained from system.                                                                |
| `go_memstats_last_gc_time_seconds`              | gauge     | Number of seconds since 1970 of last garbage collection.                                                  |
| `go_memstats_lookups_total`                     | counter   | Total number of pointer lookups.                                                                          |
| `go_memstats_mallocs_total`                     | counter   | Total number of mallocs.                                                                                  |
| `go_memstats_mcache_inuse_bytes`                | gauge     | Number of bytes in use by mcache structures.                                                              |
| `go_memstats_mcache_sys_bytes`                  | gauge     | Number of bytes used for mcache structures obtained from system.                                          |
| `go_memstats_mspan_inuse_bytes`                 | gauge     | Number of bytes in use by mspan structures.                                                               |
| `go_memstats_mspan_sys_bytes`                   | gauge     | Number of bytes used for mspan structures obtained from system.                                           |
| `go_memstats_next_gc_bytes`                     | gauge     | Number of heap bytes when next garbage collection will take place.                                        |
| `go_memstats_other_sys_bytes`                   | gauge     | Number of bytes used for other system allocations.                                                        |
| `go_memstats_stack_inuse_bytes`                 | gauge     | Number of bytes in use by the stack allocator.                                                            |
| `go_memstats_stack_sys_bytes`                   | gauge     | Number of bytes obtained from system for stack allocator.                                                 |
| `go_memstats_sys_bytes`                         | gauge     | Number of bytes obtained from system.                                                                     |
| `go_threads`                                    | gauge     | Number of OS threads created.                                                                             |
| `leader_election_master_status`                 | gauge     | Gauge of if the reporting system is master of the relevant lease, 0 indicates backup, 1 indicates master. |
| `process_cpu_seconds_total`                     | counter   | Total user and system CPU time spent in seconds.                                                          |
| `process_max_fds`                               | gauge     | Maximum number of open file descriptors.                                                                  |
| `process_open_fds`                              | gauge     | Number of open file descriptors.                                                                          |
| `process_resident_memory_bytes`                 | gauge     | Resident memory size in bytes.                                                                            |
| `process_start_time_seconds`                    | gauge     | Start time of the process since unix epoch in seconds.                                                    |
| `process_virtual_memory_bytes`                  | gauge     | Virtual memory size in bytes.                                                                             |
| `process_virtual_memory_max_bytes`              | gauge     | Maximum amount of virtual memory available in bytes.                                                      |
| `rest_client_requests_total`                    | counter   | Number of HTTP requests, partitioned by status code, method, and host.                                    |
| `workqueue_adds_total`                          | counter   | Total number of adds handled by workqueue.                                                                |
| `workqueue_depth`                               | gauge     | Current depth of workqueue.                                                                               |
| `workqueue_longest_running_processor_seconds`   | gauge     | How many seconds has the longest running processor for workqueue been running.                            |
| `workqueue_queue_duration_seconds`              | histogram | How long in seconds an item stays in workqueue before being requested.                                    |
| `workqueue_retries_total`                       | counter   | Total number of retries handled by workqueue.                                                             |
| `workqueue_unfinished_work_seconds`             | gauge     | How many seconds of work has been done that is in progress and hasn't been observed by work_duration.     |
| `workqueue_work_duration_seconds`               | histogram | How long in seconds processing an item from workqueue takes.                                              |

## Agent metrics

| Name                                           | Type      | Description                                                                                          |
|------------------------------------------------|-----------|------------------------------------------------------------------------------------------------------|
| `certwatcher_read_certificate_errors_total`    | counter   | Total number of certificate read errors                                                              |
| `certwatcher_read_certificate_total`           | counter   | Total number of certificate reads                                                                    |
| `controller_runtime_active_workers`            | gauge     | Number of currently used workers per controller                                                      |
| `controller_runtime_max_concurrent_reconciles` | gauge     | Maximum number of concurrent reconciles per controller                                               |
| `controller_runtime_reconcile_errors_total`    | counter   | Total number of reconciliation errors per controller                                                 |
| `controller_runtime_reconcile_time_seconds`    | histogram | Length of time per reconciliation per controller                                                     |
| `controller_runtime_reconcile_total`           | counter   | Total number of reconciliations per controller                                                       |
| `go_gc_duration_seconds`                       | summary   | A summary of the pause duration of garbage collection cycles                                         |
| `go_goroutines`                                | gauge     | Number of goroutines that currently exist                                                            |
| `go_info`                                      | gauge     | Information about the Go environment                                                                 |
| `go_memstats_alloc_bytes`                      | gauge     | Number of bytes allocated and still in use                                                           |
| `go_memstats_alloc_bytes_total`                | counter   | Total number of bytes allocated, even if freed                                                       |
| `go_memstats_buck_hash_sys_bytes`              | gauge     | Number of bytes used by the profiling bucket hash table                                              |
| `go_memstats_frees_total`                      | counter   | Total number of frees                                                                                |
| `go_memstats_gc_sys_bytes`                     | gauge     | Number of bytes used for garbage collection system metadata                                          |
| `go_memstats_heap_alloc_bytes`                 | gauge     | Number of heap bytes allocated and still in use                                                      |
| `go_memstats_heap_idle_bytes`                  | gauge     | Number of heap bytes waiting to be used                                                              |
| `go_memstats_heap_inuse_bytes`                 | gauge     | Number of heap bytes that are in use                                                                 |
| `go_memstats_heap_objects`                     | gauge     | Number of allocated objects                                                                          |
| `go_memstats_heap_released_bytes`              | gauge     | Number of heap bytes released to OS                                                                  |
| `go_memstats_heap_sys_bytes`                   | gauge     | Number of heap bytes obtained from system                                                            |
| `go_memstats_last_gc_time_seconds`             | gauge     | Number of seconds since 1970 of last garbage collection                                              |
| `go_memstats_lookups_total`                    | counter   | Total number of pointer lookups                                                                      |
| `go_memstats_mallocs_total`                    | counter   | Total number of mallocs                                                                              |
| `go_memstats_mcache_inuse_bytes`               | gauge     | Number of bytes in use by mcache structures                                                          |
| `go_memstats_mcache_sys_bytes`                 | gauge     | Number of bytes used for mcache structures obtained from system                                      |
| `go_memstats_mspan_inuse_bytes`                | gauge     | Number of bytes in use by mspan structures                                                           |
| `go_memstats_mspan_sys_bytes`                  | gauge     | Number of bytes used for mspan structures obtained from system                                       |
| `go_memstats_next_gc_bytes`                    | gauge     | Number of heap bytes when next garbage collection will take place                                    |
| `go_memstats_other_sys_bytes`                  | gauge     | Number of bytes used for other system allocations                                                    |
| `go_memstats_stack_inuse_bytes`                | gauge     | Number of bytes in use by the stack allocator                                                        |
| `go_memstats_stack_sys_bytes`                  | gauge     | Number of bytes obtained from system for stack allocator                                             |
| `go_memstats_sys_bytes`                        | gauge     | Number of bytes obtained from system                                                                 |
| `go_threads`                                   | gauge     | Number of OS threads created                                                                         |
| `iptables_chains`                              | gauge     | Number of active iptables chains                                                                     |
| `iptables_lines_executed`                      | counter   | Number of iptables rule updates executed                                                             |
| `iptables_lock_acquire_secs`                   | summary   | Time in seconds that it took to acquire the iptables lock(s)                                         |
| `iptables_lock_retries`                        | counter   | Number of times the iptables lock was held by someone else and retries were needed                   |
| `iptables_restore_calls`                       | counter   | Number of iptables-restore calls                                                                     |
| `iptables_restore_errors`                      | counter   | Number of iptables-restore errors                                                                    |
| `iptables_rules`                               | gauge     | Number of active iptables rules                                                                      |
| `iptables_save_calls`                          | counter   | Number of iptables-save calls                                                                        |
| `iptables_save_errors`                         | counter   | Number of iptables-save errors                                                                       |
| `process_cpu_seconds_total`                    | counter   | Total user and system CPU time spent in seconds                                                      |
| `process_max_fds`                              | gauge     | Maximum number of open file descriptors                                                              |
| `process_open_fds`                             | gauge     | Number of open file descriptors                                                                      |
| `process_resident_memory_bytes`                | gauge     | Resident memory size in bytes                                                                        |
| `process_start_time_seconds`                   | gauge     | Start time of the process since unix epoch in seconds                                                |
| `process_virtual_memory_bytes`                 | gauge     | Virtual memory size in bytes                                                                         |
| `process_virtual_memory_max_bytes`             | gauge     | Maximum amount of virtual memory available in bytes                                                  |
| `rest_client_requests_total`                   | counter   | Number of HTTP requests, partitioned by status code, method, and host                                |
| `workqueue_adds_total`                         | counter   | Total number of adds handled by workqueue                                                            |
| `workqueue_depth`                              | gauge     | Current depth of workqueue                                                                           |
| `workqueue_longest_running_processor_seconds`  | gauge     | How many seconds has the longest running processor for workqueue been running                        |
| `workqueue_queue_duration_seconds`             | histogram | How long in seconds an item stays in workqueue before being requested                                |
| `workqueue_retries_total`                      | counter   | Total number of retries handled by workqueue                                                         |
| `workqueue_unfinished_work_seconds`            | gauge     | How many seconds of work has been done that is in progress and hasn't been observed by work_duration |
| `workqueue_work_duration_seconds`              | histogram | How long in seconds processing an item from workqueue takes                                          |
