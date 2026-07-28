# 执行前基线检查清单

本清单落实 `DD-CTRL-001`。每个产品化工作包在开始实现、修改任务状态
或记录验收证据前，均应以一次新的执行记录完整运行本清单。它将本地事实、
远端只读事实和未验证项分开；本地跟踪引用不能代替远端当前状态。

本清单不授权暂存、提交、推送、创建 PR、修改 GitHub 设置、创建 Release
或其他外部写操作。若工作包需要其中任一动作，必须取得该动作所需的单独授权。

## 执行步骤

按以下顺序执行；任一关键检查失败或无法取得证据时，停止进入实现，记录
`blocked` 或 `unknown`，并明确需要的下一步，而不是沿用旧快照。

| # | 检查项 | 最低命令或来源 | 记录内容 |
|---|---|---|---|
| 1 | 任务选择与依赖 | 本计划的任务表和阶段入口规则 | 任务 ID、当前状态、全部依赖状态、是否允许开始。 |
| 2 | 仓库规则 | 搜索适用的 `AGENTS.md`、`CONTRIBUTING.md`、`GOVERNANCE.md`、工作流和任务范围内文档 | 适用规则、额外授权或验证要求。 |
| 3 | 工作树、分支和 HEAD | `git status --short --branch`；`git branch --show-current`；`git rev-parse HEAD` | 已提交、暂存、未暂存、未跟踪变更分别列出；分支和完整 SHA。 |
| 4 | 远端差异 | `git remote -v`；`git rev-list --left-right --count HEAD...origin/<branch>`；必要时只读查询远端分支 SHA | 上游、相对本地跟踪引用的 ahead/behind，以及远端 SHA 的来源和查询时间。未经刷新时标明本地跟踪引用可能过期。 |
| 5 | 已有 Release | 只读 `gh release list --repo <owner/repo>` 或等价 GitHub 页面/API | Release/tag、类型、时间和查询时间；资产、哈希和 attestation 不因列表存在而视为已验证。 |
| 6 | 相关 CI/Release 工作流 | 任务范围内 `.github/workflows/*.yml`；只读 `gh run list --repo <owner/repo>` | 相关工作流、最近 run URL、结论、绑定 SHA；旧 run 只作为历史证据。 |
| 7 | 仓库规则和外部门禁 | GitHub ruleset 的只读查询（若任务相关）及本计划第 6 节 | 可见 ruleset、所需人工决定和未获得的外部写授权。 |
| 8 | 工作包边界 | 任务内容、完成证据和当前 diff | 本轮允许修改的文件/行为、预期验证、显式排除项。 |

## 记录格式与完成规则

每次执行在本文件的“执行记录”追加一节，包含时间、执行者、工作包、八项
结果、命令或 URL、完整 SHA，以及 `ready`、`blocked` 或 `unknown` 结论。
不得把不可访问的远端、失败的命令或过期 run 记作通过。

`DD-CTRL-001` 只有在一次后续工作包完整引用并执行上述八项后才可标为
`completed`。该后续记录必须链接到本清单，并在产品化 TODO 的执行日志中
记录状态迁移和证据；本轮仅建立清单并完成它的初始基线记录，因此任务保持
`in_progress`。

## 执行记录

### 2026-07-28T11:06:32+08:00 — DD-CTRL-001 初始基线

| # | 结果 |
|---|---|
| 1 | `DD-CTRL-001` 原为 `pending`，无依赖，属于阶段零，允许开始；完成条件仍要求后续工作包完整使用本清单。 |
| 2 | 未发现 `AGENTS.md`。已读取本计划、`README.md`、`docs/adr/README.md` 和 CI/Release 工作流；本计划禁止未经授权的 Git 与外部写操作。 |
| 3 | 工作树无 `git status --porcelain=v1` 输出；分支为 `main`；HEAD 为 `aa611755554711dd44fab388f488fd2867ed093e`。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`；相对本地 `origin/main` 的 ahead/behind 为 `0/0`。GitHub REST 只读查询的远端 `main` SHA 同为 `aa611755554711dd44fab388f488fd2867ed093e`，查询于本记录时间附近完成。 |
| 5 | GitHub CLI 只读查询到三个 Pre-release：`v0.1.0-beta.3`（2026-07-27T12:39:48Z）、`v0.1.0-beta.2`（2026-07-27T01:02:57Z）和 `v0.1.0-beta.1`（2026-07-27T00:01:14Z）。本记录未核验资产、哈希或 attestation。 |
| 6 | `.github/workflows/ci.yml` 定义 `CI`；`.github/workflows/release.yml` 定义 tag 触发的 `Release`。最新关联 run 为 [CI #30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079)，`success`，绑定当前 HEAD；最近 Release run 为 [Release #30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344)，`success`，绑定 `3c46e62740143b62293f1abf526a1e159084e522`。 |
| 7 | GitHub REST 只读查询显示一个 active 的仓库 ruleset：`Protect main`（ID `19772012`，target `branch`）。本轮无 GitHub 设置、Release、Provider 或 Git 交付授权。 |
| 8 | 本轮仅新增本清单并将计划中的 `DD-CTRL-001` 改为 `in_progress`，再做 Markdown 与 diff 复核；不修改产品实现，不暂存、提交或推送。 |

结论：`ready`（仅可执行 DD-CTRL-001 的本地文档实现）。下一工作包开始时必须从
步骤 1 重新完整执行本清单；完成该使用记录后，才能将 `DD-CTRL-001` 改为
`completed`。

### 2026-07-28T11:14:07+08:00 — DD-CTRL-002 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-CTRL-002`，其唯一依赖 `DD-CTRL-001` 在本次完整清单使用后满足完成证据；阶段零任务可开始。 |
| 2 | 未发现 `AGENTS.md`。已复核 `CONTRIBUTING.md`、`GOVERNANCE.md`、本计划与两个工作流；DCO、焦点范围、文档化设计及外部写操作的明确审核要求均适用。 |
| 3 | 当前分支为 `feature/productization-control-baseline`，HEAD 为 `aa611755554711dd44fab388f488fd2867ed093e`。两份 DD-CTRL 文档为已暂存变更；无未暂存或未跟踪变更。暂存不是提交、推送或批准。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`；相对本地 `origin/main` 的 ahead/behind 为 `0/0`。GitHub REST 只读查询的远端 `main` SHA 同为 `aa611755554711dd44fab388f488fd2867ed093e`，查询于本记录时间附近完成。 |
| 5 | GitHub CLI 只读查询仍列出三个 Pre-release：`v0.1.0-beta.3`、`v0.1.0-beta.2` 和 `v0.1.0-beta.1`。本工作包未核验资产、哈希或 attestation。 |
| 6 | 相关工作流仍为 `CI` 与 `Release`。最新关联 run 为 [CI #30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079)，`success`，绑定当前 HEAD；最近 Release run 为 [Release #30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344)，`success`，绑定 `3c46e62740143b62293f1abf526a1e159084e522`。 |
| 7 | GitHub REST 只读查询仍显示 active 的 `Protect main` ruleset（ID `19772012`，target `branch`）。本轮没有 GitHub 设置、Release、Provider 或 Git 交付授权。 |
| 8 | 本轮只新增状态/证据记录约定，完成 `DD-CTRL-001` 并开始 `DD-CTRL-002`，随后复核 Markdown 与 diff；明确排除产品实现、`git add`、commit、push 和外部写入。 |

结论：`ready`（仅限 DD-CTRL-002 的本地文档实现）。该记录完整使用了本清单，
因此满足 `DD-CTRL-001` 的最后一项完成证据。

### 2026-07-28T11:17:10+08:00 — DD-CTRL-003 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为无依赖的 `DD-CTRL-003`，允许开始；任务验收要求定义、采集方式、基线、目标值和负责人。 |
| 2 | 未发现 `AGENTS.md`。`CONTRIBUTING.md` 的焦点范围、DCO 与不扩张边界，以及 `GOVERNANCE.md` 的维护者审核与发布绑定要求均适用。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `aa611755554711dd44fab388f488fd2867ed093e`。此前 DD-CTRL 文件有暂存和未暂存混合变更；无提交、推送或其他 Git 交付。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`；相对本地 `origin/main` 的 ahead/behind 为 `0/0`。GitHub REST 只读查询的远端 `main` SHA 同为当前 HEAD。 |
| 5 | GitHub CLI 只读查询列出三个 Pre-release：`v0.1.0-beta.3`、`v0.1.0-beta.2` 和 `v0.1.0-beta.1`；未把列表当作安装或产品指标证据。 |
| 6 | 工作流仍为 `CI` 与 `Release`。最新 CI [#30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079) 成功且绑定当前 HEAD；Release [#30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344) 成功但绑定历史 beta.3 SHA。 |
| 7 | GitHub REST 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`）。本轮未获任何外部写、Provider 调用或 Git 交付授权。 |
| 8 | 本轮只定义指标、零观测基线、目标和责任角色，并更新 DD-CTRL-003 状态；不修改 CLI、Schema、Release 或外部系统。 |

结论：`ready`（仅限 DD-CTRL-003 的本地文档实现）。仓库检索未发现合格历史指标
账本，因此基线必须保留为 `N=0 / 不可计算`，直到对应任务产生可审计样本。

### 2026-07-28T11:20:22+08:00 — DD-CTRL-004 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-CTRL-004`，其依赖 `DD-CTRL-001` 已完成；验收要求为带证据链接且映射计划 ID 的差距矩阵。 |
| 2 | 未发现 `AGENTS.md`。已复核 `CONTRIBUTING.md`、`GOVERNANCE.md`、本计划、CI/Release 工作流；文档化设计、焦点范围、DCO 与外部写操作门禁适用。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `aa611755554711dd44fab388f488fd2867ed093e`；DD-CTRL 文档存在暂存/未暂存混合变更和未跟踪新增文档，没有提交、推送或其他 Git 交付。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`；相对本地 `origin/main` 为 `0/0`。GitHub REST 只读查询的远端 `main` SHA 同为当前 HEAD。 |
| 5 | GitHub CLI 只读查询确认 `v0.1.0-beta.1`、`.2`、`.3` 三个 Pre-release；未验证资产、checksum、SBOM、attestation 或安装。 |
| 6 | 相关工作流为 `CI` 与 `Release`。当前 HEAD 的 [CI #30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079) 为 `success`；最近 Release [#30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344) 为 `success`，绑定历史 beta.3 SHA。 |
| 7 | GitHub REST 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`）；本轮无 GitHub 设置、Release、Provider 或 Git 交付授权。 |
| 8 | 本轮仅建立对八条“当前基线”结论的分类矩阵，更新 DD-CTRL-004 状态并复核文档；不修改产品行为、Schema、工作流或外部状态。 |

结论：`ready`（仅限 DD-CTRL-004 的本地文档实现）。矩阵保留 A-01 的可漂移性、
A-02 的未验证资产边界，并将其余未关闭结论映射到原子任务。

### 2026-07-28T11:23:11+08:00 — DD-CTRL-005 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为无依赖的 `DD-CTRL-005`；验收要求为 CONTRIBUTING 或规划文档引用决策规则，且模板可用。 |
| 2 | 未发现 `AGENTS.md`。已复核 `CONTRIBUTING.md`、`GOVERNANCE.md`、ADR 目录、本计划和工作流；现有 ADR 格式与外部操作门禁适用。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `aa611755554711dd44fab388f488fd2867ed093e`；已有 DD-CTRL 文档为暂存/未暂存混合变更及未跟踪新增文档，没有提交或推送。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`；相对本地 `origin/main` 为 `0/0`。GitHub REST 只读查询的远端 `main` SHA 同为当前 HEAD。 |
| 5 | GitHub CLI 只读查询仍列出三个 beta Pre-release；本工作包不核验资产或触发 Release。 |
| 6 | 相关工作流仍为 `CI` 与 `Release`；当前 HEAD 的 [CI #30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079) 为 `success`，最近 Release [#30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344) 绑定历史 beta.3 SHA。 |
| 7 | GitHub REST 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`）；本轮无 GitHub 设置、Release、Provider、网络执行或 Git 交付授权。 |
| 8 | 本轮仅新增 ADR 模板、决策记录规则和 CONTRIBUTING 引用，并更新 DD-CTRL-005 状态；不改变产品行为或执行外部操作。 |

结论：`ready`（仅限 DD-CTRL-005 的本地文档实现）。规则将 ADR 与外部授权分离，
并要求高风险设计在实现前留下可审计决策。

### 2026-07-28T11:26:53+08:00 — DD-DOC-001 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-DOC-001`，依赖 `DD-CTRL-001` 已完成；验收要求是维护者的明确人工决定，用户名相似性不得代替。 |
| 2 | 未发现 `AGENTS.md`。已复核 GOVERNANCE、NOTICE、SECURITY、SUPPORT、CONTRIBUTING、Release 文档、DCO 历史记录及本计划；身份与批准边界只能由有权维护者确认。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `aa611755554711dd44fab388f488fd2867ed093e`；工作树含此前文档的暂存/未暂存混合变更及未跟踪文档，没有提交或推送。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`；相对本地 `origin/main` 为 `0/0`。GitHub REST 只读查询的远端 `main` SHA 同为当前 HEAD。 |
| 5 | GitHub CLI 只读查询仍列出三个 beta Pre-release；本任务不创建或修改 Release。 |
| 6 | 相关工作流仍为 `CI` 与 `Release`；当前 HEAD 的 [CI #30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079) 为 `success`，最近 Release [#30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344) 绑定历史 beta.3 SHA。 |
| 7 | GitHub REST 只读查询显示公开 owner 字段为 `lliangcol`（`User`）和 active 的 `Protect main` ruleset（ID `19772012`）；该字段不证明与 `liuliang1` 或 Liu Liang 的自然人关系。本轮无外部写或 Git 交付授权。 |
| 8 | 本轮仅记录身份决定前事实和最小确认格式，将 DD-DOC-001 标为 `in_progress`；不改写身份文档、不改变 GitHub 设置或外部状态。 |

结论：`blocked`。继续完成 DD-DOC-001 需要有权维护者提供内容绑定的人工作决定；
在此之前，旧文档和公开账户字段均不能作为替代批准。

### 2026-07-28T11:29:55+08:00 — DD-DOC-001 决定收口

| # | 结果 |
|---|---|
| 1 | `DD-DOC-001` 依赖已完成；维护者已明确确认 `lliangcol`、`liuliang1` 与 Liu Liang 为同一人，满足人工决定要求。 |
| 2 | 身份表述的证据范围复核了 GOVERNANCE、NOTICE、DCO 历史例外、Release 文档和身份决定请求；本轮不执行 DD-DOC-002 的文案统一。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `aa611755554711dd44fab388f488fd2867ed093e`；工作树含计划文档变更，无提交、推送或其他 Git 交付。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`，相对本地 `origin/main` 为 `0/0`；本轮 GitHub REST 的远端 `main` SHA 同为当前 HEAD。 |
| 5 | GitHub CLI 只读查询仍列出三个 beta Pre-release；本任务不修改 Release。 |
| 6 | 当前 HEAD 的 [CI #30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079) 为 `success`；最近 Release [#30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344) 绑定历史 beta.3 SHA。 |
| 7 | GitHub REST 公开 owner 字段仍为 `lliangcol`（`User`），ruleset 为 active 的 `Protect main`（ID `19772012`）；该只读事实与维护者关系确认共同构成职责映射，但不授权外部写。 |
| 8 | 本轮只记录确认决定、完成 DD-DOC-001 并复核；明确排除身份文档统一、GitHub 设置、权限、Release 与公开数据操作。 |

结论：`ready`（DD-DOC-001 已具完整人工决定）。下一项 DD-DOC-002 可使用本决定
统一身份文案，但仍不得把它扩大为任何外部操作授权。

### 2026-07-28T11:34:11+08:00 — DD-DOC-002 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-DOC-002`，依赖 `DD-DOC-001` 已完成；验收要求统一指定身份文档，并完成全仓身份搜索/文档 review。 |
| 2 | 未发现 `AGENTS.md`。已复核身份决定、GOVERNANCE、NOTICE、SECURITY、SUPPORT、CONTRIBUTING、Release 文档、DCO 历史例外与本计划；历史记录须保留其捕获时语境。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `aa611755554711dd44fab388f488fd2867ed093e`；工作树含此前计划和文档变更，无提交、推送或其他 Git 交付。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`，相对本地 `origin/main` 为 `0/0`；GitHub REST 的远端 `main` SHA 同为当前 HEAD。 |
| 5 | GitHub CLI 只读查询仍列出三个 beta Pre-release；本任务不创建或修改 Release。 |
| 6 | 当前 HEAD 的 [CI #30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079) 为 `success`；最近 Release [#30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344) 绑定历史 beta.3 SHA。 |
| 7 | GitHub REST owner 字段为 `lliangcol`（`User`），ruleset 为 active 的 `Protect main`（ID `19772012`）；本轮无 GitHub 设置、Release、Provider 或 Git 交付授权。 |
| 8 | 本轮仅新增当前身份规范并统一 GOVERNANCE、SECURITY、SUPPORT、CONTRIBUTING 和 Release 文档；NOTICE/DCO 历史记录保留已正确的版权与签署事实，不改写历史决定。 |

结论：`ready`（仅限 DD-DOC-002 的本地文档实现）。后续搜索将区分当前规范、
正确的历史表述和不允许保留的角色冲突。

### 2026-07-28T11:39:44+08:00 — DD-DOC-003 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-DOC-003`，依赖 `DD-CTRL-005` 已完成；验收要求为包含五个字段和不可作为当前状态提示的规范与示例。 |
| 2 | 未发现 `AGENTS.md`。已盘点 `docs/checkpoints/` 的 7 个文件及现有 ADR/文档规则；当前 checkpoint 未采用统一元数据块，历史表述不得在本任务中重写为当前事实。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `aa611755554711dd44fab388f488fd2867ed093e`；工作树含此前文档变更，没有提交、推送或其他 Git 交付。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`，相对本地 `origin/main` 为 `0/0`。本轮 GitHub `main` SHA 查询超时，故远端 main 当前 SHA 为 `unknown`。 |
| 5 | GitHub CLI 只读查询确认三个 beta Pre-release 仍列出；本任务不验证资产或修改 Release。 |
| 6 | GitHub Actions run 与 ruleset 查询在本轮超时，均为 `unknown`；不将前轮结果当作本轮 live 证据。 |
| 7 | 本轮无 GitHub 设置、Release、Provider、网络执行或 Git 交付授权；外部查询失败不被记作通过。 |
| 8 | 本轮只新增 checkpoint 元数据规范和示例，并更新 DD-DOC-003 状态；不批量修改现有 checkpoint（留给 DD-DOC-004）。 |

结论：`ready`（仅限 DD-DOC-003 的本地文档实现）。外部 main/CI/ruleset 事实为
`unknown`，但不影响本地规范的定义；后续工作包必须重新查询。

### 2026-07-28T12:32:32+08:00 — DD-DOC-004 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-DOC-004`，唯一依赖 `DD-DOC-003` 已完成；验收要求为逐个分类全部 checkpoint、补齐元数据，且不把历史内容当作当前事实。允许开始。 |
| 2 | 未发现 `AGENTS.md`。已复核 CONTRIBUTING、GOVERNANCE、ADR 规则、本计划、状态/证据约定、checkpoint 规范与 CI/Release 工作流；保持范围聚焦，历史记录不得改写为当前事实，外部写和 Git 交付仍需明确授权。 |
| 3 | 工作树干净（无已暂存、未暂存或未跟踪文件）；分支为 `feature/productization-control-baseline`，HEAD 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 4 | 上游为 `origin/feature/productization-control-baseline`；相对本地跟踪引用 ahead/behind 为 `0/0`（跟踪引用本身可能过期）。本轮 GitHub API 只读查询远端同名分支 SHA 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`，查询时间为本记录时间。 |
| 5 | 本轮 GitHub CLI 只读查询列出 `v0.1.0-beta.3`、`v0.1.0-beta.2`、`v0.1.0-beta.1` 三个 Pre-release；未由列表推断资产、哈希或 attestation 已验证。 |
| 6 | 相关工作流为 `CI` 与 `Release`。本轮可见最新 CI #30286989079（success，main，历史 SHA）和 Release #30266531344（success，`v0.1.0-beta.3`，历史 SHA）；均非当前分支 HEAD 的验证。 |
| 7 | 本轮 GitHub API 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`，target `branch`）。本任务无 GitHub 设置、Release、Provider、公开数据或 Git 交付授权。 |
| 8 | 本轮只允许修改 7 个现有 checkpoint 的元数据、计划状态/执行日志及本清单记录；验证元数据字段、历史提示、来源 SHA 与 Markdown/diff。明确排除产品实现、工作流、外部状态、`git add`、commit 和 push。 |

结论：`ready`（仅限 DD-DOC-004 的本地文档分类和元数据补齐）。

### 2026-07-28T12:42:01+08:00 — DD-SCH-001 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-SCH-001`，依赖 `DD-CTRL-004` 已完成，且属于阶段零公共契约基线；验收要求为每项 Schema 具备 owner、兼容范围和测试入口。允许开始。 |
| 2 | 未发现 `AGENTS.md`。已复核 CONTRIBUTING、GOVERNANCE、ADR 规则、计划、状态/证据约定和 CI/Release 工作流；本轮仅盘点既有契约，不改变 public contract，因此无需新增 ADR。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。暂存区和未跟踪区为空；未暂存区有 9 个 DD-DOC-004 预期文档修改，未发生 Git 交付。 |
| 4 | 上游为 `origin/feature/productization-control-baseline`，相对本地跟踪引用 ahead/behind 为 `0/0`。本轮 GitHub API 只读查询的远端同名分支 SHA 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 5 | 本轮 GitHub CLI 只读查询列出 `v0.1.0-beta.3`、`v0.1.0-beta.2`、`v0.1.0-beta.1` 三个 Pre-release；不将其视为 Schema 或安装验证。 |
| 6 | 相关工作流为 `CI` 与 `Release`。本轮可见最新 CI #30286989079 与 Release #30266531344 均为历史运行，不作为当前分支 Schema 验证。 |
| 7 | 本轮 GitHub API 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`，target `branch`）。没有 GitHub 设置、Release、Provider、公开数据或 Git 交付授权。 |
| 8 | 本轮范围是新增契约清单及任务/基线记录；检查 `schemas/*.schema.json`、`pkg/schema`、生产/读取路径和测试入口。明确排除 Schema、Go 类型、CLI、Provider、工作流和外部状态变更。 |

结论：`ready`（仅限 DD-SCH-001 的本地契约盘点）。

### 2026-07-28T12:49:22+08:00 — DD-DOC-005 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-DOC-005`，依赖 `DD-CTRL-001` 已完成；验收要求是可回溯到 run、runner、版本、commit 的平台证据矩阵。允许开始。 |
| 2 | 未发现 `AGENTS.md`。已复核 CONTRIBUTING、GOVERNANCE、ADR 规则、本计划、状态/证据约定、现有平台文档和 CI/Release 工作流；本轮仅记录既有证据，不改变平台承诺或工作流，故无需 ADR。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。暂存区为空；未暂存区有 9 个既有文档修改，另有 DD-SCH-001 清单未跟踪；未发生 Git 交付。 |
| 4 | 上游为 `origin/feature/productization-control-baseline`，相对本地跟踪引用 ahead/behind 为 `0/0`。本轮 GitHub API 只读查询的远端同名分支 SHA 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 5 | 本轮 GitHub CLI 只读查询列出三个 Pre-release；最新 `v0.1.0-beta.3` 发布于 2026-07-27T12:39:48Z，包含六平台归档、`SHA256SUMS`、SBOM、provenance 和 release manifest。资产存在不等于本轮下载/哈希/attestation 核验。 |
| 6 | 相关工作流为 `CI` 与 `Release`。本轮取得 CI #30286989079（success，`aa611755554711dd44fab388f488fd2867ed093e`）和 Release #30266531344（success，`3c46e62740143b62293f1abf526a1e159084e522`）的 job/step 详情；二者均为历史 SHA，不作为当前分支验证。 |
| 7 | 本轮 GitHub API 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`，target `branch`）。没有 GitHub 设置、Release、Provider、公开数据或 Git 交付授权。 |
| 8 | 本轮只允许新增平台证据矩阵及更新任务/基线记录；明确分开支持 Tier、原生运行、race、cross-build、安装 smoke、最低版本与未验证语义。排除当前平台文案修正、发布、下载制品、安装或外部写。 |

结论：`ready`（仅限 DD-DOC-005 的本地证据矩阵）。

### 2026-07-28T12:58:03+08:00 — DD-DOC-006 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-DOC-006`，依赖 DD-DOC-005 已完成；验收要求为平台主题文档与当前矩阵一致，且不把 cross-build 提升为 native 证据。允许开始。 |
| 2 | 未发现 `AGENTS.md`。已复核 CONTRIBUTING、GOVERNANCE、ADR 规则、本计划、状态/证据约定、平台矩阵和 CI/Release 工作流；只修正文案，不改变平台承诺、工作流或契约。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。暂存区为空；未暂存区有 9 个既有文档修改，另有 2 个既有未跟踪文档；未发生 Git 交付。 |
| 4 | 上游为 `origin/feature/productization-control-baseline`，相对本地跟踪引用 ahead/behind 为 `0/0`。本轮 GitHub API 只读查询的远端同名分支 SHA 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 5 | 本轮 GitHub CLI 只读查询仍列出三个 Pre-release；未将其作为安装或当前分支验证。 |
| 6 | 相关工作流为 `CI` 与 `Release`。最新可见 CI #30286989079 与 Release #30266531344 为历史 SHA 的成功 run；其细节已在 DD-DOC-005 矩阵中区分。 |
| 7 | 本轮 GitHub API 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`，target `branch`）。没有 GitHub 设置、Release、Provider、公开数据或 Git 交付授权。 |
| 8 | 本轮只更新 `docs/platform-compatibility.md`、任务/基线记录并按风险复核 Markdown 与差异；排除任何代码、CI、Release、安装或外部写操作。 |

结论：`ready`（仅限 DD-DOC-006 的本地文档修正）。

### 2026-07-28T12:59:44+08:00 — DD-DOC-007 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-DOC-007`，依赖 DD-CTRL-004 与 DD-DOC-005 均完成；验收要求是覆盖指定文档、区分已有 beta/未有 stable 与历史状态的对账矩阵。允许开始。 |
| 2 | 未发现 `AGENTS.md`。已复核 CONTRIBUTING、GOVERNANCE、ADR 规则、本计划、状态/证据约定、Release 流程、平台矩阵和六份目标文档；本轮仅盘点/提出修正文案，不改变发布、支持或安全契约。 |
| 3 | 分支为 `feature/productization-control-baseline`，HEAD 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。暂存区为空；未暂存区有 10 个既有文档修改，另有 2 个既有未跟踪文档；未发生 Git 交付。 |
| 4 | 上游为 `origin/feature/productization-control-baseline`，相对本地跟踪引用 ahead/behind 为 `0/0`。本轮 GitHub API 只读查询的远端同名分支 SHA 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 5 | 本轮 GitHub CLI 只读查询列出三个 Pre-release：`v0.1.0-beta.1`、`.2`、`.3`；最新 beta.3 发布于 2026-07-27T12:39:48Z。没有 stable release 的结论仅限这次查询时点。 |
| 6 | 相关工作流为 `CI` 与 `Release`。最新可见 CI #30286989079 与 Release #30266531344 为历史 SHA 的成功 run，不能替代当前分支验证。 |
| 7 | 本轮 GitHub API 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`，target `branch`）。没有 GitHub 设置、Release、Provider、公开数据或 Git 交付授权。 |
| 8 | 本轮只新增发布状态对账矩阵及任务/基线记录；覆盖 README、SUPPORT、install、release-process、SECURITY、CHANGELOG，逐项标注事实、历史语境、来源和修正建议。排除直接文案修正、Release 或外部写操作。 |

结论：`ready`（仅限 DD-DOC-007 的本地对账矩阵）。

### 2026-07-28T13:02:04+08:00 — DD-DOC-008 工作包

| # | 结果 |
|---|---|
| 1 | DD-DOC-008 依赖 DD-CTRL-005 已完成，允许开始。 |
| 2 | 已复核 CONTRIBUTING、GOVERNANCE、ADR 规则；仅定义文档术语，不改变契约。 |
| 3 | 分支 `feature/productization-control-baseline`，HEAD `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`；工作树含此前未提交文档变更。 |
| 4 | 相对 `origin/feature/productization-control-baseline` ahead/behind 为 `0/0`；本轮 GitHub API 只读核对远端同 SHA。 |
| 5 | 只读查询仍列出三个 beta Pre-release；未当作发布批准。 |
| 6 | 可见 CI #30286989079 与 Release #30266531344 均为历史 SHA。 |
| 7 | active `Protect main` ruleset（ID `19772012`）；无外部写授权。 |
| 8 | 范围限于术语表、README、平台矩阵及计划记录；不改代码、CI、Release 或外部状态。 |

结论：`ready`（仅限 DD-DOC-008 的本地术语统一）。

### 2026-07-28T13:47:38+08:00 — DD-DOC-009 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-DOC-009`，依赖 `DD-DOC-003`、`DD-DOC-007` 均为 `completed`；验收要求是离线本地测试和 CI 均能拒绝故意植入的文档状态冲突，允许开始。 |
| 2 | 未发现 `AGENTS.md`。已重新读取本计划、执行清单、状态/证据约定、`CONTRIBUTING.md`、`GOVERNANCE.md`、ADR 规则和 `ci.yml`；该任务新增本地测试/CI 覆盖，不引入依赖、网络查询、命令执行、公开契约或外部状态改变。 |
| 3 | 工作树含 14 个既有已修改文件和 17 组既有未跟踪的产品化文档/Issue Form/模板；暂存区为空。分支为 `feature/productization-control-baseline`，HEAD 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`；相对本地 `origin/feature/productization-control-baseline` ahead/behind 为 `0/0`。本轮 GitHub API 只读查询的远端同名分支 SHA 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 5 | 本轮 GitHub CLI 只读查询列出 `v0.1.0-beta.3`（2026-07-27T12:39:48Z）、`.2` 和 `.1` 三个 Pre-release；列表不证明资产、哈希、attestation、安装或当前分支验证。 |
| 6 | 相关工作流为 `.github/workflows/ci.yml` 的 `CI` 和 `release.yml` 的 `Release`。最近可见 CI [#30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079) 成功但绑定历史 `main` SHA `aa611755554711dd44fab388f488fd2867ed093e`；最近 Release [#30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344) 成功但绑定历史 beta.3 SHA `3c46e62740143b62293f1abf526a1e159084e522`。 |
| 7 | 本轮 GitHub API 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`，target `branch`）。本任务不需要 GitHub 设置、Release、Provider、公开数据或 Git 交付；它只修改本地工作流文件和测试文件。 |
| 8 | 本轮允许修改离线检查实现/fixture、`ci.yml` 的本地测试入口、计划状态/执行日志和本清单记录；验证 `go test`、`go vet`、`git diff --check` 及故意漂移失败。明确排除网络真相查询、GitHub 设置、Release、提交、推送、PR 与合并。 |

结论：`ready`（仅限 DD-DOC-009 的本地离线漂移检查）。Go 1.25.12 正在被单一
官方后台下载并将在安装后现场验证；在此之前不把任何 Go 测试记为通过。

### 2026-07-28T14:04:18+08:00 — DD-DOC-010 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-DOC-010`，唯一依赖 `DD-DOC-009` 已完成；验收要求是覆盖 owner、ruleset、PVR、Release、required checks 和支持平台的人工复核清单及有效期，允许开始。 |
| 2 | 未发现 `AGENTS.md`。已复核计划、执行清单、状态/证据约定、CONTRIBUTING、GOVERNANCE、ADR 规则、治理审计、平台矩阵和 Release 清单；本轮只建立复核流程，不改变任何外部配置或公共契约。 |
| 3 | 工作树含既有的产品化文档、模板、工作流和 `internal/doccheck` 未提交修改；暂存区为空。分支为 `feature/productization-control-baseline`，HEAD 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`；相对本地 `origin/feature/productization-control-baseline` ahead/behind 为 `0/0`。本轮 GitHub API 只读查询的远端同名分支 SHA 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 5 | 本轮 GitHub CLI 只读查询列出 `v0.1.0-beta.3`（2026-07-27T12:39:48Z）、`.2` 和 `.1` 三个 Pre-release；这只证明查询时的列表可见，不验证资产、attestation、安装、当前分支或未来持续状态。 |
| 6 | 相关工作流为 `CI` 与 `Release`。最近可见 CI [#30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079) 成功但绑定历史 `main` SHA `aa611755554711dd44fab388f488fd2867ed093e`；最近 Release [#30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344) 成功但绑定历史 beta.3 SHA `3c46e62740143b62293f1abf526a1e159084e522`。 |
| 7 | 本轮 GitHub API 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`，target `branch`）。本任务不需要外部写；PVR、ruleset/required checks 或 Release 的改变仍需针对仓库、设置项、预期值和回滚方式的单独确认。 |
| 8 | 本轮只允许新增人工复核清单及更新计划/基线记录；检查字段完整性、链接和 Markdown/diff。明确排除 GitHub 设置、Release、Provider、公开数据、暂存、提交、推送、PR 与合并。 |

结论：`ready`（仅限 DD-DOC-010 的本地人工复核清单）。

### 2026-07-28T14:06:52+08:00 — DD-REL-008 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-REL-008`，唯一依赖 `DD-REL-007` 已完成；验收要求为无公开 Release 的隔离演练，含故障注入、清理和恢复证据，允许开始。 |
| 2 | 未发现 `AGENTS.md`。已复核计划、执行清单、状态/证据约定、CONTRIBUTING、GOVERNANCE、ADR 规则、`release-process.md`、`next-beta-release-checklist.md`、releaseprep 实现/测试和 Release 工作流；候选模式不授权 tag、Release 或远端写入。 |
| 3 | 工作树含既有产品化文档、模板、工作流和 `internal/doccheck` 未提交修改；暂存区为空。分支为 `feature/productization-control-baseline`，HEAD 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`；相对本地 `origin/feature/productization-control-baseline` ahead/behind 为 `0/0`。本轮 GitHub API 只读查询的远端同名分支 SHA 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 5 | 本轮 GitHub CLI 只读查询列出 `v0.1.0-beta.3`、`.2` 和 `.1` 三个 Pre-release；现有版本不被重用、不创建新 tag，列表不构成候选资产或发布批准。 |
| 6 | 相关工作流为 `CI` 与 `Release`。最近可见 CI #30286989079 和 Release #30266531344 均成功但绑定历史 SHA；它们不替代本次候选构建验证。 |
| 7 | 本轮 GitHub API 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`，target `branch`）。本任务仅作临时本地文件和 Git worktree 操作；明确排除 GitHub 设置、tag、Release、push、PR、合并、Provider 和公开数据。 |
| 8 | 本轮允许在系统临时目录创建/清理受控 detached worktree、预期失败的空输出目录和候选构建目录；使用 `releaseprep build --candidate`/`verify --smoke`。预期验证包括失败无半成品、清理后完整候选输出和无外部状态改变。 |

结论：`ready`（仅限 DD-REL-008 的本地隔离演练）。

### 2026-07-28T14:11:15+08:00 — DD-GOV-008 工作包

| # | 结果 |
|---|---|
| 1 | 目标任务为 `DD-GOV-008`，唯一依赖 `DD-GOV-004` 已完成；验收要求是新贡献者可完成一次文档或测试贡献演练，允许开始。 |
| 2 | 未发现 `AGENTS.md`。已复核计划、执行清单、状态/证据约定、CONTRIBUTING、GOVERNANCE、ADR 规则、Issue Forms、PR Template 和 beta compatibility；本轮只补充本地贡献文档，不改变稳定契约、依赖、Provider、Release 或外部设置。 |
| 3 | 工作树含既有产品化文档、模板、工作流和 `internal/doccheck` 未提交修改；暂存区为空。分支为 `feature/productization-control-baseline`，HEAD 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 4 | `origin` 为 `git@github.com:lliangcol/diffdossier.git`；相对本地 `origin/feature/productization-control-baseline` ahead/behind 为 `0/0`。本轮 GitHub API 只读查询的远端同名分支 SHA 为 `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`。 |
| 5 | 本轮 GitHub CLI 只读查询列出三个 beta Pre-release；不由此推断支持、安装、资产或发布批准。 |
| 6 | 相关工作流为 `CI` 与 `Release`。最近可见 CI #30286989079 和 Release #30266531344 都是历史 SHA 成功记录，不能替代当前分支验证。 |
| 7 | 本轮 GitHub API 只读查询显示 active 的 `Protect main` ruleset（ID `19772012`，target `branch`）。本任务无外部写；Issue/PR、ruleset、PVR 和 Release 操作均被明确排除。 |
| 8 | 本轮仅更新 CONTRIBUTING、计划/基线记录，并在临时目录执行文档贡献路径演练；验证 Markdown、链接、所列命令和 diff。 |

结论：`ready`（仅限 DD-GOV-008 的本地贡献指引）。
