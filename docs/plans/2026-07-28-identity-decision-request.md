# DD-DOC-001 身份决定请求

本文件收集维护者对仓库身份表述的明确、内容绑定决定。它不是决定本身，且不
把公开用户名、旧文档或 GitHub API 字段推断为同一自然人身份。

## 本轮只读事实（Captured-at：2026-07-28T11:26:53+08:00）

| 事实 | 来源 | 证据边界 |
|---|---|---|
| 公开仓库是 `lliangcol/diffdossier`，owner login 为 `lliangcol`，owner type 为 `User`，默认分支为 `main`。 | GitHub REST `repos/lliangcol/diffdossier` 的本轮只读结果。 | 证明账户字段和仓库归属，不证明该账户对应哪位自然人。 |
| `GOVERNANCE.md` 将 `liuliang1`（Liu Liang）称为维护者和 GitHub repository owner。 | `GOVERNANCE.md:3`。 | 是当前仓库文案，不是本轮人工确认。 |
| DCO 历史例外记录将 `liuliang1`、Liu Liang、copyright holder 和 prospective sign-off identity 关联。 | `docs/governance/dco-history-exception.md:4-7`。 | 是历史贡献权利记录；不自动确认当前 GitHub 账户关系或未来权限。 |
| 发布流程也引用 Liu Liang 的版权与签署身份。 | `docs/release-process.md:32-35`。 | 是已有流程文案，不替代维护者决定。 |

## 需要维护者确认的决定

请由有权代表仓库维护与版权/签署事实的维护者明确确认以下内容，并说明决定
日期与其权限来源：

1. `lliangcol` GitHub User 账户与 `liuliang1` 维护者标识是否代表同一自然人；
   若否，分别说明其关系与可公开表述。
2. Liu Liang 是否为可公开使用的姓名；若是，是否与上述一个或两个账户标识对应。
3. 各标识的职责边界：GitHub repository owner、日常 maintainer、版权持有人、
   DCO prospective sign-off identity，以及 Release/安全/公开数据批准角色（如不同）。
4. 是否允许在 `GOVERNANCE.md`、`NOTICE`、`SECURITY.md`、`SUPPORT.md`、
   `CONTRIBUTING.md` 和 Release 文档中按该决定统一表述；这项确认只授权文档
   表述，不能授权 GitHub 设置、Release、外部发布或权限变更。

## 可粘贴的确认格式

```text
DD-DOC-001 decision
Date: <ISO 8601 with timezone>
Authority: <role and basis for making this confirmation>
Relationship: <state whether lliangcol, liuliang1, and Liu Liang identify the same person; if not, state the public relationship>
Roles:
- GitHub repository owner: <identifier>
- Maintainer: <identifier or role>
- Copyright holder: <legal/public name or role>
- DCO prospective sign-off identity: <name and email, or state no change>
- Release/security/public-data approver: <identifier or role>
Documentation scope: <confirm or deny the listed repository documents>
Expiry/review trigger: <if any>
```

Until this decision is supplied, `DD-DOC-001` remains `in_progress`; no identity
document is normalized and no ownership, approval, copyright, or signing claim
is upgraded.

## 已确认决定（2026-07-28T11:29:55+08:00）

维护者在本工作包中明确确认：`lliangcol`、`liuliang1` 与 Liu Liang 是同一人。
该确认解决的是自然人关系，结合本轮只读 GitHub owner 字段和既有的 DCO/NOTICE
记录，DD-DOC-001 的职责映射如下：

| 职责 | 公开表述与证据 |
|---|---|
| GitHub repository owner | `lliangcol` GitHub User 账户；本轮 GitHub REST 只读结果为 `lliangcol/diffdossier` 的 owner。 |
| Maintainer | `liuliang1`（Liu Liang），依 `GOVERNANCE.md:3` 与本次同一人确认。 |
| Copyright holder | Liu Liang，依 `NOTICE:2` 和 DCO 历史例外记录。 |
| DCO prospective sign-off identity | `Liu Liang <lliang@outlook.com>`，依 `docs/governance/dco-history-exception.md:7`；本次未修改该身份。 |
| Release、安全与公开数据的批准角色 | 同一人以 GitHub repository owner/maintainer 身份履行既有治理规则要求的明确 owner review；每项敏感动作仍须内容绑定的单独授权。 |

本决定只完成关系和职责确认，不自动变更任何 GitHub 设置、权限、Release、公开
数据或仓库文案。`DD-DOC-002` 将在其自身工作包中统一文档表述并重新审阅。
