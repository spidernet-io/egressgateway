本章节全面介绍了 EgressGateway 导出的所有指标，并根据指标类型和描述解释了每个指标的含义。

## Controller metrics

| Name                                            | Type      | Description                      |
|-------------------------------------------------|-----------|----------------------------------|
| `certwatcher_read_certificate_errors_total`     | counter   | 证书读取错误总数                         |
| `certwatcher_read_certificate_total`            | counter   | 证书读取总数                           |
| `controller_runtime_active_workers`             | gauge     | 每个 controller 当前使用的工作线程数         |
| `controller_runtime_max_concurrent_reconciles`  | gauge     | 每个 controller 的最大并发 reconcile 数量 |
| `controller_runtime_reconcile_errors_total`     | counter   | 每个 controller 的 reconcile 错误总数   |
| `controller_runtime_reconcile_time_seconds`     | histogram | 每个 controller 的对账时间长度            |
| `controller_runtime_reconcile_total`            | counter   | 每个 controller 的对账总数              |
| `controller_runtime_webhook_latency_seconds`    | histogram | 处理准入请求的延迟的直方图                    |
| `controller_runtime_webhook_requests_in_flight` | gauge     | 当前正在处理的准入请求数量                    |
| `controller_runtime_webhook_requests_total`     | counter   | 按 HTTP 状态码分类的准入请求总数              |
| `egress_ip_allocate_next_restore_calls`         | counter   | 用于恢复操作的IP分配（allocate next）调用总数   |
| `egress_ip_allocate_release_calls`              | counter   | IP释放（release）调用总数                |
| `egress_mark_allocate_next_calls`               | counter   | 标记分配（mark allocate next）计数调用总数   |
| `egress_mark_release_calls`                     | counter   | 标记释放（mark release）调用总数           |
| `go_gc_duration_seconds`                        | summary   | 垃圾收集周期暂停时间的总结                    |
| `go_goroutines`                                 | gauge     | 当前存在的goroutines数量                |
| `go_info`                                       | gauge     | Go 环境的信息                         |
| `go_memstats_alloc_bytes`                       | gauge     | 已分配且仍在使用的字节数                     |
| `go_memstats_alloc_bytes_total`                 | counter   | 总共分配的字节数，即使已释放                   |
| `go_memstats_buck_hash_sys_bytes`               | gauge     | 由分析桶哈希表使用的字节数                    |
| `go_memstats_frees_total`                       | counter   | 总释放次数                            |
| `go_memstats_gc_sys_bytes`                      | gauge     | 用于垃圾收集系统元数据的字节数                  |
| `go_memstats_heap_alloc_bytes`                  | gauge     | 堆上已分配且仍在使用的字节数                   |
| `go_memstats_heap_idle_bytes`                   | gauge     | 等待使用的堆字节数                        |
| `go_memstats_heap_inuse_bytes`                  | gauge     | 正在使用的堆字节数                        |
| `go_memstats_heap_objects`                      | gauge     | 已分配对象的数量                         |
| `go_memstats_heap_released_bytes`               | gauge     | 释放给操作系统的堆字节数                     |
| `go_memstats_heap_sys_bytes`                    | gauge     | 从系统获得的堆字节数                       |
| `go_memstats_last_gc_time_seconds`              | gauge     | 上次垃圾收集以来的秒数（自 1970 年起）           |
| `go_memstats_lookups_total`                     | counter   | 指针查找的总数                          |
| `go_memstats_mallocs_total`                     | counter   | mallocs 的总数                      |
| `go_memstats_mcache_inuse_bytes`                | gauge     | mcache 结构正在使用的字节数                |
| `go_memstats_mcache_sys_bytes`                  | gauge     | 从系统获取的用于 mcache 结构的字节数           |
| `go_memstats_mspan_inuse_bytes`                 | gauge     | mspan 结构正在使用的字节数                 |
| `go_memstats_mspan_sys_bytes`                   | gauge     | 从系统获取的用于 mspan 结构的字节数            |
| `go_memstats_next_gc_bytes`                     | gauge     | 下次垃圾收集将发生时的堆字节数                  |
| `go_memstats_other_sys_bytes`                   | gauge     | 用于其他系统分配的字节数                     |
| `go_memstats_stack_inuse_bytes`                 | gauge     | 栈分配器正在使用的字节数                     |
| `go_memstats_stack_sys_bytes`                   | gauge     | 从系统获得的用于栈分配器的字节数                 |
| `go_memstats_sys_bytes`                         | gauge     | 从系统获得的字节数                        |
| `go_threads`                                    | gauge     | 创建的操作系统线程数                       |
| `leader_election_master_status`                 | gauge     | 报告系统是否为多副本选举主节点，0 表示备份，1 表示      |
| `process_cpu_seconds_total`                     | counter   | 在秒内消耗的总用户和系统 CPU 时间              |
| `process_max_fds`                               | gauge     | 最大打开文件描述符数量                      |
| `process_open_fds`                              | gauge     | 打开的文件描述符数量                       |
| `process_resident_memory_bytes`                 | gauge     | 常驻内存大小（字节）                       |
| `process_start_time_seconds`                    | gauge     | 进程自 Unix 纪元以来的启动时间（秒）            |
| `process_virtual_memory_bytes`                  | gauge     | 虚拟内存大小（字节）                       |
| `process_virtual_memory_max_bytes`              | gauge     | 可用的最大虚拟内存量（字节）                   |
| `rest_client_requests_total`                    | counter   | 按状态码、方法和主机划分的 HTTP 请求数量          |
| `workqueue_adds_total`                          | counter   | 由工作队列处理的添加操作总数                   |
| `workqueue_depth`                               | gauge     | 工作队列的当前深度                        |
| `workqueue_longest_running_processor_seconds`   | gauge     | 工作队列中最长运行处理器运行的秒数                |
| `workqueue_queue_duration_seconds`              | histogram | 工作队列中的对象在被请求之前停留的秒数              |
| `workqueue_retries_total`                       | counter   | 由工作队列处理的重试总数                     |
| `workqueue_unfinished_work_seconds`             | gauge     | 正在进行且未被 work_duration 观察到的工作秒数   |
| `workqueue_work_duration_seconds`               | histogram | 从工作队列处理一个对象所需的秒数                 |

## Agent metrics

| Name                                           | Type      | Description                                    |
|------------------------------------------------|-----------|------------------------------------------------|
| `certwatcher_read_certificate_errors_total`    | counter   | 证书读取错误总数                                       |
| `certwatcher_read_certificate_total`           | counter   | 证书读取总数                                         |
| `controller_runtime_active_workers`            | gauge     | 每个 controller 当前使用的工作者数                        |
| `controller_runtime_max_concurrent_reconciles` | gauge     | 每个 controller 允许的最大并发协调数                       |
| `controller_runtime_reconcile_errors_total`    | counter   | 每个 controller 的协调错误总数                          |
| `controller_runtime_reconcile_time_seconds`    | histogram | 每个 controller 每次协调的时间长度                        |
| `controller_runtime_reconcile_total`           | counter   | 每个 controller 的协调总数                            |
| `go_gc_duration_seconds`                       | summary   | 垃圾回收周期暂停持续时间的摘要                                |
| `go_goroutines`                                | gauge     | 当前存在的 goroutine 数量                             |
| `go_info`                                      | gauge     | Go 环境信息                                        |
| `go_memstats_alloc_bytes`                      | gauge     | 分配且仍在使用的字节数                                    |
| `go_memstats_alloc_bytes_total`                | counter   | 分配的总字节数，即使已释放                                  |
| `go_memstats_buck_hash_sys_bytes`              | gauge     | 分析桶哈希表使用的字节数                                   |
| `go_memstats_frees_total`                      | counter   | 释放的总次数                                         |
| `go_memstats_gc_sys_bytes`                     | gauge     | 用于垃圾回收系统元数据的字节数                                |
| `go_memstats_heap_alloc_bytes`                 | gauge     | 堆分配且仍在使用的字节数                                   |
| `go_memstats_heap_idle_bytes`                  | gauge     | 等待使用的堆字节数                                      |
| `go_memstats_heap_inuse_bytes`                 | gauge     | 正在使用的堆字节数                                      |
| `go_memstats_heap_objects`                     | gauge     | 分配的对象数量                                        |
| `go_memstats_heap_released_bytes`              | gauge     | 释放给操作系统的堆字节数                                   |
| `go_memstats_heap_sys_bytes`                   | gauge     | 从系统获得的堆字节数                                     |
| `go_memstats_last_gc_time_seconds`             | gauge     | 上次垃圾回收以来的秒数（自1970年起）                           |
| `go_memstats_lookups_total`                    | counter   | 指针查找总数                                         |
| `go_memstats_mallocs_total`                    | counter   | malloc 的总次数                                    |
| `go_memstats_mcache_inuse_bytes`               | gauge     | mcache 结构使用的字节数                                |
| `go_memstats_mcache_sys_bytes`                 | gauge     | 从系统获得的用于 mcache 结构的字节数                         |
| `go_memstats_mspan_inuse_bytes`                | gauge     | mspan结构使用的字节数                                  |
| `go_memstats_mspan_sys_bytes`                  | gauge     | 从系统获得的用于 mspan 结构的字节数                          |
| `go_memstats_next_gc_bytes`                    | gauge     | 下一次垃圾回收将发生时的堆字节数                               |
| `go_memstats_other_sys_bytes`                  | gauge     | 用于其他系统分配的字节数                                   |
| `go_memstats_stack_inuse_bytes`                | gauge     | 堆栈分配器使用的字节数                                    |
| `go_memstats_stack_sys_bytes`                  | gauge     | 从系统为堆栈分配器获得的字节数                                |
| `go_memstats_sys_bytes`                        | gauge     | 从系统获得的字节数                                      |
| `go_threads`                                   | gauge     | 创建的操作系统线程数                                     |
| `iptables_chains`                              | gauge     | 活动的 iptables 链数                                |
| `iptables_lines_executed`                      | counter   | 执行的 iptables 规则更新次数                            |
| `iptables_lock_acquire_secs`                   | summary   | 获取 iptables 锁所需的时间（秒）                          |
| `iptables_lock_retries`                        | counter   | iptables 锁被他人持有且需要重试的次数                        |
| `iptables_restore_calls`                       | counter   | iptables-restore 调用次数                          |
| `iptables_restore_errors`                      | counter   | iptables-restore 错误数                           |
| `iptables_rules`                               | gauge     | 活动的 iptables 规则数                               |
| `iptables_save_calls`                          | counter   | iptables-save 调用次数                             |
| `iptables_save_errors`                         | counter   | iptables-save 错误数                              |
| `process_cpu_seconds_total`                    | counter   | 用户和系统 CPU 时间总计（秒）                              |
| `process_max_fds`                              | gauge     | 最大打开文件描述符数                                     |
| `process_open_fds`                             | gauge     | 打开的文件描述符数                                      |
| `process_resident_memory_bytes`                | gauge     | 常驻内存大小（字节）                                     |
| `process_start_time_seconds`                   | gauge     | 进程自Unix纪元起的启动时间（秒）                             |
| `process_virtual_memory_bytes`                 | gauge     | 虚拟内存大小（字节）                                     |
| `process_virtual_memory_max_bytes`             | gauge     | 可用的最大虚拟内存量（字节）                                 |
| `rest_client_requests_total`                   | counter   | 按状态码、方法和主机划分的HTTP请求总数                          |
| `workqueue_adds_total`                         | counter   | workqueue 处理的添加总数                              |
| `workqueue_depth`                              | gauge     | workqueue 的当前深度                                |
| `workqueue_longest_running_processor_seconds`  | gauge     | workqueue 中运行时间最长的处理器已运行的秒数                    |
| `workqueue_queue_duration_seconds`             | histogram | 对象在 workqueue 中停留后被请求的秒数                       |
| `workqueue_retries_total`                      | counter   | workqueue 处理的重试总数                              |
| `workqueue_unfinished_work_seconds`            | gauge     | workqueue 正在进行的工作已进行的秒数，但尚未由 work_duration 观察到 |
| `workqueue_work_duration_seconds`              | histogram | 从 workqueue 处理一个对象所需的时间（秒）                     |
