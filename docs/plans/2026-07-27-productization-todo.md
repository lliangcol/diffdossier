# DiffDossier 产品化闭环 TODO

本文将 2026-07-27 的全面评审拆解为可跨多轮对话执行、验收和追踪的原子任务。它是执行计划，不是完成声明；除“当前基线”中明确列出的只读观察外，任何分析结论都不能替代任务验收证据。

## 计划元数据

- 计划版本：`1.13`
- 计划状态：`active`
- 创建时间：`2026-07-27T23:58:02+08:00`
- 最近更新时间：`2026-07-28T00:43:36+08:00`
- 默认任务状态：`pending`
- 允许的任务状态：`pending`、`in_progress`、`completed`、`cancelled`
- 当前主目标：停止横向扩张，优先完成“理解、安装、运行、验证、分享”的产品闭环。
- 范围边界：本文件只规划工作；创建本文件不授权发布、修改 GitHub 设置、调用付费 Provider、公开数据、创建包管理渠道或执行其他外部写操作。

## 当前基线

以下观察用于修正原分析中已经过期的顺序，不代表相关领域已经完整关闭：

- 2026-07-28 复核时，本地 `main` 与本地记录的 `origin/main` 同指向 `3c46e62`；除本计划所在的未跟踪 `docs/plans/` 外未发现其他工作树变更。该远端跟踪引用不是对 GitHub 远端分支的实时证明。
- 本地存在带注释标签 `v0.1.0-beta.1`、`v0.1.0-beta.2`、`v0.1.0-beta.3`；2026-07-28 的只读 GitHub 查询也确认三个公开 Pre-release 仍存在。beta.3 页面列出六个平台归档、`SHA256SUMS`、SPDX SBOM、provenance 和 release manifest；资产存在不等于哈希、attestation 或安装验证已经完成。因此不得再把“创建 beta.1”作为未完成任务，也不得覆盖或重发已有标签。
- README、SUPPORT、安装文档、发布文档、平台兼容性文档和部分 checkpoint 仍包含“尚无公开 Release”“原生平台验证待完成”等旧描述。
- `GOVERNANCE.md` 把 `liuliang1` 同时描述为维护者和 GitHub 仓库所有者，而远端仓库路径为 `lliangcol/diffdossier`；两者是否代表同一人的不同身份必须由维护者确认，不能自动改写。
- `.github` 当前只有 CI 和 Release 工作流，未发现 Issue Form、PR Template 或 CODEOWNERS。
- CLI 当前未提供 `quickstart`、`next`、`explain`、guided workflow 或 Shell completion。
- `config.schema.json` 的 `review`、`state`、`risk` 仍是宽泛 object；`output-envelope.schema.json` 尚未按 `status` 约束 `data/error` 互斥；现有 Schema 测试主要检查 JSON 可解析及 `$schema`、`$id` 存在。
- 风险覆盖仍使用 `filepath.Match`，尚无独立的 DiffDossier Glob 规范。

后续任何轮次开始前都必须重新验证与所选任务有关的 live state；本节是历史基线，不是永久真相。

## 状态与执行协议

1. 每个任务只使用表格中的唯一 ID；不得复制 ID 或创建无 ID 的临时 TODO。
2. 开始任务时先把其状态改为 `in_progress`，并把 `updated_at` 改为实际 ISO 8601 时间；同一工作包内可以有多个 `in_progress`，但必须解释并行原因。
3. 任务内容、依赖或完成证据发生变化时，即使状态仍为 `pending`，也必须更新该行 `updated_at`；仅阅读且未改动任务定义时不更新时间。
4. 只有全部依赖处于 `completed`、验收证据已获得、相关文档已同步时，任务才能改为 `completed`。旧测试结果、旧 CI、交叉编译、Mock 或分析判断不能冒充当前验收证据。
5. 放弃任务时改为 `cancelled`，保留原行，并在执行日志记录原因、替代方案和批准人；依赖该任务的项目保持阻塞，直到计划明确改写依赖，`cancelled` 不得被当作 `completed`。
6. 已完成任务的证据若被后续变更或外部状态漂移失效，必须重新转为 `pending` 或 `in_progress`、更新时间并记录原因，不能继续保留虚假的 `completed`。
7. 修改任一实现、Schema、工作流、案例或文档后，之前针对旧快照的 review/test seal 自动失效；必须对新快照重新验证。
8. 每轮优先选择一个可独立验收的工作包。结束时更新任务状态、时间戳、证据链接或命令结果，并记录下一项未阻塞任务。
9. 外部写操作、付费 Provider、真实仓库数据公开、GitHub 设置修改、Release、包管理器发布和安全披露均需单独、精确授权；计划存在本身不构成授权。
10. `ready`、本地 PASS 或某阶段退出只说明声明范围内证据充分，不代表可以合并、发布或上线。
11. 阶段二、三、四中除各自入口任务本身外的实施与退出任务，分别把 `DD-P1-ENTRY-001`、`DD-P2-ENTRY-001`、`DD-P3-ENTRY-001` 视为共同前置依赖；为避免每行重复，该继承依赖只在阶段入口和退出任务中显式列出。入口未完成时可以调研，但不得把该阶段实施任务改为 `in_progress`。
12. 本计划授权的本地实现和验证不自动授权 `git add`、commit、push、创建 PR 或合并；这些交付动作必须针对当前精确 diff、分支和目标另行获得用户授权。

## 完成定义

任务的“完成证据”列是最低要求。默认还必须满足：

- 变更范围与任务内容一致，没有顺手扩展未批准功能；
- 受影响测试、静态检查、文档检查和跨平台检查按风险重新运行；
- 面向不可信输入或外部命令的变更包含失败路径与边界测试；
- 公共契约变更包含兼容性结论、迁移说明和 Changelog；
- 用户可见行为同时更新 README、命令帮助和主题文档；
- 外部状态只按实际取得的证据级别表述。

## 0. 计划控制与基线重验

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-CTRL-001 | 建立每轮执行前的基线检查清单：工作树、分支、HEAD、远端差异、已有 Release、相关 CI、任务依赖和仓库规则。 | completed | 2026-07-28T11:14:07+08:00 | 无 | 清单落盘，并在一次后续工作包中完整使用。 |
| DD-CTRL-002 | 为本计划建立状态更新和证据记录约定，确保任务状态变更与代码、文档或外部证据在同一轮变更集中记录；是否提交由用户另行授权。 | completed | 2026-07-28T11:15:12+08:00 | DD-CTRL-001 | 至少一次状态迁移记录包含旧状态、新状态、时间、证据和下一步，且没有擅自提交。 |
| DD-CTRL-003 | 定义产品化核心指标：首次成功时间、安装成功率、可复现案例数、支持仓库数、Provider 数、Schema 兼容率、误报率和 Release 节奏。 | completed | 2026-07-28T11:18:27+08:00 | 无 | 指标有定义、采集方式、基线、目标值和负责人。 |
| DD-CTRL-004 | 将分析中的每项结论分类为“已完成”“部分完成”“仍缺失”“需 live 验证”，避免把旧快照直接转成实施任务。 | completed | 2026-07-28T11:21:29+08:00 | DD-CTRL-001 | 形成带证据链接的差距矩阵，并映射到本计划 ID。 |
| DD-CTRL-005 | 建立决策记录规则：涉及公共契约、依赖、网络、命令执行、发布、安全或数据公开的设计先写 ADR 或等价决策记录。 | completed | 2026-07-28T11:24:11+08:00 | 无 | CONTRIBUTING 或规划文档引用该规则，且模板可用。 |

### 0.1 公共契约基线

`DD-SCH-001` 是阶段一发布事实所需的只读基线任务，明确属于阶段零，不继承阶段二入口依赖；后续 Schema 实施仍受 `DD-P1-ENTRY-001` 约束。

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-SCH-001 | 建立全部公共 Schema、Go 类型、生产者、消费者、当前版本和稳定级别清单。 | pending | 2026-07-28T00:36:19+08:00 | DD-CTRL-004 | 每个 Schema 有 owner、兼容范围和测试入口。 |

## 1. 阶段一：当前事实与 Beta 产品闭环（P0，目标 0～2 周）

### 1.1 身份、状态与文档一致性

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-DOC-001 | 由维护者确认 `liuliang1`、`lliangcol` 和 Liu Liang 的关系，以及各自应承担的仓库 owner、maintainer、版权和签署身份。 | completed | 2026-07-28T11:29:55+08:00 | DD-CTRL-001 | 有明确人工决定；不得仅依据用户名相似性推断。 |
| DD-DOC-002 | 按 DD-DOC-001 的决定统一 GOVERNANCE、NOTICE、SECURITY、SUPPORT、CONTRIBUTING 和 Release 文档中的身份表达。 | completed | 2026-07-28T11:35:46+08:00 | DD-DOC-001 | 全仓身份搜索无未解释冲突，文档 review 通过。 |
| DD-DOC-003 | 定义 checkpoint 元数据规范，至少包括 `Status`、`Captured-at`、`Source-commit`、`Superseded-by` 和“不可作为当前状态”的提示。 | completed | 2026-07-28T11:40:39+08:00 | DD-CTRL-005 | 规范文档和示例通过 review。 |
| DD-DOC-004 | 逐个分类 `docs/checkpoints/` 现有文件为 current、historical 或 superseded，并补齐 DD-DOC-003 元数据。 | pending | 2026-07-27T23:58:02+08:00 | DD-DOC-003 | 所有 checkpoint 均有状态；历史内容没有被当作当前事实引用。 |
| DD-DOC-005 | 依据最新原生 CI 与发布运行重建平台证据矩阵，分开 Tier、原生运行、race、跨编译、安装 smoke、未验证语义和最低版本。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-001 | 每项支持声明可追溯到具体 run、runner、版本和 commit。 |
| DD-DOC-006 | 更新 `docs/platform-compatibility.md`，删除已被证据推翻的“原生验证全部待完成”表述，同时保留 arm64、ACL、路径、Unicode 等真实缺口。 | pending | 2026-07-27T23:58:02+08:00 | DD-DOC-005 | 支持矩阵与当前 CI 证据一致，且不把跨编译提升为原生证据。 |
| DD-DOC-007 | 为 README、SUPPORT、install、release-process、SECURITY、CHANGELOG 建立发布状态对账矩阵，逐条标出当前事实、历史陈述、权威来源和待修正文案。 | pending | 2026-07-28T00:14:02+08:00 | DD-CTRL-004, DD-DOC-005 | 矩阵覆盖全部现时声明，且把“已有 beta、尚未 stable”与历史状态分开。 |
| DD-DOC-008 | 定义并统一 `ready`、`verified`、`finalized`、`approved`、`mergeable`、`released`、Tier 1/2 等术语。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-005 | 术语表发布，README 和主题文档使用一致。 |
| DD-DOC-009 | 增加离线文档漂移检查，校验 README、支持矩阵、仓库内声明的版本/Release 元数据和 checkpoint 元数据之间的可自动判定一致性；不得在普通 CI 中把网络查询当作稳定真相。 | pending | 2026-07-28T00:14:02+08:00 | DD-DOC-003, DD-DOC-007 | 本地测试与 CI 均能发现故意植入的仓库内状态冲突，且无需网络。 |
| DD-DOC-010 | 为无法自动验证的外部事实定义人工复核清单和有效期，避免把一次 GitHub 查询永久写成当前事实。 | pending | 2026-07-27T23:58:02+08:00 | DD-DOC-009 | 清单包含 owner、ruleset、PVR、Release、required checks 和支持平台。 |
| DD-DOC-011 | 按 DD-DOC-001 的决定更新 GitHub 仓库描述、链接或其他外部身份展示；没有需修改项时记录只读核对结论。 | pending | 2026-07-28T00:14:02+08:00 | DD-DOC-001, DD-GOV-001 | 外部展示与仓库文档一致；任何写操作均有单独授权和回读证据。 |

### 1.2 定位、README 与首次成功路径

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-UXD-001 | 确认对外定位为“面向大型 Git 变更的本地优先、Provider 中立的评审证据控制层”，并列出不做事项。 | pending | 2026-07-27T23:58:02+08:00 | DD-DOC-008 | 一句话定位和 3～5 条边界经维护者确认。 |
| DD-UXD-002 | 写入“small change first”原则：先判断能否安全拆分，只有无法合理拆分的大型变更才进入完整 Dossier 工作流。 | pending | 2026-07-27T23:58:02+08:00 | DD-UXD-001 | README 与规划文档均明确该原则，不鼓励巨大 PR。 |
| DD-UXD-003 | 定义 5,000 行跨模块变更的演示故事，包括用户痛点、输入、关键步骤、输出和最终证据。 | pending | 2026-07-27T23:58:02+08:00 | DD-UXD-001 | 场景可由公开或合成数据完整复现。 |
| DD-UXD-004 | 重构 README 骨架与当前可证实内容：Logo/定位、small-change-first 原则、场景、演示入口、核心能力、安装、5 分钟 Quick Start、结果、竞品边界、安全摘要、文档、Roadmap、贡献；详细安全边界继续由主题文档承载。 | pending | 2026-07-28T00:23:39+08:00 | DD-DOC-007, DD-UXD-001, DD-UXD-002, DD-UXD-003, DD-UXD-013 | 新读者在首屏能回答“是什么、为何不同、如何开始”，且没有未完成资产的虚假占位或未证实宣称。 |
| DD-UXD-005 | 编写最短安装路径，并按 macOS、Linux、Windows 给出校验和与 attestation 验证方法。 | pending | 2026-07-28T00:14:02+08:00 | DD-REL-002, DD-DOC-005 | 三类系统命令与最新 beta 制品名称及当前工具语法一致；实际四平台 smoke 由 DD-REL-003 验收。 |
| DD-UXD-006 | 编写不依赖付费 Provider 的 5 分钟 synthetic Quick Start，覆盖安装、doctor、准备、计划、手工或 mock 结果、verify 和报告查看。 | pending | 2026-07-28T00:14:02+08:00 | DD-UXD-003, DD-UXD-005, DD-DOC-007 | 全新临时仓库按文档可在 10 分钟内完成一次 review。 |
| DD-UXD-007 | 为 Quick Start 提供可复制的固定 fixture、期望输出摘要和清理步骤。 | pending | 2026-07-27T23:58:02+08:00 | DD-UXD-006 | fixture 可重复运行，输出中的非确定字段已说明。 |
| DD-UXD-008 | 在私有工作区录制 30～60 秒终端 GIF 或视频候选，展示从变更到过期证据失效再到最终报告的关键价值；本任务不发布媒体。 | pending | 2026-07-28T00:19:45+08:00 | DD-UXD-006, DD-CASE-007, DD-CASE-009 | 媒体候选绑定获批公开输入，无私有路径、凭据或不可公开数据，并经人工视觉确认。 |
| DD-UXD-009 | 在私有工作区生成代表性报告截图候选，标注 Snapshot 绑定、Finding 状态、Gate 证据和残余风险；本任务不发布截图。 | pending | 2026-07-28T00:19:45+08:00 | DD-CASE-006, DD-CASE-009 | 截图候选绑定获批公开输入，与当前 CLI 输出一致，并通过隐私检查。 |
| DD-UXD-010 | 编写 DiffDossier 与单 Prompt、Qodo、Reviewdog、Danger 的边界对比，避免无法验证的优越性或性能宣称。 | pending | 2026-07-28T00:14:02+08:00 | DD-UXD-001, DD-CASE-008 | 每项功能对比有可公开来源，效果差异仅引用冻结实验结果和样本限制；没有营销性绝对结论。 |
| DD-UXD-011 | 建立文档入口页或导航，区分新手路径、运维、安全、Provider 开发、Schema/CLI 契约和历史 checkpoint。 | pending | 2026-07-27T23:58:02+08:00 | DD-UXD-004, DD-DOC-004 | 所有非历史文档从 README 至多两跳可达。 |
| DD-UXD-012 | 由一名实际不熟悉架构的新用户或获批代表执行走查；当前实现者、自动化 Agent 或 Provider 不得冒充该角色。记录完成时间、卡点、误解和改进项。 | pending | 2026-07-28T00:29:26+08:00 | DD-UXD-004, DD-UXD-005, DD-UXD-006, DD-UXD-007, DD-UXD-011 | 走查者身份与独立性、同意范围和记录完整，阻塞性问题已修复或进入有 ID 的 TODO。 |
| DD-UXD-013 | 确认或制作可公开使用的 Logo 与最小视觉资产，核对项目名称、第三方素材许可、商标风险、深浅色和可访问性。 | pending | 2026-07-28T00:14:02+08:00 | DD-UXD-001 | 源文件、导出格式、许可和人工视觉确认完整；不使用来源不明素材。 |
| DD-UXD-014 | 将获批演示媒体、报告截图、案例与边界对比集成进 README，并执行链接、渲染、移动端宽度和隐私复核。 | pending | 2026-07-28T00:19:45+08:00 | DD-UXD-004, DD-UXD-010, DD-UXD-011, DD-UXD-015 | README 无断链、无私有数据、无过期截图，最终渲染经人工确认。 |
| DD-UXD-015 | 对 GIF/视频和截图候选分别执行 prepare、许可与隐私扫描，并为每个最终字节候选取得独立、私有、内容绑定的公开批准；若候选属于 `redacted_summary`，另取 `redaction_approval`。 | pending | 2026-07-28T00:19:45+08:00 | DD-UXD-008, DD-UXD-009 | 每个媒体候选的哈希、类别、扫描、适用批准和最终公开字节一致；公开记录不含内部 source ID 或审批人身份，撤销产生 tombstone。 |

### 1.3 已有 Beta Release 的审计与下一版闭环

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-REL-001 | 盘点 beta.1～beta.3 的 tag、commit、Release、Actions run、资产和发布说明，记录每版实际证据与缺口。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-001 | 版本矩阵可追溯到 GitHub URL 和 SHA。 |
| DD-REL-002 | 验证最新 beta 的六平台制品、`SHA256SUMS`、SPDX SBOM、GitHub Attestation、嵌入版本和来源 commit 是否一致。 | pending | 2026-07-27T23:58:02+08:00 | DD-REL-001 | 每个资产均有校验记录；缺失项被明确标记而非推断通过。 |
| DD-REL-003 | 在 Windows amd64、Linux amd64、macOS Intel、macOS ARM 上分别执行下载、校验、安装、version、doctor 和 synthetic smoke。 | pending | 2026-07-28T00:14:02+08:00 | DD-REL-002, DD-UXD-005, DD-UXD-006, DD-UXD-007 | 四个平台的原生记录绑定同一 Release 和资产哈希。 |
| DD-REL-004 | 明确 beta 支持范围、已知限制、配置/输出/Provider Schema 兼容承诺和不支持事项。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-001, DD-DOC-005 | SUPPORT、install、release notes 和兼容矩阵一致。 |
| DD-REL-005 | 补齐撤销、受影响版本标记、升级和回滚说明，明确 Release 不原地修补且不得复用标签。 | pending | 2026-07-27T23:58:02+08:00 | DD-REL-001 | 使用历史 beta 做一次桌面演练，步骤可执行。 |
| DD-REL-006 | 修正所有仍称“没有公开 Release”的现时文档，并保留“没有 stable release”的准确边界。 | pending | 2026-07-27T23:58:02+08:00 | DD-REL-001, DD-DOC-007 | 文档搜索和人工 review 无矛盾。 |
| DD-REL-007 | 为下一 beta 建立逐项发布清单：clean tag、CI、制品、checksums、SBOM、attestation、安装 smoke、已知限制、案例和维护者批准。 | pending | 2026-07-27T23:58:02+08:00 | DD-REL-002, DD-REL-004, DD-REL-005 | 清单可失败关闭，并绑定候选 commit。 |
| DD-REL-008 | 在不创建公开 Release 的隔离候选引用或等价安全环境中演练 Release 工作流，验证失败不会留下半发布或可误认的稳定资产。 | pending | 2026-07-28T00:14:02+08:00 | DD-REL-007 | 演练记录包含故障注入、清理和恢复证据；可重复构建对比仍由 DD-SEC-012 独立验收。 |
| DD-REL-009 | 决定下一 beta 的版本号、范围、候选 SHA、已知限制和是否进入发布审批；本任务不创建 tag 或 Release。 | pending | 2026-07-28T00:14:02+08:00 | DD-REL-007, DD-REL-008 | 版本决策和候选范围可审计，未获得批准时明确停止。 |
| DD-REL-010 | 在全部 P0 退出门禁通过并获得单独明确授权后，创建新的不可复用 tag/Pre-release，并完成发布后回读与 smoke。 | pending | 2026-07-28T00:14:02+08:00 | DD-REL-009, DD-P0-EXIT-001, DD-P0-EXIT-002, DD-P0-EXIT-003, DD-P0-EXIT-004, DD-P0-EXIT-005, DD-P0-EXIT-006 | tag、候选 SHA、资产、checksum、SBOM、attestation、安装 smoke 和 Release 页面一致；失败按撤销流程处理。 |

### 1.4 第一个公开、可复现的真实案例

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-CASE-001 | 制定案例选择标准：公开许可、可固定 commit、足够大、包含跨模块风险、允许再分发、没有个人或私有数据。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-005 | 标准通过安全和许可 review。 |
| DD-CASE-002 | 选择并冻结第一个大型 Go 变更案例，记录上游仓库、许可、base/head SHA、文件数、行数和字节数。 | pending | 2026-07-27T23:58:02+08:00 | DD-CASE-001 | 输入可从公开来源重建，哈希和许可记录完整。 |
| DD-CASE-003 | 定义对照实验协议：同一输入、单 Prompt 基线、DiffDossier 多任务流程、Provider/模型/Pass/Perspective、超时、Token、成本和人工确认规则。 | pending | 2026-07-27T23:58:02+08:00 | DD-CASE-002 | 协议在运行前冻结，避免按结果修改评价标准。 |
| DD-CASE-004 | 生成案例的 Inventory、Snapshot、Contract、Risk 和 Task Plan，并保存可公开的确定性摘要。 | pending | 2026-07-27T23:58:02+08:00 | DD-CASE-002, DD-CASE-003 | 重跑得到相同内容摘要；私有状态未进入仓库。 |
| DD-CASE-005 | 执行单 Prompt 基线和完整 Dossier 流程；任何网络、费用、凭据和 Provider 版本必须针对本次运行取得精确授权与记录，不能等待或复用未来通用协议模板。 | pending | 2026-07-28T00:19:45+08:00 | DD-CASE-003, DD-CASE-004 | 原始结果、逐次授权、版本、时间、成本和失败均被保存，不挑选性丢弃。 |
| DD-CASE-006 | 由人工逐项确认、拒绝或接受 Finding 风险，记录理由、证据位置和最终残余风险。 | pending | 2026-07-27T23:58:02+08:00 | DD-CASE-005 | 没有把 Provider 输出自动提升为已确认问题。 |
| DD-CASE-007 | 修改受绑定输入并演示旧结果自动失效，再完成刷新和重新验证。 | pending | 2026-07-28T00:14:02+08:00 | DD-CASE-004, DD-CASE-005 | 演示清晰证明已有结果在输入变化后成为 stale，且无法继续通过 verify。 |
| DD-CASE-008 | 汇总 Task 数、发现数、确认数、基线漏报、未知项、耗时、Token、成本和残余风险，注明样本限制。 | pending | 2026-07-27T23:58:02+08:00 | DD-CASE-006, DD-CASE-007 | 指标能从保存证据重算，避免只公布正向数字。 |
| DD-CASE-009 | 对拟公开案例执行许可、秘密、路径、身份、日志和衍生数据扫描，并取得私有、内容绑定的 `public_export_approval`；若候选类别为 `redacted_summary`，还必须取得独立的 `redaction_approval`。 | pending | 2026-07-28T00:14:02+08:00 | DD-CASE-008 | 扫描报告、候选哈希、类别、两类适用批准和最终 bundle 哈希一致；撤销路径与 tombstone 已演练。 |
| DD-CASE-010 | 在 `examples/case-studies/` 发布案例说明、重现脚本或命令、固定输入引用、期望摘要和局限性。 | pending | 2026-07-28T00:14:02+08:00 | DD-CASE-009 | 全新环境可按文档重现；公开包不暴露私有 object、内部 source ID 或审批人身份。 |

### 1.5 社区入口与仓库治理

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-GOV-001 | 只读审计当前 GitHub visibility、ruleset、branch protection、required checks、PVR、Actions 权限、Release 和默认分支状态。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-001 | 审计时间、API 结果和证据级别完整记录。 |
| DD-GOV-002 | 新增 Bug、Feature、Documentation 和 Case Study Issue Forms，并禁止提交私有源码、凭据、路径和日志。 | pending | 2026-07-27T23:58:02+08:00 | DD-DOC-002 | 表单语法通过，预览和示例完成。 |
| DD-GOV-003 | 增加 `config.yml`，提供支持、安全披露和讨论入口，关闭不安全的空白敏感报告路径。 | pending | 2026-07-27T23:58:02+08:00 | DD-GOV-002 | GitHub 表单入口符合预期且链接有效。 |
| DD-GOV-004 | 新增 PR Template，要求范围、风险、测试、兼容性、数据分类、DCO 和授权边界。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-005 | 用示例 PR 验证模板能覆盖代码、Schema、工作流和文档变更。 |
| DD-GOV-005 | 新增 CODEOWNERS，覆盖核心、Schema、CI/Release、安全、文档和 Provider 协议；owner 必须来自 DD-DOC-001 的决定。 | pending | 2026-07-27T23:58:02+08:00 | DD-DOC-001 | GitHub 能解析所有 owner，敏感路径无空白覆盖。 |
| DD-GOV-006 | 新增公开 Roadmap，使用阶段、退出标准和非承诺声明，并链接本计划中的稳定范围。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-003 | Roadmap 不泄露内部数据，也不把目标日期表述为保证。 |
| DD-GOV-007 | 新增 MAINTAINERS 或扩展 GOVERNANCE，定义角色、决策、Reviewer、继任、失联和利益冲突规则。 | pending | 2026-07-27T23:58:02+08:00 | DD-DOC-001 | 维护者明确确认，文档不把仓库 owner 等同于独立批准。 |
| DD-GOV-008 | 扩充 beta 最低 CONTRIBUTING：开发环境、当前测试矩阵、DCO、fixture 数据规则、隐私边界和 first issue 流程；尚未稳定的 Schema/Provider/Reporter 政策只链接当前边界，不提前承诺。 | pending | 2026-07-28T00:19:45+08:00 | DD-GOV-004 | 新贡献者可完成一次文档或测试贡献演练。 |
| DD-GOV-009 | 验证并启用 Private Vulnerability Reporting；若不可用，提供经过确认的私密备用渠道。 | pending | 2026-07-27T23:58:02+08:00 | DD-GOV-001 | GitHub 设置证据和 SECURITY 指引一致；该外部写操作有单独授权。 |
| DD-GOV-010 | 在 CI 检查名称稳定并实际成功后，配置 main ruleset：PR、对话解决、线性历史、禁止删除/force push 和 required checks。 | pending | 2026-07-27T23:58:02+08:00 | DD-GOV-001, DD-DOC-009 | ruleset live 查询与预期一致，且不会让单维护者流程死锁。 |
| DD-GOV-011 | 明确“独立人工批准”启用条件；在第二位合格 Reviewer 可用前，不启用会令仓库不可合并的批准要求。 | pending | 2026-07-27T23:58:02+08:00 | DD-GOV-007, DD-GOV-010 | 决策记录含当前限制、触发条件和后续复查时间。 |
| DD-GOV-012 | 定义 beta 支持响应范围、版本支持窗口和每 4～6 周发布目标，不承诺无法履行的 SLA。 | pending | 2026-07-27T23:58:02+08:00 | DD-REL-004, DD-GOV-007 | SUPPORT 与 Roadmap、Release policy 一致。 |

### 1.6 阶段一退出门禁

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-P0-EXIT-001 | 确认所有当前身份、Release、平台和 checkpoint 文档不再互相矛盾。 | pending | 2026-07-28T00:14:02+08:00 | DD-DOC-002, DD-DOC-004, DD-DOC-006, DD-DOC-007, DD-DOC-008, DD-DOC-009, DD-DOC-010, DD-DOC-011, DD-REL-006 | 离线文档漂移检查、外部事实人工复核和完整文档审阅均通过。 |
| DD-P0-EXIT-002 | 确认一名不依赖架构文档的新用户可在 10 分钟内完成 synthetic review。 | pending | 2026-07-27T23:58:02+08:00 | DD-UXD-012 | 计时走查成功，阻塞问题为零。 |
| DD-P0-EXIT-003 | 确认至少一个真实公开案例可复现，结果含对照、人工确认和残余风险。 | pending | 2026-07-27T23:58:02+08:00 | DD-CASE-010 | 独立重现成功，公开批准和隐私扫描有效。 |
| DD-P0-EXIT-004 | 确认最新 beta 的资产、attestation、四个平台安装和已知限制形成闭环。 | pending | 2026-07-28T00:14:02+08:00 | DD-REL-002, DD-REL-003, DD-REL-004, DD-REL-005 | 所有必需证据均存在，非阻塞限制已公开；若仍有关键缺口，本任务保持 pending 并阻止下一版发布。 |
| DD-P0-EXIT-005 | 确认 Issue Form、PR Template、CODEOWNERS、Roadmap、维护者、贡献指引、支持范围、安全入口和基础 main ruleset/required checks 达到 beta 最低社区要求。 | pending | 2026-07-28T00:23:39+08:00 | DD-GOV-002, DD-GOV-003, DD-GOV-004, DD-GOV-005, DD-GOV-006, DD-GOV-007, DD-GOV-008, DD-GOV-009, DD-GOV-010, DD-GOV-011, DD-GOV-012 | 仓库入口和 ruleset live 验证通过，且没有把待启用设置表述为已启用。 |
| DD-P0-EXIT-006 | 确认最终 README 已集成获批媒体、报告、案例和边界对比，并保持新手路径与安全边界准确。 | pending | 2026-07-28T00:14:02+08:00 | DD-UXD-014 | 最终渲染、链接、隐私和事实审阅均通过。 |

## 2. 阶段二：产品可用性与契约加固（P1，目标 2～6 周）

### 2.0 阶段入口

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-P1-ENTRY-001 | 确认阶段一六项退出门禁及下一版 beta 发布闭环均已完成，冻结阶段二起始基线，并记录阶段一未阻塞但需继续观察的风险。 | pending | 2026-07-28T00:36:19+08:00 | DD-P0-EXIT-001, DD-P0-EXIT-002, DD-P0-EXIT-003, DD-P0-EXIT-004, DD-P0-EXIT-005, DD-P0-EXIT-006, DD-REL-010 | 六项退出任务和新 beta 发布任务均为 completed，起始 HEAD、Release、规则和残余风险有时间绑定记录。 |

### 2.1 CLI 引导层

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-CLI-001 | 编写引导层设计，明确 `quickstart`、`next`、`explain`、guided workflow 与现有精确命令的映射及信任边界。 | pending | 2026-07-27T23:58:02+08:00 | DD-UXD-006, DD-CTRL-005 | 设计通过安全、CLI 契约和可用性 review。 |
| DD-CLI-002 | 实现 `diffdossier quickstart`，只创建安全 fixture 或给出步骤，不隐式联网、不执行项目命令、不越过授权。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-001 | 正常、重复、失败和恶意路径测试通过。 |
| DD-CLI-003 | 建立统一工作流状态机查询 API，供 `status`、`next`、`explain` 和 guided 共用，避免提示与真实门禁漂移。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-001 | 状态转换表与现有 workflow 测试一致。 |
| DD-CLI-004 | 实现 `diffdossier next`，仅输出当前允许的下一步、阻塞原因和所需授权，不自动执行下一步。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-003 | 对每种状态有 golden test，禁止跨越 Gate、Provider、fix 和 public export 授权。 |
| DD-CLI-005 | 实现 `diffdossier explain <run-id>`，解释 Snapshot、Task、Finding、失效传播、Gate 和 readiness，默认脱敏。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-003, DD-RISK-007 | 私有路径与对象不会出现在默认输出；JSON 契约有 Schema。 |
| DD-CLI-006 | 实现 `diffdossier review guided`，逐步引导但每个授权点仍要求内容绑定的明确确认。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-004, DD-CLI-005 | E2E 覆盖取消、恢复、stale、Provider 失败、Gate 拒绝和终端状态。 |
| DD-CLI-007 | 为执行计划增加人类可读的差异摘要，展示 argv、cwd、env 名称、网络、写范围和相对上次计划的变化。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-001 | Digest 仍是唯一授权绑定；摘要改变不会弱化精确校验。 |
| DD-CLI-008 | 重构 `--help` 为顶层、常用、精确/高级和危险操作分级展示。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-001 | 终端宽度和跨平台快照测试通过。 |
| DD-CLI-009 | 实现 `diffdossier docs <topic>` 或等价离线主题索引，覆盖安全、Provider、Gate、export、archive、gc 和 Schema。 | pending | 2026-07-27T23:58:02+08:00 | DD-UXD-011 | 主题链接在安装包内或指向版本固定文档。 |
| DD-CLI-010 | 生成并测试 Bash completion。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-008 | 在支持版本 Bash 中完成安装与补全 smoke。 |
| DD-CLI-011 | 生成并测试 Zsh completion。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-008 | 在支持版本 Zsh 中完成安装与补全 smoke。 |
| DD-CLI-012 | 生成并测试 Fish completion。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-008 | 在支持版本 Fish 中完成安装与补全 smoke。 |
| DD-CLI-013 | 生成并测试 PowerShell completion。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-008 | 在支持版本 PowerShell 中完成安装与补全 smoke。 |
| DD-CLI-014 | 发布 Exit Code 手册，并为 usage、validation、stale、authorization、Provider、Gate、integrity 和 internal failure 建立稳定测试。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-001 | 文档与 `internal/cli/exitcodes.go` 及测试一一对应。 |
| DD-CLI-015 | 盘点所有 `--json` 输出，定义稳定字段、可选字段、错误 envelope、版本和兼容政策。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-001, DD-SCH-005 | 每个公开命令有 fixture、Schema 或明确非稳定声明。 |
| DD-CLI-016 | 对引导命令做可用性回归，量化命令数、人工选择数、首次成功时间和误操作率。 | pending | 2026-07-27T23:58:02+08:00 | DD-CLI-002, DD-CLI-004, DD-CLI-005, DD-CLI-006 | 相对当前 Quick Start 有记录的改进，且安全边界测试不退化。 |

### 2.2 Schema 与兼容性

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-SCH-002 | 决定 Draft 2020-12 validator 的实现方式，评估零第三方依赖原则、审计范围、离线构建和维护成本。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-001, DD-CTRL-005 | ADR 记录被选方案、拒绝方案和供应链影响。 |
| DD-SCH-003 | 在测试中使用固定、可审计且离线可运行的 validator 真正验证所有发布 Schema 符合 Draft 2020-12，而不只验证 JSON 可解析。 | pending | 2026-07-28T00:14:02+08:00 | DD-SCH-002 | 故意破坏关键字或引用时测试失败，外部引用无需在普通 CI 临时联网解析。 |
| DD-SCH-004 | 完整定义 Config 的 `review`、`state`、`risk` 及其他嵌套字段、必填项、边界和 `additionalProperties`。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-001, DD-SCH-002 | 当前合法配置全通过；未知、错型、越界和冲突配置均被拒绝。 |
| DD-SCH-005 | 为 output envelope 增加条件约束：`ok` 必须有 `data` 且无 `error`，`error` 必须有 `error` 且无 `data`。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-002 | 正反例和所有 CLI envelope 测试通过。 |
| DD-SCH-006 | 为每个公共 Schema 添加至少一个合法 fixture，并验证生产代码生成的实例符合 Schema。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-003 | fixture 全部在 CI 验证。 |
| DD-SCH-007 | 为安全关键 Schema 添加非法 fixture：缺字段、额外字段、错枚举、坏 digest、越界大小、条件冲突和未知版本。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-003 | 每个非法 fixture 都有明确拒绝原因。 |
| DD-SCH-008 | 建立 Go 结构体与 Schema 字段的一致性测试或生成/比较机制，防止生产者与契约漂移。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-001, DD-SCH-003 | 故意改动一侧时测试可靠失败。 |
| DD-SCH-009 | 以 Review Result 1.0/1.1 为首个对象建立旧写新读、新写旧读、未知字段、未知版本和 digest 稳定性矩阵，并为其他已版本化公共契约逐项声明适用组合。 | pending | 2026-07-28T00:14:02+08:00 | DD-SCH-006, DD-SCH-007 | 每个被声明支持或拒绝的组合均有测试和迁移结论。 |
| DD-SCH-010 | 发布 Schema 升级、废弃、支持窗口、breaking change 和迁移政策。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-009 | GOVERNANCE、CONTRIBUTING、SUPPORT 和 Provider 文档一致。 |
| DD-SCH-011 | 将完整化 Config、Draft 验证、fixture、Go/Schema 一致性和兼容矩阵纳入具有稳定名称的 CI job；是否设为 required check 由 DD-GOV-013 在成功运行后决定。 | pending | 2026-07-28T00:23:39+08:00 | DD-SCH-003, DD-SCH-004, DD-SCH-008, DD-SCH-009 | CI 对故意引入的漂移失败，并在目标分支和 Tier 1 环境稳定运行。 |

### 2.3 Glob、风险引擎与变更拆分建议

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-RISK-001 | 定义 DiffDossier Glob 规范：分隔符、`*`、`?`、`[]`、递归 `**`、根锚定、大小写、无效模式和跨平台路径标准化。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-005 | 规范含规范化示例与反例，跨平台 review 通过。 |
| DD-RISK-002 | 实现独立 Glob matcher，替换风险覆盖中的隐式 `filepath.Match` 语义。 | pending | 2026-07-28T00:14:02+08:00 | DD-RISK-001, DD-RISK-004 | Windows、Unix、特殊字符和深层 `**` conformance 测试通过。 |
| DD-RISK-003 | 为旧配置检测语义变化并提供 warning、doctor 诊断和迁移指导，避免静默改变覆盖范围。 | pending | 2026-07-27T23:58:02+08:00 | DD-RISK-002 | 固定旧配置 fixture 能得到明确兼容或迁移结果。 |
| DD-RISK-004 | 在实现 matcher 前建立 Glob conformance fixture，覆盖 Config、risk、Gate path 和未来 Policy Pack 的规范示例、反例与跨平台期望。 | pending | 2026-07-28T00:14:02+08:00 | DD-RISK-001 | fixture 独立于具体实现通过人工 review，并可被所有路径消费者复用。 |
| DD-RISK-005 | 建立带人工标签的风险样本集，覆盖 Go、Java、JS/TS、配置、SQL、消息契约、迁移、认证、支付和删除操作。 | pending | 2026-07-27T23:58:02+08:00 | DD-CASE-001 | 数据许可清楚，标签过程和不确定项可审计。 |
| DD-RISK-006 | 为每条风险规则统计命中、误报、漏报、未知率和样本量，并设置最低样本门槛。 | pending | 2026-07-27T23:58:02+08:00 | DD-RISK-005 | 报告可重复生成，不以小样本宣称高准确率。 |
| DD-RISK-007 | 定义 `why-risk` 解释链数据模型，记录规则、证据路径、风险提升、置信边界和未知原因。 | pending | 2026-07-27T23:58:02+08:00 | DD-RISK-005, DD-SCH-010 | Schema、Go 类型和示例完成兼容 review。 |
| DD-RISK-008 | 在 CLI/report 中展示 `why-risk`，默认避免泄露绝对路径或私有规则内容。 | pending | 2026-07-27T23:58:02+08:00 | DD-RISK-007 | 文本、JSON、脱敏和空证据测试通过。 |
| DD-RISK-009 | 用测试锁定“项目配置只能提高风险、不能降低内建安全下限”的不变量。 | pending | 2026-07-27T23:58:02+08:00 | DD-RISK-002 | 属性测试和回归 fixture 能阻止风险降级。 |
| DD-RISK-010 | 设计“先拆分”评估：判断变更是否可安全拆分，并输出依赖感知建议；建议不得自动修改 Git 或掩盖必须共同审查的契约。 | pending | 2026-07-27T23:58:02+08:00 | DD-RISK-007, DD-CTRL-005 | 设计含不可拆分判据、循环依赖、迁移和生成代码案例。 |
| DD-RISK-011 | 实现只读拆分建议及有版本的 JSON/Markdown 输出，完整 Dossier 仍可由用户显式选择。 | pending | 2026-07-28T00:14:02+08:00 | DD-RISK-010, DD-SCH-010 | 固定案例的拆分建议稳定、依赖完整且不会丢文件，输出契约有兼容测试。 |

### 2.4 安全验证与供应链

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-SEC-001 | 刷新威胁模型，覆盖不可信 Git 输出、路径、JSON、Schema、Provider、命令、日志、ZIP/TAR、公开导出、CI 和发布链。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-005 | 威胁、控制、残余风险和测试映射完整。 |
| DD-SEC-002 | 定义 Go fuzz 目标、输入上限、超时、语料、崩溃保留和 CI/夜间运行策略。 | pending | 2026-07-27T23:58:02+08:00 | DD-SEC-001 | fuzz 计划有资源预算和失败处理。 |
| DD-SEC-003 | 为路径标准化、特殊文件名和 NUL 分隔 Git 输出增加 fuzz target。 | pending | 2026-07-27T23:58:02+08:00 | DD-SEC-002 | 固定时长无崩溃，发现的问题转为回归语料。 |
| DD-SEC-004 | 为 Config/Schema、结果导入、Provider envelope 和版本兼容增加 fuzz target。 | pending | 2026-07-27T23:58:02+08:00 | DD-SEC-002, DD-SCH-003 | panic、无限资源增长和接受非法状态均为零。 |
| DD-SEC-005 | 为 ZIP/TAR、portable export、archive、manifest 和路径穿越增加 fuzz target。 | pending | 2026-07-27T23:58:02+08:00 | DD-SEC-002 | 路径逃逸、重复条目、超限和畸形 archive 均失败关闭。 |
| DD-SEC-006 | 为日志和公开导出 Redaction 增加 fuzz target 与 privacy canary。 | pending | 2026-07-27T23:58:02+08:00 | DD-SEC-002 | canary 不出现在输出，截断和编码边界有回归测试。 |
| DD-SEC-007 | 建立固定 fuzz corpus 和最小化流程，并保证公开仓库语料只含合成或批准数据。 | pending | 2026-07-27T23:58:02+08:00 | DD-SEC-003, DD-SEC-004, DD-SEC-005, DD-SEC-006 | corpus 许可、来源和回归映射完整。 |
| DD-SEC-008 | 增加最小权限、固定 Action SHA 的 CodeQL 工作流，覆盖 Go；对 GitHub Actions 工作流按实施时官方支持范围选择 CodeQL 或其他官方受支持分析能力，不虚构语言覆盖。 | pending | 2026-07-28T00:19:45+08:00 | DD-GOV-001, DD-SEC-001 | 所选分析范围有官方依据，首次 run 成功，告警有 triage 记录，required check 决策明确。 |
| DD-SEC-009 | 增加固定版本的 `govulncheck`，定义工具更新和零依赖情况下的告警处理策略。 | pending | 2026-07-27T23:58:02+08:00 | DD-SEC-001 | CI 可重复执行，发现项有阻塞等级和例外流程。 |
| DD-SEC-010 | 增加 OpenSSF Scorecard，并审查所需 token、权限、结果发布和误报处理。 | pending | 2026-07-28T00:19:45+08:00 | DD-GOV-001, DD-SEC-001 | workflow 最小权限，结果可见性经维护者批准；是否成为 required check 由 DD-GOV-013 决定。 |
| DD-SEC-011 | 建立测试覆盖率趋势，先记录基线，再为关键包设置防退化门槛而非追求单一总百分比。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-003 | 关键包门槛和趋势报告稳定，无通过删测试提升覆盖率。 |
| DD-SEC-012 | 用两个独立 Builder 对同一 commit 生成制品并比较，记录可重复构建范围和不可避免差异。 | pending | 2026-07-27T23:58:02+08:00 | DD-REL-002 | 归档、manifest、checksum 和 SBOM 对比结果可审计。 |
| DD-SEC-013 | 定义 CodeQL、govulncheck、Scorecard、fuzz、覆盖率和可重复构建差异的 triage、例外、到期和披露流程。 | pending | 2026-07-28T00:19:45+08:00 | DD-SEC-008, DD-SEC-009, DD-SEC-010, DD-SEC-011, DD-SEC-012 | SECURITY/GOVERNANCE 引用该流程，例外不可永久静默。 |

### 2.5 Provider Protocol 与 Conformance Kit

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-PROV-001 | 盘点当前 handshake、command plan、attempt ledger、result import、egress grant 和 adapter 行为，确定公开协议范围。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-001 | 协议对象与实现、Schema、测试一一映射。 |
| DD-PROV-002 | 发布 `PROVIDER_PROTOCOL.md`，描述 argv-only 边界、stdin/stdout、大小、超时、错误、取消、重试、脱敏和信任绑定。 | pending | 2026-07-27T23:58:02+08:00 | DD-PROV-001, DD-SCH-010 | 文档可独立指导第三方实现，不依赖阅读核心源码。 |
| DD-PROV-003 | 定义 Handshake/Review 状态机及失败语义，区分 accepted、rejected、unknown、timeout 和 maintenance failure。 | pending | 2026-07-28T00:14:02+08:00 | DD-PROV-001 | 状态机与 attempt/result 行为测试一致；accepted/rejected 为终态，只有 unknown outcome 可重试，终态后的维护失败不得触发重复派发。 |
| DD-PROV-004 | 发布核心版本、协议版本、Result Schema、Codex adapter 和 Claude adapter 的兼容矩阵。 | pending | 2026-07-27T23:58:02+08:00 | DD-PROV-002, DD-SCH-009 | 矩阵含支持、实验、拒绝和迁移路径。 |
| DD-PROV-005 | 实现最小 Reference Provider，默认本地、无网络、可确定性返回 fixture。 | pending | 2026-07-27T23:58:02+08:00 | DD-PROV-002 | 独立构建并通过协议正常与异常路径。 |
| DD-PROV-006 | 发布 Mock Server/Fixture，覆盖正常结果、坏 Schema、超限输出、超时、取消、部分写和恶意内容。 | pending | 2026-07-27T23:58:02+08:00 | DD-PROV-003 | fixture 可在 Windows、Linux、macOS 重现。 |
| DD-PROV-007 | 建立 Conformance Test Kit，第三方 Provider 可在不修改核心仓库的情况下运行。 | pending | 2026-07-27T23:58:02+08:00 | DD-PROV-005, DD-PROV-006 | Reference Provider 全通过，故意坏实现按预期失败。 |
| DD-PROV-008 | 定义 Provider 认证等级 experimental、beta、stable 及升级、降级、撤销要求。 | pending | 2026-07-27T23:58:02+08:00 | DD-PROV-004, DD-PROV-007 | GOVERNANCE、SUPPORT 和兼容矩阵一致。 |
| DD-PROV-009 | 发布协议升级、废弃、兼容窗口和安全修复政策。 | pending | 2026-07-27T23:58:02+08:00 | DD-PROV-008, DD-SCH-010 | 旧 Provider 失败模式明确且可诊断。 |
| DD-PROV-010 | 使用 Conformance Kit 重验 Codex 与 Claude Code adapters，并固定可重现的版本/模型/Schema 证据。 | pending | 2026-07-27T23:58:02+08:00 | DD-PROV-007, DD-PROV-009 | 两个 adapter 的兼容记录绑定精确二进制与版本。 |
| DD-PROV-011 | 为任何真实 Provider 运行建立逐次授权门禁：目标、数据分类、可执行文件、版本、模型、凭据来源、目的地、Token/费用上限和 egress grant。 | pending | 2026-07-27T23:58:02+08:00 | DD-PROV-002 | 授权模板失败关闭，未授权或超预算测试通过。 |

### 2.6 Reporter 与输出适配层

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-REP-001 | 盘点现有 JSON/Markdown report、portable export 和 public bundle，确定 Reporter 复用范围与缺口。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-001 | 生产者、字段、数据分类和稳定性矩阵完成。 |
| DD-REP-002 | 编写 Reporter 接口 ADR，规定输入证据、Snapshot 绑定、脱敏、离线默认、远端写授权和失败语义。 | pending | 2026-07-27T23:58:02+08:00 | DD-REP-001, DD-SEC-001 | ADR 通过安全和兼容性 review。 |
| DD-REP-003 | 稳定并测试 Markdown Reporter，明确可公开与私有字段。 | pending | 2026-07-27T23:58:02+08:00 | DD-REP-002 | golden、脱敏、stale 和大输出测试通过。 |
| DD-REP-004 | 稳定并测试 JSON Reporter，发布 Schema 和字段兼容政策。 | pending | 2026-07-27T23:58:02+08:00 | DD-REP-002, DD-SCH-010 | Schema、fixture 和 CLI 输出一致。 |
| DD-REP-005 | 实现自包含 HTML Reporter，默认不加载远程脚本、字体或分析服务。 | pending | 2026-07-27T23:58:02+08:00 | DD-REP-002 | 离线打开、内容安全、转义、大小和隐私测试通过。 |
| DD-REP-006 | 实现 SARIF Reporter，定义 Finding 严重度、位置缺失、规则 ID、baseline 和 suppression 映射。 | pending | 2026-07-27T23:58:02+08:00 | DD-REP-002 | SARIF validator 与多个 fixture 通过，未知位置不会伪造精确行号。 |
| DD-REP-007 | 实现 reviewdog-compatible 输出并记录信息损失和安全边界。 | pending | 2026-07-27T23:58:02+08:00 | DD-REP-002 | 固定 fixture 可被 reviewdog 解析，丢失字段有文档。 |
| DD-REP-008 | 为所有 Reporter 增加一致性测试，确保同一 Finding 的 ID、状态、证据和 stale 结论不冲突。 | pending | 2026-07-27T23:58:02+08:00 | DD-REP-003, DD-REP-004, DD-REP-005, DD-REP-006, DD-REP-007 | 跨格式语义对比测试通过。 |
| DD-REP-009 | 明确 GitHub Checks 与 GitLab Code Quality 是可选远端 Reporter，默认不开启且不随本地 `ready` 自动发布。 | pending | 2026-07-27T23:58:02+08:00 | DD-REP-002 | 设计记录权限、幂等、撤销、评论噪声和 fork 安全。 |

### 2.7 性能与规模门槛

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-PERF-001 | 定义公开、固定的规模 fixture：小、中、大、超限，覆盖文件数、行数、字节数、二进制、长路径和多模块。 | pending | 2026-07-27T23:58:02+08:00 | DD-CASE-001 | fixture 来源与生成方法可复现。 |
| DD-PERF-002 | 记录 inventory、snapshot、planning、packet、import、verify、report、export 的时间、内存和磁盘基线。 | pending | 2026-07-27T23:58:02+08:00 | DD-PERF-001 | 结果绑定硬件、OS、Go 版本和 commit。 |
| DD-PERF-003 | 为关键阶段定义 beta 性能门槛、回归阈值和超限时的失败或降级行为。 | pending | 2026-07-27T23:58:02+08:00 | DD-PERF-002, DD-CTRL-003 | 阈值基于数据，超限不会静默截断并标记完成。 |
| DD-PERF-004 | 将稳定、低噪声的固定 fixture benchmark 纳入 CI，重型 benchmark 放入独立周期任务。 | pending | 2026-07-27T23:58:02+08:00 | DD-PERF-003 | CI 波动率和误报率可接受。 |
| DD-PERF-005 | 发布规模限制、预估资源、超限诊断和调优指南。 | pending | 2026-07-27T23:58:02+08:00 | DD-PERF-003 | README/文档不宣称未验证的上限。 |

### 2.8 阶段二治理收口

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-GOV-013 | 在新增 Schema、安全和供应链检查稳定成功后，重新评估并按单独授权更新 required checks；不得仅因工作流文件存在就宣称已受保护。 | pending | 2026-07-28T00:38:11+08:00 | DD-GOV-010, DD-SCH-011, DD-SEC-008, DD-SEC-009, DD-SEC-010 | 每个被要求的 check 均已在目标分支实际成功，ruleset 回读与文档一致。 |
| DD-GOV-014 | 在 Schema、Provider Protocol 与 Reporter 边界稳定后扩充高级贡献指南、兼容性清单和对应 PR 检查项。 | pending | 2026-07-28T00:38:11+08:00 | DD-GOV-008, DD-SCH-010, DD-PROV-009, DD-REP-002 | 第三方贡献者可按文档完成一个 conformance fixture 或 Reporter 扩展示例。 |

### 2.9 阶段二退出门禁

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-P1-EXIT-001 | 在至少 3 个不同公开或获批仓库上完成端到端验证，覆盖 2 个 Provider。 | pending | 2026-07-28T00:23:39+08:00 | DD-P1-ENTRY-001, DD-CLI-006, DD-PROV-010, DD-PROV-011 | 每个运行绑定输入、Provider、人工判断和最终证据。 |
| DD-P1-EXIT-002 | 确认所有公开 Schema 通过 Draft、fixture、Go 一致性和兼容矩阵测试。 | pending | 2026-07-28T00:23:39+08:00 | DD-P1-ENTRY-001, DD-SCH-004, DD-SCH-011 | 稳定命名的 CI job 在最终候选 SHA 上成功；required 状态仅按 live ruleset 证据表述。 |
| DD-P1-EXIT-003 | 确认 guided workflow 将首次命令链显著缩短，且未弱化任何授权边界。 | pending | 2026-07-28T00:23:39+08:00 | DD-P1-ENTRY-001, DD-CLI-016 | 可用性数据与安全 E2E 同时通过。 |
| DD-P1-EXIT-004 | 确认 Glob、风险质量指标、风险安全下限、解释链和拆分建议具有跨平台一致语义。 | pending | 2026-07-28T00:23:39+08:00 | DD-P1-ENTRY-001, DD-RISK-003, DD-RISK-006, DD-RISK-008, DD-RISK-009, DD-RISK-011 | conformance、带标签样本指标与真实案例 review 通过。 |
| DD-P1-EXIT-005 | 确认 fuzz、CodeQL、govulncheck、Scorecard、覆盖率和可重复构建均有当前证据与 triage 机制。 | pending | 2026-07-28T00:23:39+08:00 | DD-P1-ENTRY-001, DD-SEC-007, DD-SEC-008, DD-SEC-009, DD-SEC-010, DD-SEC-011, DD-SEC-012, DD-SEC-013, DD-GOV-013 | 阻塞发现为零；其余未关闭发现均有 owner、严重度、处置和到期时间。 |
| DD-P1-EXIT-006 | 确认 Markdown、JSON、HTML、SARIF 和 reviewdog 输出语义一致，并保持本地离线默认。 | pending | 2026-07-28T00:23:39+08:00 | DD-P1-ENTRY-001, DD-REP-008 | 跨格式测试与隐私测试通过。 |
| DD-P1-EXIT-007 | 确认高级贡献指南与稳定后的 Schema、Provider Protocol、Conformance Kit 和 Reporter 边界一致。 | pending | 2026-07-28T00:23:39+08:00 | DD-P1-ENTRY-001, DD-GOV-014, DD-PROV-010, DD-REP-009 | 第三方扩展示例可按文档完成，实验接口未被误标为 stable。 |
| DD-P1-EXIT-008 | 确认性能基线、回归阈值、CI/周期 benchmark 和规模限制形成可执行门槛。 | pending | 2026-07-28T00:23:39+08:00 | DD-P1-ENTRY-001, DD-PERF-003, DD-PERF-004, DD-PERF-005 | 最终候选无未解释性能回归，超限行为失败关闭。 |
| DD-P1-EXIT-009 | 确认帮助分级、计划差异、离线主题、四类 Shell completion、Exit Code 和 `--json` 契约均已验收。 | pending | 2026-07-28T00:23:39+08:00 | DD-P1-ENTRY-001, DD-CLI-007, DD-CLI-008, DD-CLI-009, DD-CLI-010, DD-CLI-011, DD-CLI-012, DD-CLI-013, DD-CLI-014, DD-CLI-015 | 文档、跨平台 smoke、快照和契约测试全部通过。 |

## 3. 阶段三：生态与真实验证（P2，目标 6～12 周）

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-P2-ENTRY-001 | 确认阶段二九项退出门禁均已完成，冻结阶段三起始基线，并记录兼容、安全、性能和外部验证的残余风险。 | pending | 2026-07-28T00:23:39+08:00 | DD-P1-EXIT-001, DD-P1-EXIT-002, DD-P1-EXIT-003, DD-P1-EXIT-004, DD-P1-EXIT-005, DD-P1-EXIT-006, DD-P1-EXIT-007, DD-P1-EXIT-008, DD-P1-EXIT-009 | 九项退出任务均为 completed，起始 HEAD、Release、矩阵和残余风险有时间绑定记录。 |
| DD-ECO-001 | 将 Codex 与 Claude Code adapters 纳入独立版本、模型、协议、Schema、平台和认证等级兼容矩阵；任何 live smoke 的真实 Provider 调用仍需逐次授权。 | pending | 2026-07-28T00:34:22+08:00 | DD-PROV-010 | 每个支持组合有当前 conformance 和获批 live smoke 证据。 |
| DD-ECO-002 | 定义第三方 Provider 提交流程、测试证据、安全 review、命名、维护和撤销规则。 | pending | 2026-07-28T00:19:45+08:00 | DD-PROV-008, DD-GOV-014 | 贡献模板可用，实验 Provider 不被误标为稳定。 |
| DD-ECO-003 | 设计并发布 Go 风险策略包，保留内建风险下限和解释链。 | pending | 2026-07-27T23:58:02+08:00 | DD-RISK-005, DD-RISK-009 | 在带标签 Go 样本上有命中与噪声报告。 |
| DD-ECO-004 | 设计并发布 Java/Spring 风险策略包，覆盖多模块、事务、配置、消息和数据库迁移。 | pending | 2026-07-27T23:58:02+08:00 | DD-RISK-005, DD-RISK-009 | 在带标签 Java 样本上有命中与噪声报告。 |
| DD-ECO-005 | 设计并发布 JavaScript/TypeScript 风险策略包，覆盖包边界、API、构建、依赖和前后端契约。 | pending | 2026-07-27T23:58:02+08:00 | DD-RISK-005, DD-RISK-009 | 在带标签 JS/TS 样本上有命中与噪声报告。 |
| DD-ECO-006 | 定义 Policy Pack 格式与权限：只能影响分析和提高风险，不能扩大命令、网络、写入或发布权限。 | pending | 2026-07-27T23:58:02+08:00 | DD-RISK-009, DD-CTRL-005 | 恶意 pack 无法降低风险或新增执行能力。 |
| DD-ECO-007 | 实现可选 GitHub Checks Reporter，使用最小权限、明确授权、幂等更新和撤销语义；实现与本地 fixture 测试不依赖 main ruleset 已启用。 | pending | 2026-07-28T00:19:45+08:00 | DD-REP-009 | fork、重跑、stale、权限不足和撤销测试通过；真实写入仍需逐次授权。 |
| DD-ECO-008 | 实现可选 GitLab Code Quality Reporter，保持与 GitHub Reporter 相同的证据和授权边界。 | pending | 2026-07-27T23:58:02+08:00 | DD-REP-009 | fixture 与真实沙箱验证通过。 |
| DD-ECO-009 | 对 Java/Spring 案例最终候选独立执行许可/隐私扫描、内容绑定公开批准、发布和撤销验证；不得复用第一案例记录。 | pending | 2026-07-28T00:26:49+08:00 | DD-ECO-004, DD-ECO-013, DD-ECO-014, DD-ECO-015 | 案例可独立重现，最终候选有自己的哈希、扫描、适用批准、发布回读和 tombstone 演练。 |
| DD-ECO-010 | 对配置/SQL/迁移/消息契约案例最终候选独立执行许可/隐私扫描、内容绑定公开批准、发布和撤销验证。 | pending | 2026-07-28T00:26:49+08:00 | DD-ECO-004, DD-ECO-016, DD-ECO-017, DD-ECO-018 | 案例覆盖跨契约残余风险，可重现，最终候选有自己的扫描、适用批准、发布回读和撤销证据。 |
| DD-ECO-011 | 对 AI Agent 生成变更案例最终候选独立执行许可/隐私扫描、内容绑定公开批准、发布和撤销验证。 | pending | 2026-07-28T00:26:49+08:00 | DD-ECO-019, DD-ECO-020, DD-ECO-021 | 输入与生成元数据可公开，结论不泛化到所有 Agent，最终候选有自己的扫描、适用批准、发布回读和撤销证据。 |
| DD-DIST-001 | 设计 Homebrew 分发，确定 tap、formula、checksum、签名/attestation、更新和撤销责任。 | pending | 2026-07-27T23:58:02+08:00 | DD-P0-EXIT-004, DD-REL-007 | 设计和维护成本经维护者批准。 |
| DD-DIST-002 | 发布并实测 Homebrew beta 安装、升级、降级和卸载。 | pending | 2026-07-27T23:58:02+08:00 | DD-DIST-001 | Intel/ARM macOS clean smoke 绑定确切版本。 |
| DD-DIST-003 | 设计 Scoop 与 Winget 分发，明确 manifest、审核、签名、更新延迟和撤销流程。 | pending | 2026-07-27T23:58:02+08:00 | DD-P0-EXIT-004, DD-REL-007 | 选择 Scoop、Winget 或两者的决定有依据。 |
| DD-DIST-004 | 发布并实测获批的 Windows 包管理渠道。 | pending | 2026-07-27T23:58:02+08:00 | DD-DIST-003 | Windows clean install、upgrade、rollback、uninstall 通过。 |
| DD-DIST-005 | 确保所有包管理渠道只引用已发布 Release 的相同哈希，不在渠道内重新构建不可追溯二进制。 | pending | 2026-07-27T23:58:02+08:00 | DD-DIST-002, DD-DIST-004 | 渠道资产与 Release checksum/attestation 对账成功。 |
| DD-ECO-012 | 建立每 4～6 周 beta 候选、回归、发布、反馈和支持窗口节奏。 | pending | 2026-07-27T23:58:02+08:00 | DD-GOV-012, DD-REL-007 | 连续两个 beta 周期按流程完成或记录有依据的偏差。 |
| DD-ECO-013 | 选择并冻结大型 Java/Spring 多模块案例的公开输入、许可、base/head SHA 和实验协议。 | pending | 2026-07-28T00:26:49+08:00 | DD-P2-ENTRY-001, DD-CASE-010, DD-ECO-004 | 输入可重建，协议在运行前冻结，许可与数据分类清楚。 |
| DD-ECO-014 | 按冻结协议执行 Java/Spring 案例的单 Prompt 基线和完整 Dossier 流程，逐次记录 Provider 授权、版本、成本、时间和失败。 | pending | 2026-07-28T00:26:49+08:00 | DD-ECO-013 | 原始结果完整保存，无选择性丢弃；任何真实 Provider 调用均有本次授权。 |
| DD-ECO-015 | 对 Java/Spring 案例逐项完成人工 Finding 决策、基线对比、指标汇总、残余风险和 stale evidence 演示。 | pending | 2026-07-28T00:26:49+08:00 | DD-ECO-014 | 人工 ledger、可重算指标、失效演示和公开候选摘要完整。 |
| DD-ECO-016 | 选择并冻结配置、SQL、数据库迁移和消息契约混合案例的公开输入、许可、base/head SHA 和实验协议。 | pending | 2026-07-28T00:26:49+08:00 | DD-P2-ENTRY-001, DD-CASE-010, DD-ECO-004 | 输入可重建，跨契约范围和协议在运行前冻结，许可与数据分类清楚。 |
| DD-ECO-017 | 按冻结协议执行混合案例的单 Prompt 基线和完整 Dossier 流程，逐次记录 Provider 授权、版本、成本、时间和失败。 | pending | 2026-07-28T00:26:49+08:00 | DD-ECO-016 | 原始结果完整保存，无选择性丢弃；任何真实 Provider 调用均有本次授权。 |
| DD-ECO-018 | 对混合案例逐项完成人工 Finding 决策、基线对比、指标汇总、残余风险和 stale evidence 演示。 | pending | 2026-07-28T00:26:49+08:00 | DD-ECO-017 | 人工 ledger、可重算指标、跨契约残余风险、失效演示和公开候选摘要完整。 |
| DD-ECO-019 | 选择并冻结 AI Agent 生成变更案例的公开输入、生成工具/模型/提示或配置、许可、base/head SHA 和实验协议。 | pending | 2026-07-28T00:26:49+08:00 | DD-P2-ENTRY-001, DD-CASE-010 | 生成来源与可公开性可追溯，协议在运行前冻结。 |
| DD-ECO-020 | 按冻结协议执行 AI Agent 案例的单 Prompt 基线和完整 Dossier 流程，逐次记录 Provider 授权、版本、成本、时间和失败。 | pending | 2026-07-28T00:26:49+08:00 | DD-ECO-019 | 原始结果完整保存，无选择性丢弃；任何真实 Provider 调用均有本次授权。 |
| DD-ECO-021 | 对 AI Agent 案例逐项完成人工 Finding 决策、基线对比、指标汇总、残余风险和 stale evidence 演示。 | pending | 2026-07-28T00:26:49+08:00 | DD-ECO-020 | 人工 ledger、可重算指标、失效演示和公开候选摘要完整，结论不外推到未测试 Agent。 |
| DD-P2-EXIT-001 | 确认至少 4 类真实大型变更案例、3 类语言/框架风险包、Policy Pack 权限边界、2 个 Provider，以及 GitHub 和 GitLab 两类可选远端 Reporter 有可复现证据。 | pending | 2026-07-28T00:23:39+08:00 | DD-P2-ENTRY-001, DD-ECO-001, DD-ECO-002, DD-ECO-003, DD-ECO-004, DD-ECO-005, DD-ECO-006, DD-ECO-007, DD-ECO-008, DD-ECO-009, DD-ECO-010, DD-ECO-011 | 生态矩阵完整，实验项未被标为 stable。 |
| DD-P2-EXIT-002 | 确认 Homebrew 与获批 Windows 渠道的资产可追溯、可撤销并通过 clean install。 | pending | 2026-07-28T00:23:39+08:00 | DD-P2-ENTRY-001, DD-DIST-005 | 渠道和 Release 哈希一致，无悬空旧版本。 |
| DD-P2-EXIT-003 | 确认 beta 候选、回归、发布、反馈和支持节奏已经过连续两个周期验证。 | pending | 2026-07-28T00:23:39+08:00 | DD-P2-ENTRY-001, DD-ECO-012 | 两个周期均有候选 SHA、门禁、发布或取消决定、反馈和偏差记录。 |

## 4. 阶段四：稳定版准备（P3，目标 3～6 个月）

| ID | 内容 | 状态 | updated_at | 依赖 | 完成证据 |
|---|---|---|---|---|---|
| DD-P3-ENTRY-001 | 确认阶段三三项退出门禁均已完成，冻结稳定版准备起始基线，并记录生态、分发和发布节奏的残余风险。 | pending | 2026-07-28T00:23:39+08:00 | DD-P2-EXIT-001, DD-P2-EXIT-002, DD-P2-EXIT-003 | 三项退出任务均为 completed，起始 HEAD、最新 beta、渠道状态和残余风险有时间绑定记录。 |
| DD-V1-001 | 定义 v1.0 准入指标：真实用户数、持续使用周期、成功仓库数、升级样本、严重缺陷、兼容性和支持能力。 | pending | 2026-07-27T23:58:02+08:00 | DD-CTRL-003, DD-P2-EXIT-001 | 指标有数值、证据来源和不可妥协项。 |
| DD-V1-002 | 建立外部用户反馈闭环，区分产品问题、文档问题、Provider 问题、风险误报和安全问题；联系用户、录制会话或收集非公开反馈需另行授权并取得适用同意。 | pending | 2026-07-28T00:34:22+08:00 | DD-GOV-002, DD-P2-EXIT-001 | 至少两个 beta 周期能从反馈追踪到决定和发布，反馈的用途、留存、访问和删除边界可审计。 |
| DD-V1-003 | 实现配置、Schema、Provider Protocol 的兼容性检查和迁移工具，默认只读预览后再显式写入。 | pending | 2026-07-27T23:58:02+08:00 | DD-SCH-010, DD-PROV-009 | 多版本 fixture 的 dry-run、迁移、回滚和幂等测试通过。 |
| DD-V1-004 | 冻结 v1 的 CLI、Config、JSON、Schema、Provider 和 Reporter 稳定政策及废弃周期。 | pending | 2026-07-27T23:58:02+08:00 | DD-V1-003 | 政策与实际兼容矩阵、SUPPORT、SemVer 和 Changelog 一致。 |
| DD-V1-005 | 培养至少一位独立人工维护者或 Reviewer，并验证其能够审查安全、兼容和 Release 变更；联系候选人、发出邀请或授予权限必须另行获得精确授权。 | pending | 2026-07-28T00:29:26+08:00 | DD-GOV-007, DD-GOV-011 | 有实际独立 review 证据后才调整批准 ruleset；人员、角色、权限范围和撤销方式均有记录。 |
| DD-V1-006 | 对 crash recovery、锁、journal、archive、shared blob、GC、revocation 和升级执行长期压力与故障注入。 | pending | 2026-07-27T23:58:02+08:00 | DD-SEC-007, DD-PERF-003 | 长期运行无未解释数据丢失；已知限制被文档化。 |
| DD-V1-007 | 执行跨版本升级、降级、归档恢复和回滚矩阵，覆盖所有仍支持的 beta 数据。 | pending | 2026-07-27T23:58:02+08:00 | DD-V1-003, DD-V1-006 | 每条路径有 fixture、结果和不可逆边界。 |
| DD-V1-008 | 在计划内功能、外部反馈、迁移、长期可靠性和分发渠道完成后，冻结独立安全审计候选 SHA、范围、审计方、数据边界、费用、保密、披露和交付标准，并在单独授权后委托审计。 | pending | 2026-07-28T00:23:39+08:00 | DD-P3-ENTRY-001, DD-V1-001, DD-V1-002, DD-V1-003, DD-V1-004, DD-V1-005, DD-V1-006, DD-V1-007 | 审计范围与合同或委托记录可审计，未授权时不向外部提供材料；后续候选变更会使审计 seal 失效。 |
| DD-V1-009 | 在审计问题处置后复核 ADR-0002、ADR-0003 及完整威胁模型；由复核触发的实现或文档修复必须在独立复测前进入候选。 | pending | 2026-07-28T00:26:49+08:00 | DD-V1-014 | 所有边界与实现一致，漂移项已在候选中修复或由有权角色明确接受。 |
| DD-V1-010 | 将 DD-V1-015 已独立复测的同一 SHA 在本地或私有空间打包为 v1.0 release candidate，不创建公开 tag/Release，并执行不改变候选内容的完整 Schema、Provider、Reporter、平台、安装、升级、回滚、安全和案例验证；任何内容变更都必须返回 DD-V1-014、DD-V1-009 和 DD-V1-015。 | pending | 2026-07-28T00:34:22+08:00 | DD-P3-ENTRY-001, DD-V1-001, DD-V1-002, DD-V1-004, DD-V1-005, DD-V1-007, DD-V1-015 | 独立复测 SHA、打包输入 SHA 和候选 SHA 一致，全量门禁成功，未解决阻塞缺陷为零。 |
| DD-V1-011 | 由维护者和独立 Reviewer 对 v1.0 候选、已知风险、支持承诺和撤销计划作最终人工决定。 | pending | 2026-07-27T23:58:02+08:00 | DD-V1-010 | 内容绑定的批准记录完整，拒绝不会触发发布。 |
| DD-V1-012 | 在分别获得公开 Release、公告及每个包管理渠道的精确授权后发布 DD-V1-010 的不变候选，并验证 tag、资产、checksum、SBOM、attestation、包管理渠道、安装和公告一致。 | pending | 2026-07-28T00:34:22+08:00 | DD-V1-011 | 发布输入与获批候选 SHA 一致，Release live 证据和发布后 smoke 完整；任何失败按撤销流程处理。 |
| DD-V1-013 | 接收独立审计报告并逐项分级、去重、确认可复现性，记录 owner、处置、披露和截止时间。 | pending | 2026-07-28T00:14:02+08:00 | DD-V1-008 | 全部审计项进入受控 ledger，原始报告访问边界明确。 |
| DD-V1-014 | 修复已确认审计问题，或由有权角色对不可修复项作内容绑定的风险接受；不得由实现者自动替代批准人。 | pending | 2026-07-28T00:14:02+08:00 | DD-V1-013 | 阻塞问题为零，每个非阻塞项有修复证据或未过期风险接受。 |
| DD-V1-015 | 由独立审计方或等价独立 Reviewer 对包含审计修复和威胁模型复核变更的候选重新复测并关闭审计范围。 | pending | 2026-07-28T00:26:49+08:00 | DD-V1-014, DD-V1-009 | 独立复测绑定最终候选 SHA，未关闭阻塞项为零。 |

## 5. 原 Top 10 与任务映射

| 原建议 | 对应主任务 |
|---|---|
| 统一身份信息 | DD-DOC-001～DD-DOC-002、DD-DOC-011 |
| 处理过期 checkpoint 与平台状态 | DD-DOC-003～DD-DOC-007 |
| 重写 README、Quick Start、截图 | DD-UXD-001～DD-UXD-015 |
| 真实大型变更案例 | DD-CASE-001～DD-CASE-010 |
| 完成 beta 发布闭环 | DD-REL-001～DD-REL-010；现有 beta.1～beta.3 不重复创建 |
| 完善 Schema 与兼容性测试 | DD-SCH-001～DD-SCH-011 |
| 引导命令 | DD-CLI-001～DD-CLI-016 |
| Fuzz、CodeQL、govulncheck、Scorecard | DD-SEC-001～DD-SEC-013 |
| Provider Protocol、Conformance、Reporter | DD-PROV-001～DD-PROV-011、DD-REP-001～DD-REP-009 |
| 社区与贡献入口 | DD-GOV-001～DD-GOV-014 |

## 6. 外部操作与人工门禁

以下动作即使对应 TODO 已进入 `in_progress`，也必须在执行当轮获得单独授权：

| 触发动作 | 相关任务 | 必须确认的最小范围 |
|---|---|---|
| 修改 GitHub 身份展示、ruleset、required checks、PVR 或仓库权限 | DD-DOC-011、DD-GOV-009～DD-GOV-011、DD-GOV-013 | 仓库、设置项、预期值、回滚方式 |
| 推送、启用或首次外部运行安全工作流，或公开其结果 | DD-SEC-008～DD-SEC-010 | 分支、精确工作流、权限、触发范围、结果可见性和回滚方式；仅在本地编辑与测试工作流无需该外部门禁 |
| 调用真实或付费 Provider | DD-CASE-005、DD-PROV-010、DD-P1-EXIT-001、DD-ECO-001、DD-ECO-014、DD-ECO-017、DD-ECO-020 | 数据分类、目的地、可执行文件、模型、凭据、费用上限、egress grant；仅编写协议或授权模板不构成调用 |
| 公开案例、截图、报告或衍生包 | DD-UXD-015、DD-CASE-009～DD-CASE-010、DD-ECO-009～DD-ECO-011 | 候选哈希、许可、隐私扫描、公开类别、`public_export_approval`、适用的 `redaction_approval`、撤销计划 |
| 向 GitHub Checks 或 GitLab 写入 Reporter 结果 | DD-ECO-007～DD-ECO-008、DD-P2-EXIT-001 | 目标仓库/项目、候选哈希、权限、幂等键、写入范围、撤销或清理方式 |
| 创建 tag、Release 或公开公告 | DD-REL-010、DD-ECO-012、DD-V1-012 | 精确版本、候选 SHA、资产、门禁、已知限制、回滚/撤销 |
| 发布 Homebrew、Scoop 或 Winget | DD-DIST-002、DD-DIST-004、DD-V1-012 | 渠道、版本、资产哈希、维护责任、撤销方式；v1.0 的 Release 授权不自动覆盖渠道发布 |
| 委托、扩展或重新执行外部安全审计 | DD-V1-008、DD-V1-015 | 审计方、候选 SHA、数据范围、费用、保密、报告披露、复测范围和修复流程 |
| 由人工角色确认身份、定位、媒体、治理、风险标签、Finding、结果可见性、分发设计、风险接受或 Release 决定 | DD-DOC-001、DD-UXD-001、DD-UXD-008、DD-UXD-013～DD-UXD-014、DD-REL-009、DD-CASE-006、DD-GOV-007、DD-GOV-011、DD-RISK-005、DD-SEC-010、DD-P1-EXIT-001、DD-DIST-001、DD-ECO-015、DD-ECO-018、DD-ECO-021、DD-V1-009、DD-V1-014、DD-V1-011 | 角色资格、精确候选或证据、决定范围、时间和有效期；实现者、Agent 或 Provider 不得替代要求的人工决定 |
| 执行新用户首次成功验收 | DD-UXD-012、DD-P0-EXIT-002 | 实际不熟悉架构的走查者或获批代表、测试范围、无私有数据保证，以及录制或留存观察的同意；当前 Agent 不得冒充走查者 |
| 联系外部用户、录制会话或收集非公开反馈 | DD-V1-001～DD-V1-002 | 精确对象或招募范围、联系内容、数据字段、用途、同意、访问、留存和删除方式 |
| 联系、邀请维护者或授予仓库权限 | DD-V1-005 | 精确人员、联系内容、角色、最小权限、期限和撤销方式 |
| 暂存、提交、推送、创建 PR 或合并本计划产生的变更 | 全部可能产生仓库变更的任务 | 当前精确 diff、分支、提交范围、远端目标和是否创建 PR/合并；本地实现或测试通过不构成交付授权 |

## 7. 建议的后续多轮工作包顺序

1. `计划控制与身份决定包`：DD-CTRL-001～DD-CTRL-005、DD-DOC-001、DD-GOV-001；DD-CTRL-002 在第一次真实状态迁移后收口，外部写操作仍暂停。
2. `Release/平台事实包`：DD-REL-001～DD-REL-002、DD-DOC-005、DD-SCH-001；先取得后续文档所需事实，不创建新 Release。
3. `文档真相包`：DD-DOC-002～DD-DOC-011、DD-REL-004～DD-REL-006；DD-DOC-011 在外部写门禁处暂停。
4. `首次成功包`：DD-UXD-001～DD-UXD-007、DD-UXD-011～DD-UXD-013。
5. `真实案例包`：DD-CASE-001～DD-CASE-010；Provider 和公开动作在门禁处暂停。
6. `README 收口包`：DD-UXD-008～DD-UXD-010、DD-UXD-014～DD-UXD-015、DD-P0-EXIT-002～DD-P0-EXIT-003、DD-P0-EXIT-006。
7. `社区入口包`：DD-GOV-002～DD-GOV-012、DD-P0-EXIT-005；先做本地文件，再单独申请 GitHub 设置授权。
8. `当前 Beta 审计与 P0 收口包`：DD-REL-003、DD-REL-007～DD-REL-009、DD-P0-EXIT-001、DD-P0-EXIT-004；DD-REL-010 只在全部 P0 门禁完成且单独授权后执行。
9. `新 Beta 发布与阶段二入口包`：六项 P0 退出全部 completed 后，在精确授权下执行 DD-REL-010；只有 DD-REL-010 也 completed 后才完成 DD-P1-ENTRY-001，冻结起始基线后再启动任何阶段二实施。
10. `Schema 加固包`：DD-SCH-002～DD-SCH-011。
11. `Glob/风险包`：DD-RISK-001、DD-RISK-004、DD-RISK-002～DD-RISK-003、DD-RISK-005～DD-RISK-011。
12. `CLI 引导包`：DD-CLI-001～DD-CLI-016。
13. `安全验证包`：DD-SEC-001～DD-SEC-013、DD-GOV-013。
14. `Provider/Reporter/高级贡献包`：DD-PROV-001～DD-PROV-011、DD-REP-001～DD-REP-009、DD-GOV-014。
15. `性能与阶段二退出包`：DD-PERF-001～DD-PERF-005、DD-P1-EXIT-001～DD-P1-EXIT-009；九项退出均 completed 后才完成 DD-P2-ENTRY-001。
16. `阶段三生态、案例与分发包`：DD-ECO-001～DD-ECO-021、DD-DIST-001～DD-DIST-005；真实 Provider、远端 Reporter、公开和渠道发布在各自人工门禁处暂停。
17. `阶段三退出与稳定版入口包`：DD-P2-EXIT-001～DD-P2-EXIT-003 全部 completed 后才完成 DD-P3-ENTRY-001。
18. `稳定版准备包`：按依赖执行 DD-V1-001～DD-V1-015；独立审计、人工风险接受、最终批准和 v1.0 发布不得合并为同一授权。

## 8. 执行日志

每个造成计划定义、任务状态或验收证据变化的轮次在此追加一行，且不得改写历史记录。纯只读且无新发现的复审不得仅为记录 clean 结论而修改本文件；其结论应在会话或独立审查制品中绑定完整文件 SHA，避免追加日志后立即使 clean seal 失效。

| 时间 | 任务 ID | 状态变化 | 证据摘要 | 剩余风险或下一步 |
|---|---|---|---|---|
| 2026-07-27T23:58:02+08:00 | plan | created | 基于全面评审，并按本地 main、公开 beta.1～beta.3、当前文档/Schema/CLI 状态校正顺序。 | 所有 TODO 初始为 pending；尚未执行任何产品化任务。 |
| 2026-07-28T00:14:02+08:00 | plan-review-1 | repaired | 修复基线漂移、重复/混合任务、漏失依赖、P0 退出门禁、Release 决策与发布分离、公开批准类型、远端 Reporter 门禁、Schema CI 与 ruleset 边界、Glob 顺序及独立安全审计拆分。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:19:45+08:00 | plan-review-2 | repaired | 修复媒体候选批准复用、P0 被 P1 贡献政策反向阻塞、案例依赖未来 Provider 模板、控制任务遗漏、CLI/风险包顺序、平台安全能力表述、P1/P2 退出覆盖以及安全审计候选冻结时机。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:23:39+08:00 | plan-review-3 | repaired | 修复阶段退出可绕过 Config Schema、风险指标/下限、第三方 Provider 流程和 GitLab Reporter 的缺口；补齐基础 ruleset 门禁、阶段入口继承依赖和外部反馈完成后再冻结审计候选。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:26:49+08:00 | plan-review-4 | repaired | 修复计划状态留证与提交授权混淆、v1 审计复测后仍可能改候选的问题，并将三个阶段三案例分别拆成输入冻结、受控执行、人工确认和独立公开门禁。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:29:26+08:00 | plan-review-5 | repaired | 区分本地安全工作流实现与推送/首次外部运行，移除纯模板任务的 Provider 调用门禁，并补齐 Git 交付、新用户验收、人工角色决定和维护者招募授权边界。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:32:28+08:00 | plan-review-6 | repaired | 修复建议工作包把阶段二入口排在阶段二实施之后的门禁冲突，并将阶段二入口、阶段三入口、阶段四入口及后续生态和稳定版工作按实际依赖展开。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:34:22+08:00 | plan-review-7 | repaired | 锁定独立复测 SHA 与 v1 候选打包输入，补齐 v1 包管理渠道、阶段三 live Provider、外部用户反馈和审计复测的精确授权边界，并扩充关键人工角色门禁。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:36:19+08:00 | plan-review-8 | repaired | 消除 P0 发布支持声明依赖阶段二 Schema 任务所形成的隐式入口环：将 Schema 清单前移为阶段零基线，并要求下一版 beta 完成后才能进入阶段二。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:37:05+08:00 | plan-review-9 | repaired | 明确阶段入口继承规则排除入口任务本身，避免字面解释形成自依赖。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:38:11+08:00 | plan-review-10 | repaired | 将依赖阶段二实现的 required-checks 与高级贡献治理任务移入阶段二，使其明确继承阶段二入口，消除可被提前置为 in_progress 的阶段边界缺口。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:40:22+08:00 | plan-review-11 | repaired | 补齐定位、治理、Scorecard 结果可见性、P1 真实仓库验证和包管理分发设计等明确要求维护者或人工判断的角色门禁。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:41:10+08:00 | plan-review-12 | repaired | 将威胁模型复核中由有权角色接受漂移的决定纳入人工门禁，避免实现者或 Agent 自行接受边界偏差。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T00:43:36+08:00 | plan-review-13 | repaired | 消除 clean review 必须写回被审文件而立即改变 SHA 的日志悖论；无变更复审改为在外部记录中绑定完整文件 SHA。 | 对新快照重新执行完整复审；本行不代表 clean seal。 |
| 2026-07-28T11:06:32+08:00 | DD-CTRL-001 | pending → in_progress | 新增 `docs/plans/2026-07-28-execution-baseline-checklist.md`，其中包含八项可重复检查、完成规则和本轮 live Git/GitHub 初始基线记录（HEAD `aa611755554711dd44fab388f488fd2867ed093e`、三个 beta Pre-release、当前 CI 成功）。 | 下一工作包须完整使用该清单；完成记录后才可将 DD-CTRL-001 标为 completed。 |
| 2026-07-28T11:14:07+08:00 | DD-CTRL-001 | in_progress → completed | `DD-CTRL-002` 工作包完整复跑 `docs/plans/2026-07-28-execution-baseline-checklist.md` 的八项检查；当前 HEAD/远端 main 均为 `aa611755554711dd44fab388f488fd2867ed093e`，Release、CI 与 ruleset 均以本轮只读查询记录。 | DD-CTRL-001 完成；后续每个工作包仍须复跑清单。未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:14:07+08:00 | DD-CTRL-002 | pending → in_progress | 新增 `docs/plans/2026-07-28-task-state-evidence-convention.md`，规定任务表、执行日志、可复现证据、基线清单和交付边界须同轮记录。 | 复核约定、计划状态和 diff；满足验收后再完成 DD-CTRL-002。未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:15:12+08:00 | DD-CTRL-002 | in_progress → completed | 已复核状态/证据约定包含任务 ID、旧/新状态、时间、证据、下一步/风险与交付边界；本轮 `DD-CTRL-001` 和 `DD-CTRL-002` 迁移均按该约定记录，并通过 `git diff --check` 与记录字段断言。 | DD-CTRL-002 完成。下一项可选择独立的 DD-CTRL-003 或依赖已解除的 DD-CTRL-004；未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:17:10+08:00 | DD-CTRL-003 | pending → in_progress | 新增 `docs/plans/2026-07-28-product-metrics.md`，定义八项指标、采集口径、零观测起始基线、目标、角色负责人、样本下限与数据边界。 | 复核指标完整性和计划状态后收口；未执行 Git 交付、Provider 调用或外部写操作。 |
| 2026-07-28T11:18:27+08:00 | DD-CTRL-003 | in_progress → completed | 已复核八项指标均具定义、采集方式、起始基线、目标值和角色负责人，并通过 `git diff --check`、暂存区 diff 检查及 M-01～M-08 字段断言。 | DD-CTRL-003 完成；首份合格账本产生时须替换各项 `N=0 / 不可计算` 基线。未执行 Git 交付、Provider 调用或外部写操作。 |
| 2026-07-28T11:20:22+08:00 | DD-CTRL-004 | pending → in_progress | 新增 `docs/plans/2026-07-28-analysis-gap-matrix.md`，将“当前基线”八条结论逐项分类为需 live 验证、部分完成或仍缺失，并附本轮证据边界与计划 ID 映射。 | 复核八条覆盖、分类、映射与链接后收口；未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:21:29+08:00 | DD-CTRL-004 | in_progress → completed | 已复核矩阵覆盖当前基线 A-01～A-08，包含四类分类标签、每项本轮证据边界与计划 ID 映射，并通过 `git diff --check`、暂存区 diff 检查和覆盖/字段断言。 | DD-CTRL-004 完成；下一项未阻塞控制任务为 DD-CTRL-005。未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:23:11+08:00 | DD-CTRL-005 | pending → in_progress | 新增 `docs/adr/template.md`，并在 `docs/adr/README.md` 与 `CONTRIBUTING.md` 写入高风险设计必须先有 ADR/等价记录的规则；模板覆盖上下文、方案、替代项、后果/验证与逐项外部门禁。 | 复核规则覆盖、模板字段与 CONTRIBUTING 链接后收口；未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:24:11+08:00 | DD-CTRL-005 | in_progress → completed | 已复核 ADR 规则覆盖公共契约、依赖、网络、命令执行、发布、安全和数据公开七类触发条件；模板具备必需字段，CONTRIBUTING 已链接规则，并通过 `git diff --check`、暂存区 diff 检查与字段断言。 | DD-CTRL-005 完成。下一工作包可开始已解除依赖的 DD-DOC-001（需维护者人工身份决定）或阶段零 DD-SCH-001 的调研前置。未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:26:53+08:00 | DD-DOC-001 | pending → in_progress | 新增 `docs/plans/2026-07-28-identity-decision-request.md`，记录 GitHub owner 字段、仓库内既有身份文案及其非证明边界，并提供维护者内容绑定确认格式。 | 等待有权维护者确认账户关系、职责与文档表述范围；在此之前不改写身份文档。未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:29:55+08:00 | DD-DOC-001 | in_progress → completed | 维护者明确确认 `lliangcol`、`liuliang1` 与 Liu Liang 为同一人；结合本轮 GitHub owner 字段、GOVERNANCE、NOTICE 和 DCO 历史例外，已记录 repository owner、maintainer、版权、DCO 签署及敏感操作批准职责映射。 | DD-DOC-001 完成；DD-DOC-002 可统一仓库文案，但本决定不授权 GitHub 设置、Release 或其他外部写操作。 |
| 2026-07-28T11:34:11+08:00 | DD-DOC-002 | pending → in_progress | 新增 `docs/governance/identity-and-roles.md` 作为当前规范，并更新 GOVERNANCE、SECURITY、SUPPORT、CONTRIBUTING 与 Release 文档的身份/路由表达；NOTICE 与 DCO 历史记录保留已正确的版权和签署内容。 | 执行全仓身份搜索与 Markdown/diff review，确认无未解释冲突后收口；未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:35:46+08:00 | DD-DOC-002 | in_progress → completed | 已复核 GOVERNANCE、SECURITY、SUPPORT、CONTRIBUTING 和 Release 文档均链接当前身份规范；NOTICE 的 Liu Liang 版权表述、DCO/Release/历史 checkpoint 中的 `liuliang1` 记录均与已确认角色映射一致且保留历史语境。全仓 Markdown 身份搜索、链接字段断言、`git diff --check` 与暂存区 diff 检查通过。 | DD-DOC-002 完成；下一项未阻塞任务为 DD-DOC-003。未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:39:44+08:00 | DD-DOC-003 | pending → in_progress | 新增 `docs/checkpoints/README.md`，定义 Status、Captured-at、Source-commit、Superseded-by、Current-state notice 的顺序、取值、约束与完整示例，并规定现有 checkpoint 留待 DD-DOC-004 分类和补齐。 | 复核字段、状态语义、历史提示和示例后收口；外部 main/CI/ruleset 查询超时，仅按 unknown 记录。未执行 Git 交付或外部写操作。 |
| 2026-07-28T11:40:39+08:00 | DD-DOC-003 | in_progress → completed | 已复核五项元数据字段及其顺序、三种状态语义、历史 checkpoint 固定提示、Source-commit 约束和完整示例；通过 `git diff --check`、暂存区 diff 检查与字段/顺序断言。 | DD-DOC-003 完成；下一项 DD-DOC-004 将逐个分类并补齐现有 checkpoint 元数据。外部 main/CI/ruleset 仍为本轮 unknown，未执行 Git 交付或外部写操作。 |
