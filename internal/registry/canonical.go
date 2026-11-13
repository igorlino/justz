// Package registry holds the rules for the command registry (commands.json),
// starting with its canonical order.
package registry

import (
	"sort"
	"strconv"
	"strings"

	"github.com/igorlino/justz/internal/orderedjson"
)

// Canonicalize rewrites root, in place, into the canonical order:
//
//   - groups: "core" first, then the other keys A to Z
//   - commands and extra in each group: keys A to Z
//   - args of each command: positional arguments first, then flags, each
//     ignoring case
//   - examples: by their cmd; seeAlso: A to Z
//   - tiers[].groups, dispatchers and lab: A to Z; order: "core" first
//
// Arrays named "values" keep the order the author wrote, because they are
// semantic scales such as low, medium, high.
func Canonicalize(root *orderedjson.Value) {
	if root == nil || root.Kind != orderedjson.Object {
		return
	}
	if groups := root.Get("groups"); groups != nil && groups.Kind == orderedjson.Object {
		sortMembers(groups, func(a, b *orderedjson.Member) bool {
			ra, rb := coreRank(a.Key), coreRank(b.Key)
			if ra != rb {
				return ra < rb
			}
			return a.Key < b.Key
		})
		for _, m := range groups.Obj {
			canonicalizeGroup(m.Val)
		}
	}
	if tiers := root.Get("tiers"); tiers != nil && tiers.Kind == orderedjson.Array {
		for _, tier := range tiers.Arr {
			if g := tier.Get("groups"); g != nil && g.Kind == orderedjson.Array {
				sortValues(g)
			}
		}
	}
	for _, name := range []string{"dispatchers", "lab"} {
		if v := root.Get(name); v != nil && v.Kind == orderedjson.Array {
			sortValues(v)
		}
	}
	if order := root.Get("order"); order != nil && order.Kind == orderedjson.Array {
		sort.SliceStable(order.Arr, func(i, j int) bool {
			ri, rj := coreRankValue(order.Arr[i]), coreRankValue(order.Arr[j])
			if ri != rj {
				return ri < rj
			}
			return compare(order.Arr[i], order.Arr[j]) < 0
		})
	}
}

func canonicalizeGroup(g *orderedjson.Value) {
	if g == nil || g.Kind != orderedjson.Object {
		return
	}
	for _, name := range []string{"extra", "commands"} {
		cmds := g.Get(name)
		if cmds == nil || cmds.Kind != orderedjson.Object {
			continue
		}
		sortMembers(cmds, func(a, b *orderedjson.Member) bool { return a.Key < b.Key })
		for _, m := range cmds.Obj {
			canonicalizeCommand(m.Val)
		}
	}
	if ex := g.Get("examples"); ex != nil && ex.Kind == orderedjson.Array {
		sort.SliceStable(ex.Arr, func(i, j int) bool {
			return compare(cmdOf(ex.Arr[i]), cmdOf(ex.Arr[j])) < 0
		})
	}
	if see := g.Get("seeAlso"); see != nil && see.Kind == orderedjson.Array {
		sortValues(see)
	}
}

func canonicalizeCommand(c *orderedjson.Value) {
	if c == nil || c.Kind != orderedjson.Object {
		return
	}
	if args := c.Get("args"); args != nil && args.Kind == orderedjson.Array {
		sort.SliceStable(args.Arr, func(i, j int) bool {
			return lessKey(argKey(args.Arr[i]), argKey(args.Arr[j]))
		})
	}
}

func cmdOf(v *orderedjson.Value) *orderedjson.Value {
	if c := v.Get("cmd"); c != nil {
		return c
	}
	return &orderedjson.Value{Kind: orderedjson.Null}
}

func coreRank(key string) int {
	if key == "core" {
		return 0
	}
	return 1
}

func coreRankValue(v *orderedjson.Value) int {
	if v.Kind == orderedjson.String {
		return coreRank(v.Str)
	}
	return 1
}

func sortMembers(obj *orderedjson.Value, less func(a, b *orderedjson.Member) bool) {
	sort.SliceStable(obj.Obj, func(i, j int) bool { return less(obj.Obj[i], obj.Obj[j]) })
}

func sortValues(arr *orderedjson.Value) {
	sort.SliceStable(arr.Arr, func(i, j int) bool { return compare(arr.Arr[i], arr.Arr[j]) < 0 })
}

// argKey builds the sort key of an argument from its name:
// flags (starting with a dash) sort after positional arguments.
func argKey(arg *orderedjson.Value) []string {
	name := ""
	if n := arg.Get("name"); n != nil && n.Kind == orderedjson.String {
		name = n.Str
	}
	if strings.HasPrefix(name, "-") {
		clean := strings.TrimLeft(name, "-")
		return []string{"1", asciiLower(clean), clean, name}
	}
	stripped := strings.TrimLeft(name, "[]<*+")
	return []string{"0", asciiLower(stripped), name}
}

func lessKey(a, b []string) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

// compare orders two JSON values the way jq does: null, false, true, numbers,
// strings, arrays, objects; strings by code point.
func compare(a, b *orderedjson.Value) int {
	ra, rb := rank(a), rank(b)
	if ra != rb {
		return ra - rb
	}
	switch a.Kind {
	case orderedjson.Number:
		fa, _ := strconv.ParseFloat(a.Str, 64)
		fb, _ := strconv.ParseFloat(b.Str, 64)
		switch {
		case fa < fb:
			return -1
		case fa > fb:
			return 1
		}
		return 0
	case orderedjson.String:
		return strings.Compare(a.Str, b.Str)
	case orderedjson.Array:
		for i := 0; i < len(a.Arr) && i < len(b.Arr); i++ {
			if c := compare(a.Arr[i], b.Arr[i]); c != 0 {
				return c
			}
		}
		return len(a.Arr) - len(b.Arr)
	case orderedjson.Object:
		ka, kb := sortedKeys(a), sortedKeys(b)
		for i := 0; i < len(ka) && i < len(kb); i++ {
			if c := strings.Compare(ka[i], kb[i]); c != 0 {
				return c
			}
		}
		if len(ka) != len(kb) {
			return len(ka) - len(kb)
		}
		for _, k := range ka {
			if c := compare(a.Get(k), b.Get(k)); c != 0 {
				return c
			}
		}
	}
	return 0
}

func rank(v *orderedjson.Value) int {
	switch v.Kind {
	case orderedjson.Null:
		return 0
	case orderedjson.Bool:
		if v.Bool {
			return 2
		}
		return 1
	case orderedjson.Number:
		return 3
	case orderedjson.String:
		return 4
	case orderedjson.Array:
		return 5
	}
	return 6
}

func sortedKeys(v *orderedjson.Value) []string {
	keys := make([]string, 0, len(v.Obj))
	for _, m := range v.Obj {
		keys = append(keys, m.Key)
	}
	sort.Strings(keys)
	return keys
}
