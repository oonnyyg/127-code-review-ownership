# reviewassign — 离线评审分配工具

仅使用 Go 标准库。根据路径责任规则与团队名册，解释每项代码变更由谁负责，
核查给定批准名单的覆盖情况，并用固定贪心规则建议评审人。**不调用 GitHub，
不访问真实文件系统**，所有路径均为配置中的逻辑路径。

## 用法

```sh
go run . -config config.json -changes changes.json [-approvals approvals.json]
```

报告以 JSON 输出到标准输出。示例见 `examples/`。

## 配置（config.json）

```json
{
  "rules":  [{ "pattern": "src/**", "owners": ["@alice", "team:backend"] }],
  "teams":  [{ "name": "backend", "members": ["@bob", "team:core"] }],
  "submitter": "@bob",
  "unavailable": ["@carol"]
}
```

- `rules` 有序，**最后命中的规则覆盖前面的规则**；未命中任何规则的路径
  在报告中明确标记为缺口（`matched: false`）。
- 责任人写法：`@user` 或 `team:name`。未知团队在加载时即报错。
- 团队可嵌套，递归展开时**拒绝循环引用**并对成员去重。
- `submitter` 与 `unavailable` 中的人员不能成为本次有效评审人。

## 路径与模式语言

所有路径相对仓库根，使用正斜杠，不得有空段或单独的 `.` / `..` 段。
模式按完整路径匹配：

- `*` 匹配单段内任意字符（含零个）
- `?` 匹配单段内恰好一个字符
- `**` 仅在独占一段时合法，匹配零段或多段
- 其余字符按字面处理

**注意：该模式语言不等同于完整的 CODEOWNERS 语义。** 例如不支持
行内注释、取反（`!`）、字符类（`[...]`）、以 `/` 锚定与未锚定模式的区别、
以及 CODEOWNERS 的目录隐式后缀规则；`?` 在 gitignore/CODEOWNERS 中也不存在。
本工具只做上述明确定义的匹配。

## 变更清单（changes.json）

```json
{
  "changes": [
    { "type": "add",    "path": "src/new.go" },
    { "type": "modify", "path": "src/main.go" },
    { "type": "delete", "path": "src/old.go" },
    { "type": "rename", "old_path": "src/a.go", "new_path": "docs/a.md" }
  ]
}
```

重命名会**分别检查旧路径与新路径**，两者各自命中各自的规则。

## 覆盖核查与批准名单（approvals.json）

```json
{ "approvals": ["@alice"] }
```

- 每条路径由**至少一名有效责任人**批准即算覆盖。
- 非责任人的批准不能补齐覆盖。
- 无有效责任人的路径（未命中规则，或责任人全部为提交者/不可用人员）
  单独列入 `gaps` 并给出 `gap_reason`。

## 评审人建议

固定贪心规则：每一步选择能覆盖最多尚未覆盖路径的有效责任人，
同分按人员标识字典序，直到任何候选都不能新增覆盖为止。
报告中 `suggestion` 记录每一步的选择依据（覆盖哪些路径、剩余未覆盖数），
`remaining_gaps` 列出贪心结束后仍无有效责任人可覆盖的路径。
**该结果是贪心启发式，不宣称是人数最少的最优解**（报告 `note` 字段亦注明）。

## 测试

```sh
go test ./...
```

覆盖：规则后者覆盖前者、跨目录重命名、团队成员重叠去重、循环团队拒绝、
未知团队拒绝、提交者/不可用人员排除、非责任人批准不计覆盖、贪心与同分平局。
