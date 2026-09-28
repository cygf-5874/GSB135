归档产物要求逐字节可复现。tarcanon 是 Go 1.24 的确定性 tar 归档库，仅标准库，构建走 scripts/build.sh。
模块名即库名 `tarcanon`，`go.mod` 不写 `require`；既有用例走 `go test ./...`，自检走 `bash scripts/check.sh`。

`Canonical` 与 `Archive` 目前是空实现，方法体直接 `panic`。请按 README「对外契约」的 9 条把它实现出来，
让既有用例转绿、固定件 `check/` 的 8 个场景全过。语义细节以 README 为准。

验收：
- go test ./... 全绿；
- bash scripts/check.sh 退出码 0，8 个场景全过（format 2 + order 2 + stable 2 + meta 2）。

约束：
1. 不改 `check/`；`Entry` / `Canonical` / `Archive` 的名字与签名不许改。
2. 既有用例一条都不许删改。
3. 仅标准库（`archive/tar` 属标准库），不引入任何第三方依赖。
4. 输出必须确定：不依赖墙钟、时区、locale 或 map 迭代顺序。