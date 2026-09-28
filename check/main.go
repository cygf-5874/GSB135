// Command check 是 tarcanon 的固定验收程序。
//
// ⚠️ 不要修改本文件。它是判定「题目有没有做对」的依据；
// 修改它只会让判定失效，不会让实现变对。
//
// 8 个场景分四组：
//
//	format 2：header 归一（uid/gid/uname/gname/mtime/mode）/ 内容往返可读
//	order  2：条目按 path 字节序升序 / 目录条目先于子项
//	stable 2：两次调用字节一致 / 更换 TZ 后仍字节一致
//	meta   2：长路径用 PAX 扩展头并原样还原 / 空树产出合法空归档
//
// 起点是空壳（函数体 panic）时，每个场景都会被 recover 成 FAIL；
// 判据全部是硬编码期望值，不依赖墙钟、时区或 map 迭代顺序。
//
// 用法：
//
//	go run ./check                # 跑全部场景
//	go run ./check --only order   # 只跑一组
//	go run ./check -list          # 列出场景
package main

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tarcanon"
)

type scenario struct {
	group string
	name  string
	run   func() error
}

var scenarios = []scenario{
	{"format", "F1_header_fields_normalized", f1},
	{"format", "F2_content_roundtrip", f2},
	{"order", "O1_path_byte_order", o1},
	{"order", "O2_dir_before_children", o2},
	{"stable", "S1_two_calls_identical", s1},
	{"stable", "S2_tz_independent", s2},
	{"meta", "M1_long_path_pax", m1},
	{"meta", "M2_empty_tree", m2},
}

var groups = []string{"format", "order", "stable", "meta"}

func main() {
	only := ""
	list := false
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch {
		case a == "-list":
			list = true
		case a == "--only":
			if i+1 < len(os.Args) {
				only = os.Args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--only="):
			only = strings.TrimPrefix(a, "--only=")
		}
	}

	if list {
		for _, sc := range scenarios {
			fmt.Printf("%-8s %s\n", sc.group, sc.name)
		}
		return
	}

	if only != "" {
		found := false
		for _, g := range groups {
			if g == only {
				found = true
			}
		}
		if !found {
			fmt.Printf("未知分组 %q（可选：format / order / stable / meta）\n", only)
			os.Exit(2)
		}
	}

	passed, total := 0, 0
	for _, sc := range scenarios {
		if only != "" && sc.group != only {
			continue
		}
		total++
		if err := runScenario(sc); err != nil {
			fmt.Printf("FAIL %s/%s  %s\n", sc.group, sc.name, err.Error())
			continue
		}
		passed++
		fmt.Printf("PASS %s/%s\n", sc.group, sc.name)
	}
	fmt.Printf("结果：通过 %d/%d\n", passed, total)
	if passed != total {
		os.Exit(1)
	}
}

func runScenario(sc scenario) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("场景正常结束（不 panic） 实际=panic: " + fmt.Sprint(r))
		}
	}()
	return sc.run()
}

// ---------------------------------------------------------------- 工具

type detailError struct{ msg string }

func (e *detailError) Error() string { return e.msg }

func expect(what, got string) error {
	return &detailError{msg: "期望=" + what + " 实际=" + got}
}

type rec struct {
	name string
	hdr  *tar.Header
	body []byte
}

func readAll(data []byte) ([]rec, error) {
	r := tar.NewReader(bytes.NewReader(data))
	var out []rec
	for {
		h, err := r.Next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		b, _ := io.ReadAll(r)
		out = append(out, rec{name: h.Name, hdr: h, body: b})
	}
}

func names(rs []rec) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.name
	}
	return out
}

func canon(entries []tarcanon.Entry) ([]byte, error) {
	return tarcanon.Canonical(entries)
}

// ---------------------------------------------------------------- format

func f1() error {
	entries := []tarcanon.Entry{
		{Path: "a.txt", Mode: 0o644, Body: []byte("hello")},
		{Path: "sub", IsDir: true, Mode: 0o755},
		{Path: "sub/b.bin", Mode: 0o600, Body: []byte{0, 1, 2, 3}},
	}
	data, err := canon(entries)
	if err != nil {
		return expect("Canonical 成功", err.Error())
	}
	rs, err := readAll(data)
	if err != nil {
		return expect("输出是合法 tar", err.Error())
	}
	if len(rs) != 3 {
		return expect("条目数=3", strconv.Itoa(len(rs)))
	}
	want := map[string]int64{"a.txt": 0o644, "sub": 0o755, "sub/b.bin": 0o600}
	for _, r := range rs {
		if r.hdr.Uid != 0 || r.hdr.Gid != 0 {
			return expect(r.name+" 的 Uid=Gid=0", fmt.Sprintf("uid=%d gid=%d", r.hdr.Uid, r.hdr.Gid))
		}
		if r.hdr.Uname != "" || r.hdr.Gname != "" {
			return expect(r.name+" 的 Uname=Gname=\"\"", fmt.Sprintf("uname=%q gname=%q", r.hdr.Uname, r.hdr.Gname))
		}
		if r.hdr.ModTime.Unix() != 0 || r.hdr.ModTime.Nanosecond() != 0 {
			return expect(r.name+" 的 Mtime 固定为纪元 0", r.hdr.ModTime.String())
		}
		m, ok := want[r.name]
		if !ok {
			return expect("条目名属于输入集合", strconv.Quote(r.name))
		}
		if r.hdr.Mode != m {
			return expect(fmt.Sprintf("%s 的 Mode=%#o（只含权限位）", r.name, m), fmt.Sprintf("%#o", r.hdr.Mode))
		}
	}
	return nil
}

func f2() error {
	entries := []tarcanon.Entry{
		{Path: "a.txt", Mode: 0o644, Body: []byte("alpha")},
		{Path: "b", IsDir: true, Mode: 0o755},
		{Path: "b/c.bin", Mode: 0o600, Body: []byte{0xde, 0xad, 0xbe, 0xef}},
	}
	data, err := canon(entries)
	if err != nil {
		return expect("Canonical 成功", err.Error())
	}
	rs, err := readAll(data)
	if err != nil {
		return expect("输出是合法 tar", err.Error())
	}
	files := map[string][]byte{}
	for _, r := range rs {
		if r.hdr.Typeflag == tar.TypeDir {
			continue
		}
		files[r.name] = r.body
	}
	if !bytes.Equal(files["a.txt"], []byte("alpha")) {
		return expect("a.txt 内容=alpha", strconv.Quote(string(files["a.txt"])))
	}
	if !bytes.Equal(files["b/c.bin"], []byte{0xde, 0xad, 0xbe, 0xef}) {
		return expect("b/c.bin 内容=deadbeef", fmt.Sprintf("%x", files["b/c.bin"]))
	}
	for _, r := range rs {
		if r.name == "b" && r.hdr.Typeflag != tar.TypeDir {
			return expect("b 是目录条目", strconv.Itoa(int(r.hdr.Typeflag)))
		}
	}
	return nil
}

// ----------------------------------------------------------------- order

func o1() error {
	entries := []tarcanon.Entry{
		{Path: "c.txt", Mode: 0o644, Body: []byte("c")},
		{Path: "aa.txt", Mode: 0o644, Body: []byte("aa")},
		{Path: "b/b.txt", Mode: 0o644, Body: []byte("bb")},
		{Path: "b", IsDir: true, Mode: 0o755},
		{Path: "a.txt", Mode: 0o644, Body: []byte("a")},
	}
	data, err := canon(entries)
	if err != nil {
		return expect("Canonical 成功", err.Error())
	}
	rs, err := readAll(data)
	if err != nil {
		return expect("输出是合法 tar", err.Error())
	}
	want := []string{"a.txt", "aa.txt", "b", "b/b.txt", "c.txt"}
	got := names(rs)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		return expect("条目顺序="+strings.Join(want, ","), strings.Join(got, ","))
	}
	return nil
}

func o2() error {
	entries := []tarcanon.Entry{
		{Path: "d/y.txt", Mode: 0o644, Body: []byte("y")},
		{Path: "d", IsDir: true, Mode: 0o755},
		{Path: "d/x.txt", Mode: 0o644, Body: []byte("x")},
	}
	data, err := canon(entries)
	if err != nil {
		return expect("Canonical 成功", err.Error())
	}
	rs, err := readAll(data)
	if err != nil {
		return expect("输出是合法 tar", err.Error())
	}
	want := []string{"d", "d/x.txt", "d/y.txt"}
	got := names(rs)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		return expect("目录 d 先于其子项="+strings.Join(want, ","), strings.Join(got, ","))
	}
	return nil
}

// ---------------------------------------------------------------- stable

func sampleEntries() []tarcanon.Entry {
	return []tarcanon.Entry{
		{Path: "root.txt", Mode: 0o644, Body: []byte("r")},
		{Path: "k", IsDir: true, Mode: 0o755},
		{Path: "k/2.txt", Mode: 0o600, Body: []byte("two")},
		{Path: "k/1.txt", Mode: 0o644, Body: []byte("one")},
	}
}

func s1() error {
	first, err := canon(sampleEntries())
	if err != nil {
		return expect("Canonical 成功", err.Error())
	}
	second, err := canon(sampleEntries())
	if err != nil {
		return expect("Canonical 成功", err.Error())
	}
	if !bytes.Equal(first, second) {
		return expect("两次 Canonical 逐字节一致", fmt.Sprintf("长度 %d vs %d", len(first), len(second)))
	}

	dir, err := os.MkdirTemp("", "tarcanon-s1")
	if err != nil {
		return expect("创建临时目录", err.Error())
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "n.txt"), []byte("n"), 0o644); err != nil {
		return expect("写样本文件", err.Error())
	}
	a, err := tarcanon.Archive(dir)
	if err != nil {
		return expect("Archive 成功", err.Error())
	}
	b, err := tarcanon.Archive(dir)
	if err != nil {
		return expect("Archive 成功", err.Error())
	}
	if !bytes.Equal(a, b) {
		return expect("两次 Archive 逐字节一致", fmt.Sprintf("长度 %d vs %d", len(a), len(b)))
	}
	return nil
}

func s2() error {
	dir, err := os.MkdirTemp("", "tarcanon-s2")
	if err != nil {
		return expect("创建临时目录", err.Error())
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "t.txt"), []byte("t"), 0o644); err != nil {
		return expect("写样本文件", err.Error())
	}

	old, had := os.LookupEnv("TZ")
	defer func() {
		if had {
			os.Setenv("TZ", old)
		} else {
			os.Unsetenv("TZ")
		}
	}()

	os.Setenv("TZ", "UTC")
	a, err := tarcanon.Archive(dir)
	if err != nil {
		return expect("Archive 成功", err.Error())
	}
	os.Setenv("TZ", "Asia/Tokyo")
	b, err := tarcanon.Archive(dir)
	if err != nil {
		return expect("Archive 成功", err.Error())
	}
	os.Setenv("TZ", "America/New_York")
	c, err := tarcanon.Archive(dir)
	if err != nil {
		return expect("Archive 成功", err.Error())
	}
	if !bytes.Equal(a, b) || !bytes.Equal(b, c) {
		return expect("不同 TZ 下 Archive 逐字节一致", fmt.Sprintf("长度 %d/%d/%d", len(a), len(b), len(c)))
	}
	return nil
}

// ------------------------------------------------------------------ meta

func m1() error {
	long := strings.Repeat("a", 200) + ".txt"
	entries := []tarcanon.Entry{
		{Path: "short.txt", Mode: 0o644, Body: []byte("s")},
		{Path: long, Mode: 0o644, Body: []byte("L")},
	}
	data, err := canon(entries)
	if err != nil {
		return expect("Canonical 成功", err.Error())
	}
	rs, err := readAll(data)
	if err != nil {
		return expect("输出是合法 tar", err.Error())
	}
	var found *rec
	for i := range rs {
		if rs[i].name == long {
			found = &rs[i]
		}
	}
	if found == nil {
		return expect("长路径条目名原样还原", fmt.Sprintf("未找到长度 %d 的路径", len(long)))
	}
	if found.hdr.Format != tar.FormatPAX {
		return expect("长路径使用 PAX 扩展头", found.hdr.Format.String())
	}
	return nil
}

func m2() error {
	data, err := canon(nil)
	if err != nil {
		return expect("空输入 Canonical 成功", err.Error())
	}
	rs, err := readAll(data)
	if err != nil {
		return expect("空归档是合法 tar", err.Error())
	}
	if len(rs) != 0 {
		return expect("空归档没有条目", strconv.Itoa(len(rs)))
	}

	dir, err := os.MkdirTemp("", "tarcanon-m2")
	if err != nil {
		return expect("创建临时目录", err.Error())
	}
	defer os.RemoveAll(dir)
	empty, err := tarcanon.Archive(dir)
	if err != nil {
		return expect("空目录 Archive 成功", err.Error())
	}
	ers, err := readAll(empty)
	if err != nil {
		return expect("空目录归档是合法 tar", err.Error())
	}
	if len(ers) != 0 {
		return expect("空目录归档没有条目", strconv.Itoa(len(ers)))
	}
	return nil
}
