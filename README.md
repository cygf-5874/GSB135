# tarcanon

确定性 tar 归档库（Go 1.24，仅标准库，`go.mod` 无 `require`），用于可复现构建：
同一目录树在任意时刻、任意机器上产出**逐字节一致**的归档。

- 语言/依赖：Go 1.24，仅标准库（`archive/tar` 属标准库），无第三方依赖。
- 模块名即库名：`tarcanon`；库代码在仓库根（`tarcanon.go`）。
- 构建：`bash scripts/build.sh`；自检：`bash scripts/check.sh`（`check/` 是固定验收程序，**勿改**）；
  既有用例：`go test ./...`。

## 怎么跑

```bash
go build ./...            # 或 bash scripts/build.sh
go test ./...             # 既有用例
go run ./check            # 固定验收程序（8 个场景）
go run ./check -list      # 列出全部场景
go run ./check --only order
bash scripts/check.sh     # 薄封装：go run ./check "$@"
```

## 对外契约

下面 9 条是 `tarcanon` 的行为契约（`Entry`、`Canonical`、`Archive`）。

1. **逐字节可复现**：同一组条目 / 同一目录树，任意时刻、任意机器生成的 tar 字节流
   **完全一致**。
2. **header 归一**：每个条目的 header 必须写死为 `Uid=0`、`Gid=0`、`Uname=""`、`Gname=""`、
   `ModTime` 固定为 **Unix 纪元 0（1970-01-01T00:00:00Z，秒与纳秒都为 0）**；
   `Mode` **只保留权限位**（低 9 位，去掉 setuid/setgid/sticky 等高位）。
3. **条目顺序**：所有条目按 `Path` 的**字节序升序**排列。
4. **目录先于子项**：目录条目必须出现在它所有子项之前；同一父目录下的顺序仍按第 3 条。
5. **扩展头统一**：路径长度超过 100 字节（或长链接）必须使用**统一的一种**扩展头格式 ——
   本题固定为 **PAX**（`tar.FormatPAX` / `archive/tar` 的 PAX 扩展头），不得混用 GNU 扩展头。
6. **环境无关**：输出与 `LC_ALL`、`LANG`、`TZ` 等环境变量无关。
7. **空归档**：空条目集合（或空目录）必须产出合法的空 tar（可被 `archive/tar` 正常读到
   `io.EOF`），不得为空字节流。
8. **不污染归档树**：生成过程中不得在被归档的目录树内创建任何临时文件。
9. **重复调用一致**：对同一输入重复调用（包括中途修改 `TZ`）必须得到**逐字节相同**的结果。

`Entry` 字段：`Path`（相对路径，`/` 分隔，无前导 `./`）、`IsDir`、`Mode`（权限位）、
`Body`（文件内容；目录忽略）。`Archive(root)` 归档 `root` 下的普通文件与目录（路径相对
`root`，不含 `root` 自身；不递归进符号链接）。

## 固定件场景（8 个）

| 分组 | 场景 | 覆盖 |
| --- | --- | --- |
| `format` | `F1_header_fields_normalized` | uid/gid/uname/gname/mtime/mode 归一 |
| `format` | `F2_content_roundtrip` | 输出是合法 tar，内容可原样读回 |
| `order` | `O1_path_byte_order` | 条目按 path 字节序升序 |
| `order` | `O2_dir_before_children` | 目录条目先于其子项 |
| `stable` | `S1_two_calls_identical` | 两次 Canonical / Archive 逐字节一致 |
| `stable` | `S2_tz_independent` | 更换 `TZ` 后仍逐字节一致 |
| `meta` | `M1_long_path_pax` | 长路径用 PAX 扩展头并原样还原 |
| `meta` | `M2_empty_tree` | 空树产出合法空归档 |

`--only <组名>`（`format` / `order` / `stable` / `meta`）可只跑一组。失败不早退。

## 目录

```
.
├── .gitattributes
├── .gitignore
├── go.mod               module tarcanon
├── tarcanon.go          待补实现（方法体为 panic("not implemented")）
├── tarcanon_test.go     既有用例
├── PROMPT.md            本题的 User Prompt（逐字）
├── README.md
├── check/main.go        固定验收程序（勿改）
└── scripts/
    ├── build.sh         go build ./...
    └── check.sh         薄封装：go run ./check "$@"
```