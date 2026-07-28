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
