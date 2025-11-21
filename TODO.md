# TODO

### 生产级微服务治理 (Governance & Resiliency)
**目标**：从“单体脚手架”进化为“微服务框架”，具备在复杂分布式环境下的生存能力和管控能力。

| 步骤 ID | 模块 (Module) | 功能/优化点 | 需求描述与实现思路 | 验收标准 (Acceptance Criteria) |
| :--- | :--- | :--- | :--- | :--- |
| **1** | `registry` | **服务注册与发现 (Service Discovery)** | **缺口**: 服务是孤岛。<br>**思路**: 定义 `Registry` 接口。`Appx` 启动成功后自动注册实例 (IP/Port/Meta)，停止时注销。适配 Etcd/Consul/Nacos。 | 服务启动后，在注册中心控制台能看到节点信息；服务 Kill 后自动下线。 |
| **2** | `client` | **富客户端 (RPC Client) 集成** | **缺口**: `http.Client` 简陋，无负载均衡。<br>**思路**: 封装 `HttpClient` 和 `GrpcClient`，集成 Resolver (查注册中心) 和 Balancer (P2C/RoundRobin)。 | 客户端能根据服务名（如 `http://user-service`）自动寻址并轮询多个后端实例。 |
| **3** | `resiliency` | **熔断与降级 (Circuit Breaker)** | **缺口**: 下游故障导致级联雪崩。<br>**思路**: 在 `client` 层集成熔断器（如 Google SRE 算法）。当错误率超标时，自动拒绝请求 (Fail Fast)。 | 模拟下游 100% 错误，客户端在短暂重试后直接返回“熔断开启”错误，不再消耗网络资源。 |
| **4** | `conf` | **动态配置热加载 (Hot Reload)** | **缺口**: 修改配置需重启。<br>**思路**: 扩展 `conf` 监听文件变动/配置中心推送。实现配置总线，组件实现 `OnConfigChange` 接口。 | 运行时修改 `log.level` 为 debug，应用无需重启，立即输出 Debug 日志。 |
| **5** | `ratelimit` | **自适应限流 (Adaptive Limiting)** | **缺口**: 静态限流阈值难调优。<br>**思路**: 实现基于 CPU 使用率或 BBR 算法的限流中间件。当 CPU > 80% 时自动丢弃低优先级请求。 | 压测时，随着 CPU 飙升，QPS 自动稳定在系统最大处理能力的临界点，服务不崩溃。 |

---

### 性能极致与生态扩展 (Performance & Ecosystem)
**目标**：针对高并发热点进行底层优化，并提供完善的工程化工具链。

| 步骤 ID | 模块 (Module) | 功能/优化点 | 需求描述与实现思路 | 验收标准 (Acceptance Criteria) |
| :--- | :--- | :--- | :--- | :--- |
| **1** | `tools` | **OpenAPI 生成与 CLI 工具** | **缺口**: 缺文档，缺脚手架。<br>**思路**: 1. 解析 AST 生成 Swagger JSON。2. 开发 `my-framework new` CLI 生成项目模板。 | 运行命令生成 `openapi.yaml`，导入 Swagger UI 可进行接口测试。 |
| **2** | `httpx` | **内容协商 (Content Negotiation)** | **缺口**: 强绑定 JSON。<br>**思路**: 抽象 `Serializer` 接口。根据 `Accept` 头自动选择 JSON/XML/Proto 序列化。 | 同一接口，`Accept: application/xml` 返回 XML，`Accept: application/json` 返回 JSON。 |
