# 产品化分析差距矩阵

本矩阵落实 `DD-CTRL-004`。范围仅为
[`2026-07-27-productization-todo.md`](2026-07-27-productization-todo.md)
“当前基线”中的八条分析结论（第 20～27 行）；它们是历史分析输入，不是任务
完成声明。分类只描述该结论的当前证据状态，后续任务仍必须独立取得其完成证据。

## 分类规则

- **已完成：** 本轮有足以确认该事实的本地或只读 live 证据；不等于依赖此事实
  的产品闭环完成。
- **部分完成：** 结论的一部分已证实，但其明确的验证、兼容性或交付缺口仍在。
- **仍缺失：** 仓库证据表明能力、文档或人工决定尚未具备。
- **需 live 验证：** 结论依赖可漂移的 Git、GitHub 或运行状态；即使本轮读取到
  一致结果，后续工作包也必须重新查询。

## 矩阵（Captured-at：2026-07-28T11:20:22+08:00）

| 结论 ID | 分析结论与来源 | 分类 | 本轮证据与边界 | 映射计划 ID |
|---|---|---|---|---|
| A-01 | 本地分支、`origin/main` 与工作树状态是当时基线，且本地远端跟踪引用不是 GitHub 实时证明（计划第 20 行）。 | 需 live 验证 | 当前分支 `feature/productization-control-baseline` 的 HEAD 与 GitHub 只读查询 `main` 同为 `aa611755554711dd44fab388f488fd2867ed093e`；相对本地 `origin/main` 为 `0/0`。但工作树含计划文档的暂存/未暂存变更，且这些状态随时可变。 | DD-CTRL-001（已完成的清单机制）；所有后续工作包 |
| A-02 | 三个 beta tag/Pre-release 存在，beta.3 列出制品；制品存在不等于哈希、attestation 或安装已验证，且不得重发旧 tag（计划第 21 行）。 | 部分完成 | GitHub 只读查询确认 `v0.1.0-beta.1`～`.3` 三个 Pre-release 仍存在。未在本轮下载/核验资产、SHA256、SBOM、attestation 或四平台安装；因此不能提升为 Release 闭环已完成。 | DD-REL-001、DD-REL-002、DD-REL-003、DD-REL-005～DD-REL-010；DD-UXD-005 |
| A-03 | 多份现时文档仍有“无公开 Release”或“原生验证待完成”等过期表达（计划第 22 行）。 | 仍缺失 | 本轮仓库搜索仍在 README、checkpoint 与主题文档发现“无 public release”或 native/Tier 1 待验证的历史/现时表述；尚未建立 Release 状态对账矩阵，也未完成文档修正与离线漂移检查。 | DD-DOC-003～DD-DOC-010；DD-REL-004、DD-REL-006；DD-P0-EXIT-001 |
| A-04 | `liuliang1`、`lliangcol` 与 Liu Liang 的身份和 owner/maintainer 表述不能依用户名自动推断（计划第 23 行）。 | 仍缺失 | `GOVERNANCE.md` 仍作出该身份关联表述，远端路径为 `lliangcol/diffdossier`；本轮未获得维护者的内容绑定人工决定。 | DD-DOC-001、DD-DOC-002、DD-DOC-011；DD-GOV-005 |
| A-05 | 仓库缺少 Issue Form、PR Template 与 CODEOWNERS（计划第 24 行）。 | 仍缺失 | 本轮 `.github` 清单仍仅含 `ci.yml` 与 `release.yml`，未发现上述文件。未对 GitHub 设置作写入。 | DD-GOV-002～DD-GOV-005、DD-GOV-008～DD-GOV-012；DD-P0-EXIT-005 |
| A-06 | CLI 缺少 `quickstart`、`next`、`explain`、guided workflow 与 Shell completion（计划第 25 行）。 | 仍缺失 | 本轮对 `cmd` 与 `internal` 的命令/实现搜索未发现这些公开命令或 completion 实现；现有精确命令不构成引导层。 | DD-CLI-001～DD-CLI-016；DD-UXD-006、DD-UXD-012；DD-P1-EXIT-003、DD-P1-EXIT-009 |
| A-07 | Config/输出 Schema 的约束和 Schema 测试不足（计划第 26 行）。 | 仍缺失 | 本轮未发现已完成的公共 Schema 清单、Draft/fixture/Go 一致性/兼容矩阵；当前检索仍指向 `schemas/config.schema.json` 和现有实现。未将“可解析”误记为兼容验证。 | DD-SCH-001～DD-SCH-011；DD-P1-EXIT-002 |
| A-08 | 风险覆盖使用 `filepath.Match`，没有 DiffDossier Glob 规范（计划第 27 行）。 | 仍缺失 | `internal/risk/risk.go:59` 和 `internal/risk/policy.go:119` 仍调用 `filepath.Match`；本轮未发现独立 Glob 规范或 matcher。 | DD-RISK-001～DD-RISK-011；DD-P1-EXIT-004 |

## 使用限制与下一步

本矩阵没有把已完成的 `DD-CTRL-001`～`003` 或本轮成功 CI 外推为任何 P0/P1
出口完成。A-01 的 live 状态将在每个工作包重新验证；A-02 的 Release 存在事实
不能替代资产与安装证据；A-03～A-08 仍由各自的原子任务关闭。下一项未阻塞的
控制任务是 `DD-CTRL-005`；需要人工身份决定的 `DD-DOC-001` 保持其门禁。
