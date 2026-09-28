// Package tarcanon 生成确定性的 tar 归档：同一目录树在任意时刻、任意机器上
// 产出逐字节一致的字节流，供可复现构建使用。
//
// 完整语义见 README 的「对外契约」一节。
package tarcanon

// Entry 描述归档里的一个条目。
type Entry struct {
	Path  string // 相对路径，'/' 分隔，无前导 './'
	IsDir bool   // 目录条目
	Mode  int64  // 权限位（只保留低 9 位）
	Body  []byte // 文件内容；目录忽略
}

// Canonical 把 entries 归一化、按确定顺序编码为 tar 字节流。
func Canonical(entries []Entry) ([]byte, error) {
	panic("not implemented")
}

// Archive 读取 root 目录树并生成确定性 tar 字节流。
func Archive(root string) ([]byte, error) {
	panic("not implemented")
}
