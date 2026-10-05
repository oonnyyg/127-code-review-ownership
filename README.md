# reviewassign — 离线代码评审分配工具

`reviewassign` 根据 JSON 配置中的**有序路径责任规则**与**团队名册**，
离线解释每项代码变更由谁负责、核查给定批准名单是否覆盖全部变更，
并用固定贪心规则建议评审人。工具只读取两个 JSON 输入文件，
**不访问真实文件系统，也不调用 GitHub**。

## 构建与运行

```sh
go build ./cmd/reviewassign
./reviewassign -config examples/config.json -changes examples/changes.json
```

输出为 JSON 报告（见下文）。测试：`go test ./...`

## 配置（config JSON）

```json
{
  "people": {"alice": {}, "erin": {"unavailable": true}},
  "teams": {
    "backend":  {"members": ["alice", "bob"]},
    "all-devs": {"members": ["dave"], "teams": ["backend"]}
  },
  "rules": [
    {"pattern": "**", "owners": ["@all-devs"]},
    {"pattern": "src/**", "owners": ["@backend"]}
  ]
}
```

- `rules` 有序，**最后命中的规则覆盖前面的规则**；未命中任何规则的路径
  在报告中以 `matched: false` 明确标记。
- 规则 owner 为个人 ID，或以 `@` 前缀引用团队；团队可嵌套（`teams` 字段），
  递归展开时**拒绝循环引用**并对成员**去重**；引用未知团队或未知人员直接报错。
- `unavailable: true` 的人员与变更提交者（`author`）不能成为本次有效评审人，
  会从 `eligible_reviewers` 中排除（仍列在 `owners` 中以便解释）。

## 变更清单（changes JSON）

```json
{
  "author": "dave",
  "approvals": ["alice"],
  "changes": [
    {"kind": "modify", "path": "src/server.go"},
    {"kind": "rename", "old_path": "web/old/app.ts", "new_path": "src/web/app.ts"}
  ]
}
```

- `kind` 支持 `add`、`modify`、`delete`、`rename`；**重命名分别检查旧路径与
  新路径**，两者各自独立参与规则匹配与覆盖判定。
- 所有路径相对仓库根、使用正斜杠；不允许空段、`.`、`..` 或反斜杠。

## 路径模式语言

模式按**完整路径**逐段匹配：

- `*` 匹配单个段内任意字符（不跨 `/`）；
- `?` 匹配单个字符；
- `**` 仅在**独占一段**时合法，匹配零段或多段；
- 其余字符一律按字面处理（无字符类、无取反、无锚定语法）。

> **注意：该模式语言不等同于完整的 CODEOWNERS 语法。** CODEOWNERS 还包含
> 行顺序之外的语义（如 `!` 取反、`[...]` 字符类、前导 `/` 锚定、目录隐式
> 递归等，具体取决于平台实现），本工具均不支持，请勿直接混用。

## 覆盖判定与批准核查

- 每条路径由**至少一名有效责任人**（在册、非提交者、可用）批准即算覆盖。
- 核查 `approvals` 时，**非责任人的批准不计入覆盖**。
- 没有任何有效责任人的路径列入 `gaps` 单独报告。

## 评审人建议（固定贪心）

每次选择能覆盖最多**尚未覆盖**路径的人，同分按人员 ID 字典序，直到无人能
新增覆盖。报告中 `suggestion` 给出每步选择依据（`covers`），
`remaining_gaps` 给出剩余缺口。**该结果是确定性贪心解，不宣称是人数最少的
最优解。**

## JSON 报告字段

- `paths[]`：每条路径的命中规则、展开后的责任人、有效评审人、计入的批准与覆盖状态；
- `gaps`：无有效责任人的路径；
- `uncovered`：未被有效批准覆盖的路径；
- `suggestion` / `suggested_reviewers`：贪心建议及依据；
- `remaining_gaps`：建议后仍无法覆盖的路径。
